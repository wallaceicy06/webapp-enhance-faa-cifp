// App Engine web app that processes the FAA CIFP data for download.
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	http.HandleFunc("/", indexHandler)

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

// indexHandler responds to requests with our greeting.
func indexHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	fmt.Fprintln(w, "Welcome to the FAA CIFP data enhancer.")
	fmt.Fprintln(w, "This app is not associated with the Federal Aviation Administration and has no warranty.")
	fmt.Fprintln(w, "See http://seanharger.com/posts/hundredths-of-degrees-from-death for more information.")
}
