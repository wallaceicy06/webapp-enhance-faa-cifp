package process

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakeVerifier struct {
	GotToken string
	Email    string
	Err      error
}

func (fv *fakeVerifier) VerifyGoogle(_ context.Context, token string) (string, error) {
	fv.GotToken = token
	return fv.Email, fv.Err
}

func TestHandle(t *testing.T) {
	const serviceAccountEmail = "some-email@example.com"

	for _, tt := range []struct {
		name         string
		fakeVerifier *fakeVerifier
		authHeader   string
		wantStatus   int
	}{
		{
			name: "Good",
			fakeVerifier: &fakeVerifier{
				Email: "some-email@example.com",
			},
			authHeader: "Bearer token",
			wantStatus: http.StatusOK,
		},
		{
			name:       "NoAuthorization",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "BadEmail",
			fakeVerifier: &fakeVerifier{
				Email: "some-email@evil.com",
			},
			authHeader: "Bearer token",
			wantStatus: http.StatusForbidden,
		},
		{
			name: "VerificationError",
			fakeVerifier: &fakeVerifier{
				Err: errors.New("problem verifying token"),
			},
			authHeader: "Bearer token",
			wantStatus: http.StatusForbidden,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			handler := http.HandlerFunc((&Handler{
				ServiceAccountEmail: serviceAccountEmail,
				Verifier:            tt.fakeVerifier,
			}).Handle)
			req := httptest.NewRequest(http.MethodPost, "/", &bytes.Buffer{})
			req.Header.Set("Authorization", tt.authHeader)
			handler.ServeHTTP(rr, req)
			if status := rr.Code; status != tt.wantStatus {
				t.Errorf("handler returned wrong status code: got %d want %d",
					status, tt.wantStatus)
			}
			if tt.wantStatus == http.StatusOK && tt.fakeVerifier.GotToken != "token" {
				t.Errorf(`verifier received token %q want "token"`, tt.fakeVerifier.GotToken)
			}
		})
	}
}
