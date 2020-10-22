package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	v1beta1 "k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//MutatorConfig contains the config for the mutator
type MutatorConfig struct {
	DefaultDomain string            `json:"defaultDomain,omitempty"` // defaultDomain is supposed to be mapped as well. For example: defaultdomain could be registry-1.docker.io, the map resgirty-1.docker.io to wherever it need to go
	DomainMapping map[string]string `json:"domainMapping"`
}

//Mutator mutates
type Mutator struct {
	config MutatorConfig
}

func newMutator(configJSON string) (Mutator, error) {
	var config MutatorConfig
	err := json.Unmarshal([]byte(configJSON), &config)
	if err != nil {
		return Mutator{}, fmt.Errorf("error parsing configJSON: %v", err)
	}

	if config.DefaultDomain == "" {
		log.Printf("defaultDomain not configured, using registry-1.docker.io\n")
		config.DefaultDomain = "registry-1.docker.io"
	}

	if defaultMap, ok := config.DomainMapping[config.DefaultDomain]; !ok || defaultMap == "" {
		return Mutator{}, fmt.Errorf("Default domain not mapped. Please configure a mapping for %v", config.DefaultDomain)
	}

	return Mutator{config: config}, nil

}

func (m Mutator) hasDomain(image string) (string, bool) {
	//get domain part
	sSplit := strings.Split(image, "/")

	//if there is a dot in the host part we are going to assume this is an fqdn, if no dot present we assume default repo
	if strings.Contains(sSplit[0], ".") {
		return sSplit[0], true
	}

	return "", false
}

func (m Mutator) hasOrg(image string) bool {
	return strings.Contains(image, "/")
}

func (m Mutator) mutateImage(image string) (string, bool) {
	newImage := image
	mutated := false

	domain, hasDomain := m.hasDomain(image)
	if !hasDomain { //if no domain given we'll use the configured defaultDomain
		domain = m.config.DefaultDomain
		newImage = fmt.Sprintf("%v/%v", domain, image)
		log.Printf("Image %v has no domain, using default domain: %v", image, domain)
	}

	if !m.hasOrg(image) {
		var domainPlusOrg = fmt.Sprintf("%v/%v", domain, "library")
		newImage = strings.ReplaceAll(newImage, domain, domainPlusOrg)
	}

	//let's map the domain to the desired domain. Of we don't have a mapping we won't patch the json
	if mappedDomain, ok := m.config.DomainMapping[domain]; ok {
		newImage = strings.ReplaceAll(newImage, domain, mappedDomain)
		mutated = true
	}

	return newImage, mutated
}

func (m Mutator) mutate(r v1beta1.AdmissionReview) (v1beta1.AdmissionResponse, error) {
	patchType := v1beta1.PatchTypeJSONPatch
	response := v1beta1.AdmissionResponse{
		Allowed:   true,
		UID:       r.Request.UID,
		PatchType: &patchType,
		AuditAnnotations: map[string]string{
			"k8s-mutate-registry": "Container registry mutated by mutatingwebhook k8s-mutate-registry",
		},
	}

	if r.Request == nil {
		return response, nil
	}

	var pod corev1.Pod
	err := json.Unmarshal(r.Request.Object.Raw, &pod)
	if err != nil {
		return response, fmt.Errorf("Unable to unmarshall pod json: %v", err.Error())
	}

	patches := []map[string]string{}
	for i, c := range pod.Spec.Containers {
		newImage, mutated := m.mutateImage(c.Image)
		if mutated {
			patch := map[string]string{
				"op":    "replace",
				"path":  fmt.Sprintf("/spec/containers/%d/image", i),
				"value": newImage,
			}

			patches = append(patches, patch)
		}
	}

	for i, c := range pod.Spec.InitContainers {
		newImage, mutated := m.mutateImage(c.Image)
		if mutated {
			patch := map[string]string{
				"op":    "replace",
				"path":  fmt.Sprintf("/spec/initContainers/%d/image", i),
				"value": newImage,
			}

			patches = append(patches, patch)
		}
	}

	response.Patch, _ = json.Marshal(patches)
	response.Result = &metav1.Status{
		Status: "Success",
	}

	return response, nil
}
