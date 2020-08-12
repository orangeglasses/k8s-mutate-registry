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
	AllowedDomains []string `json:"allowedDomains"`
	DefaultDomain  string   `json:"defaultDomain"`
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

	return Mutator{config: config}, nil

}

func (m Mutator) repoAllowed(image string) bool {
	allowed := false
	for _, allowedDomain := range m.config.AllowedDomains {
		allowed = strings.HasPrefix(image, allowedDomain)
		if allowed {
			break
		}
	}

	return allowed
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

func (m Mutator) mutate(r v1beta1.AdmissionReview) (v1beta1.AdmissionResponse, error) {
	patchType := v1beta1.PatchTypeJSONPatch
	response := v1beta1.AdmissionResponse{
		Allowed:   true,
		UID:       r.Request.UID,
		PatchType: &patchType,
		AuditAnnotations: map[string]string{
			"k8s-mutate-registry": "Container registry mutated by HaaS platform",
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
		log.Println("processing image: ", c.Image)

		//check if image is pulled from allowed registry
		if !m.repoAllowed(c.Image) {
			log.Printf("Image %v not allowed\n", c.Image)
			var newImage string
			//if not allowed check if this repo has fqdn or no domain
			if domain, hasDomain := m.hasDomain(c.Image); hasDomain {

				newImage = strings.ReplaceAll(c.Image, domain, m.config.DefaultDomain)
				log.Printf("Image has domain specified, replacing with default domain. New image name: %v\n", newImage)

			} else {

				newImage = fmt.Sprintf("%v/%v", m.config.DefaultDomain, c.Image)
				log.Printf("Image has NO domain specified, Adding with domain. New image name: %v\n", newImage)
			}

			patch := map[string]string{
				"op":    "replace",
				"path":  fmt.Sprintf("/spec/containers/%d/image", i),
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
