package process

import (
	"fmt"
	"net/http"
)

// Handle processes the latest CIFP data and saves it to Google Cloud Storage.
func Handle(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Processing...")
}
