// App Engine web app that processes the FAA CIFP data for download.
package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"

	"cloud.google.com/go/firestore"
	"github.com/wallaceicy06/webapp-enhance-faa-cifp/auth"
	"github.com/wallaceicy06/webapp-enhance-faa-cifp/db"
	"github.com/wallaceicy06/webapp-enhance-faa-cifp/handlers/index"
	"github.com/wallaceicy06/webapp-enhance-faa-cifp/handlers/process"
)

var (
	serviceAccountEmail = flag.String("service_account_email", os.Getenv("SERVICE_ACCOUNT"), "Service account email to verify when processing data.")
	projectID           = flag.String("project_id", os.Getenv("PROJECT_ID"), "Project ID that contains the Firestore database.")
	disableAuth         = flag.Bool("noauth", false, "Disable authentication for testng purposes.")
)

func main() {
	ctx := context.Background()
	flag.Parse()

	if *serviceAccountEmail == "" {
		log.Fatal("Must provide a service account email.")
	}
	if *projectID == "" {
		log.Fatal("Must provide a project ID.")
	}

	fsClient, err := firestore.NewClient(ctx, *projectID)
	if err != nil {
		log.Fatalf("Could not create firestore client: %v", err)
	}
	cyclesDb := &db.Cycles{
		Client: fsClient,
	}

	http.HandleFunc("/", index.Handle)
	http.HandleFunc("/process", (&process.Handler{
		ServiceAccountEmail: *serviceAccountEmail,
		CyclesDb:            cyclesDb,
		DisableAuth:         *disableAuth,
		Verifier:            auth.NewVerifier(),
	}).Handle)

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
