// App Engine web app that processes the FAA CIFP data for download.
package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/wallaceicy06/webapp-enhance-faa-cifp/handlers/index"
	"github.com/wallaceicy06/webapp-enhance-faa-cifp/handlers/process"
)

var (
	serviceAccountEmail = flag.String("service_account_email", "", "Service account email to verify when processing data.")
)

func main() {
	flag.Parse()

	if *serviceAccountEmail == "" {
		log.Fatal("Must provide a service account email.")
	}

	http.HandleFunc("/", index.Handle)
	http.HandleFunc("/process", process.New(*serviceAccountEmail).Handle)

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
