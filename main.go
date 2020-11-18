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
	"cloud.google.com/go/storage"
	"github.com/wallaceicy06/webapp-enhance-faa-cifp/auth"
	"github.com/wallaceicy06/webapp-enhance-faa-cifp/blob"
	"github.com/wallaceicy06/webapp-enhance-faa-cifp/db"
	"github.com/wallaceicy06/webapp-enhance-faa-cifp/handlers/index"
	"github.com/wallaceicy06/webapp-enhance-faa-cifp/handlers/process"
)

var (
	serviceAccountEmail = flag.String("service_account_email", os.Getenv("SERVICE_ACCOUNT"), "Service account email to verify when processing data.")
	projectID           = flag.String("project_id", os.Getenv("PROJECT_ID"), "Project ID that contains the Firestore database.")
	gcsBucket           = flag.String("gcs_bucket", "faa-cifp-data", "The Google Cloud Storage bucket to write data to.")
	disableAuth         = flag.Bool("noauth", false, "Disable authentication for testng purposes.")
	port                = flag.String("port", os.Getenv("PORT"), "port to start server on")
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
	gcsClient, err := storage.NewClient(ctx)
	if err != nil {
		log.Fatalf("Could not create Google Cloud Storage client: %v", err)
	}

	cyclesDb := &db.Cycles{
		Client: fsClient,
	}

	http.Handle("/", handlerWithTimeout(&index.Handler{
		BucketName: *gcsBucket,
		Cycles:     cyclesDb,
	}, 5*time.Second))
	http.Handle("/process", handlerWithTimeout(&process.Handler{
		ServiceAccountEmail: *serviceAccountEmail,
		Cycles:              cyclesDb,
		DisableAuth:         *disableAuth,
		Verifier:            auth.NewVerifier(),
		CifpURL:             "https://soa.smext.faa.gov/apra/cifp/chart?edition=next",
		StorageClient:       &blob.GCSClient{Client: gcsClient, BucketName: *gcsBucket},
	}, 120*time.Second))

	if *port == "" {
		*port = "8080"
		log.Printf("Defaulting to port %s", *port)
	}

	log.Printf("Listening on port %s", *port)
	if err := http.ListenAndServe(":"+*port, nil); err != nil {
		log.Fatal(err)
	}
}
