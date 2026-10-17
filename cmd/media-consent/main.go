package main

import (
	"log"
	"net/http"
	"os"

	mediaconsent "example.com/media-consent-pipeline"
)

func main() {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		log.Fatal("INFRAI_API_KEY is required")
	}
	address := os.Getenv("LISTEN_ADDR")
	if address == "" {
		address = ":8080"
	}
	service := mediaconsent.Service{Consent: mediaconsent.NewConsentClient(key)}
	log.Printf("media consent service listening on %s", address)
	log.Fatal(http.ListenAndServe(address, service.Handler()))
}
