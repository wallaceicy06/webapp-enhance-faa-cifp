package process

import (
	"context"
	"log"
	"net/http"

	"github.com/wallaceicy06/webapp-enhance-faa-cifp/auth"
)

type googleVerifier interface {
	VerifyGoogle(context.Context, string) (string, error)
}

type Handler struct {
	serviceAccountEmail string
	verifier            googleVerifier
}

func New(serviceAccountEmail string) *Handler {
	return &Handler{
		serviceAccountEmail: serviceAccountEmail,
		verifier:            auth.NewVerifier(),
	}
}

// Handle processes the latest CIFP data and saves it to Google Cloud Storage.
func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) {
	a := r.Header.Get("Authorization")
	if a == "" {
		http.Error(w, "Must provide credentials.", http.StatusUnauthorized)
		return
	}
	email, err := h.verifier.VerifyGoogle(r.Context(), auth.ParseAuthHeader(a))
	if err != nil {
		http.Error(w, "Invalid credentials.", http.StatusForbidden)
		return
	}
	log.Printf("got user with email: %q", email)
	if email != h.serviceAccountEmail {
		http.Error(w, "Invalid credentials.", http.StatusForbidden)
		return
	}
}
