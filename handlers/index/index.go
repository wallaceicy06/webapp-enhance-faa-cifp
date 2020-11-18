package index

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/wallaceicy06/webapp-enhance-faa-cifp/db"
	"github.com/wallaceicy06/webapp-enhance-faa-cifp/templates"
)

type cyclesLister interface {
	List(context.Context) ([]*db.Cycle, error)
}

type Handler struct {
	BucketName string
	Cycles     cyclesLister
}

type baseValues struct {
	BucketName   string
	Cycles       []*db.Cycle
	DisplayError string
}

func (bv *baseValues) URLFor(name string) string {
	return fmt.Sprintf("https://storage.googleapis.com/%s/%s", bv.BucketName, name)
}

// Handle responds to requests with our greeting.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	bv := &baseValues{
		BucketName: h.BucketName,
		Cycles:     []*db.Cycle{},
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
