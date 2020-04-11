package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

func TestParseAuthHeader(t *testing.T) {
	for _, tt := range []struct {
		name   string
		header string
		want   string
	}{
		{
			name:   "Good",
			header: "Bearer token",
			want:   "token",
		},
		{
			name:   "Invalid",
			header: "token",
			want:   "",
		},
		{
			name:   "Empty",
			header: "",
			want:   "",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParseAuthHeader(tt.header); got != tt.want {
				t.Errorf("ParseAuthHeader(%q) = %q want %q", tt.header, got, tt.want)
			}
		})
	}
}

func TestVerifyGoogle(t *testing.T) {
	for _, tt := range []struct {
		name     string
		response *jwt
		want     string
		wantErr  bool
	}{
		{
			name: "Good",
			response: &jwt{
				Email:         "some-email@example.com",
				EmailVerified: "true",
				Expires:       strconv.FormatInt(time.Now().Add(time.Hour).Unix(), 10),
			},
			want: "some-email@example.com",
		},
		{
			name: "UnverifiedEmail",
			response: &jwt{
				Email:         "some-email@example.com",
				EmailVerified: "false",
				Expires:       strconv.FormatInt(time.Now().Add(time.Hour).Unix(), 10),
			},
			wantErr: true,
		},
		{
			name: "UnexpectedEmail",
			response: &jwt{
				Email:         "some-email@evil.com",
				EmailVerified: "",
				Expires:       strconv.FormatInt(time.Now().Add(time.Hour).Unix(), 10),
			},
			wantErr: true,
		},
		{
			name: "ExpiredToken",
			response: &jwt{
				Email:         "some-email@example.com",
				EmailVerified: "true",
				Expires:       strconv.FormatInt(time.Now().Add(-time.Hour).Unix(), 10),
			},
			wantErr: true,
		},
		{
			name: "InvalidExpireTime",
			response: &jwt{
				Email:         "some-email@example.com",
				EmailVerified: "true",
				Expires:       "-123abcd",
			},
			wantErr: true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			mux.Handle("/tokeninfo", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				b, err := json.Marshal(tt.response)
				if err != nil {
					http.Error(w, fmt.Sprintf("Problem marshalling JSON: %v", err), http.StatusInternalServerError)
				}
				if _, err := w.Write(b); err != nil {
					http.Error(w, fmt.Sprintf("Problem writing response: %v", err), http.StatusInternalServerError)
				}
			}))

			srv := httptest.NewServer(mux)
			defer srv.Close()

			v := &Verifier{url: srv.URL + "/tokeninfo"}
			token := "some-token"
			got, err := v.VerifyGoogle(context.Background(), token)
			if tt.wantErr {
				if err == nil {
					t.Errorf("VerifyGoogle(_, %q) = _, <nil> want _, <non-nil>", token)
				}
				return
			}
			if err != nil {
				t.Errorf("VerifyGoogle(_, %q) = _, %v want _, <nil>", token, err)
			}
			if got != tt.want {
				t.Errorf("VerifyGoogle(_, %q) = %q, _ want %q, _", token, got, tt.want)
			}
		})
	}
}
