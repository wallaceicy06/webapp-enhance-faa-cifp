// App Engine web app that processes the FAA CIFP data for download.
package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"time"

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

func handlerWithTimeout(h http.Handler, d time.Duration) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), d)
		defer cancel()
		reqWithTimeout := r.WithContext(ctx)
		h.ServeHTTP(w, reqWithTimeout)
	})
}

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

	http.Handle("/", handlerWithTimeout(&index.Handler{
		Cycles: cyclesDb,
	}, 5*time.Second))
	http.Handle("/process", handlerWithTimeout(&process.Handler{
		ServiceAccountEmail: *serviceAccountEmail,
		Cycles:              cyclesDb,
		DisableAuth:         *disableAuth,
		Verifier:            auth.NewVerifier(),
	}, 60*time.Second))

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
