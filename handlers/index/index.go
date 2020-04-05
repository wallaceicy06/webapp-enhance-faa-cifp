package index

import (
	"log"
	"net/http"

	"github.com/wallaceicy06/webapp-enhance-faa-cifp/templates"
)

// Handle responds to requests with our greeting.
func Handle(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("content-type", "text/html")
	if err := templates.Base.Execute(w, nil); err != nil {
		log.Printf("could not execute template: %v", err)
	}
}
