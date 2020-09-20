package index

import (
	"context"
	"log"
	"net/http"

	"github.com/wallaceicy06/webapp-enhance-faa-cifp/db"
	"github.com/wallaceicy06/webapp-enhance-faa-cifp/templates"
)

type cyclesLister interface {
	List(context.Context) ([]*db.Cycle, error)
}

type Handler struct {
	Cycles cyclesLister
}

type baseValues struct {
	Cycles       []*db.Cycle
	DisplayError string
}

// Handle responds to requests with our greeting.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	bv := &baseValues{
		Cycles: []*db.Cycle{},
	}
	cycles, err := h.Cycles.List(r.Context())
	if err != nil {
		log.Printf("could not list cycles: %v", err)
		bv.DisplayError = "The enhanced FAA CIFP data U/S. We apologize for the inconvenience."
	} else {
		bv.Cycles = cycles
	}

	w.Header().Set("content-type", "text/html")
	if err := templates.Base.Execute(w, bv); err != nil {
		log.Printf("could not execute template: %v", err)
	}
}
