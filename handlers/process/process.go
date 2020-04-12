package process

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/wallaceicy06/webapp-enhance-faa-cifp/auth"
	"github.com/wallaceicy06/webapp-enhance-faa-cifp/db"
)

type googleVerifier interface {
	VerifyGoogle(context.Context, string) (string, error)
}

type cyclesAdder interface {
	Add(context.Context, *db.Cycle) error
}

type Handler struct {
	ServiceAccountEmail string
	Verifier            googleVerifier
	Cycles              cyclesAdder
	DisableAuth         bool
}

// Handle processes the latest CIFP data and saves it to Google Cloud Storage.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !h.DisableAuth {
		a := r.Header.Get("Authorization")
		if a == "" {
			http.Error(w, "Must provide credentials.", http.StatusUnauthorized)
			return
		}
		email, err := h.Verifier.VerifyGoogle(r.Context(), auth.ParseAuthHeader(a))
		if err != nil {
			http.Error(w, "Invalid credentials.", http.StatusForbidden)
			return
		}
		log.Printf("got user with email: %q", email)
		if email != h.ServiceAccountEmail {
			http.Error(w, "Invalid credentials.", http.StatusForbidden)
			return
		}
	}

	if err := h.Cycles.Add(r.Context(), &db.Cycle{Name: time.Now().String(), ProcessedURL: "http://www.google.com"}); err != nil {
		log.Printf("Could not add cycle: %v", err)
		http.Error(w, "Could not add cycle.", http.StatusInternalServerError)
		return
	}
}
