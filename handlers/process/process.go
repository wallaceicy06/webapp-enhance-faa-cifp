package process

import (
	"context"
	"log"
	"net/http"

	"github.com/wallaceicy06/webapp-enhance-faa-cifp/auth"
	"github.com/wallaceicy06/webapp-enhance-faa-cifp/db"
)

type googleVerifier interface {
	VerifyGoogle(context.Context, string) (string, error)
}

type Handler struct {
	ServiceAccountEmail string
	Verifier            googleVerifier
	CyclesDb            *db.Cycles
	DisableAuth         bool
}

// Handle processes the latest CIFP data and saves it to Google Cloud Storage.
func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) {
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
}
