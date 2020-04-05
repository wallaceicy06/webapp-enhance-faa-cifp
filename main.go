// App Engine web app that processes the FAA CIFP data for download.
package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/wallaceicy06/webapp-enhance-faa-cifp/auth"
	"github.com/wallaceicy06/webapp-enhance-faa-cifp/handlers/index"
	"github.com/wallaceicy06/webapp-enhance-faa-cifp/handlers/process"
)

var (
	clientID     = flag.String("oauth_client_id", "", "OAuth client ID for this app.")
	clientSecret = flag.String("oauth_client_secret", "", "OAuth client secret for this app.")
)

func main() {
	flag.Parse()

	oidcClient, err := auth.NewOIDC(context.Background(), *clientID, *clientSecret)
	if err != nil {
		log.Fatalf("Could not create OIDC auth client: %v", err)
	}

	http.HandleFunc("/", index.Handle)
	http.HandleFunc("/process", process.New(oidcClient).Handle)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		log.Printf("Defaulting to port %s", port)
	}

	log.Printf("Listening on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
