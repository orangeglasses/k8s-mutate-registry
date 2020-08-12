package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	v1beta1 "k8s.io/api/admission/v1beta1"
)

type mutateReqHandler struct {
	mutator Mutator
}

func (h mutateReqHandler) handle(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	review := v1beta1.AdmissionReview{}
	err := json.NewDecoder(r.Body).Decode(&review)
	if err != nil {
		log.Printf("Unmarshaling request failed: %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	response, err := h.mutator.mutate(review)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("Error mutating review: %v", err)
		//json.NewEncoder(w).Encode(response)
		return
	}

	review.Response = &response

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(review)
}

func main() {
	mutator, err := newMutator(os.Getenv("MUTATE_CONFIG"))
	if err != nil {
		log.Fatalf("Could not load config. Please set MUTATE_CONFIG env variable. Error: %v\n", err)
	}
	mutateHandler := mutateReqHandler{
		mutator: mutator,
	}

	r := mux.NewRouter()
	s := &http.Server{
		Addr:           fmt.Sprintf(":%v", os.Getenv("PORT")),
		Handler:        r,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1048576
	}

	log.Println("Starting http server")
	r.Path("/mutate").Methods(http.MethodPost).HandlerFunc(mutateHandler.handle)
	log.Fatal(s.ListenAndServeTLS("ssl/tls.crt", "ssl/tls.key"))
}
