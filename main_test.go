package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHandlerWithTimeout(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-time.After(10 * time.Second):
			http.Error(w, "Request took too long", http.StatusRequestTimeout)
		case <-r.Context().Done():
			return
		}
	})
	withTimeout := handlerWithTimeout(handler, 1*time.Second)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", &bytes.Buffer{})
	withTimeout.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned status %d want 200", status)
	}
}
