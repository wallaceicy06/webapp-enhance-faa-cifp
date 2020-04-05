package process

import (
	"fmt"
	"net/http"

	"github.com/wallaceicy06/webapp-enhance-faa-cifp/auth"
)

type Handler struct {
	oidcClient *auth.OIDC
}

func New(oidcClient *auth.OIDC) *Handler {
	return &Handler{
		oidcClient: oidcClient,
	}
}

// Handle processes the latest CIFP data and saves it to Google Cloud Storage.
func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Processing. Got user %q", r.URL.Query().Get("code"))
}
