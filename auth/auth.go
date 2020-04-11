package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Verifier struct {
	url string
}

func NewVerifier() *Verifier {
	return &Verifier{
		url: "https://oauth2.googleapis.com/tokeninfo",
	}
}

type jwt struct {
	Audience      string `json:"aud"`
	Email         string `json:"email"`
	EmailVerified string `json:"email_verified"`
	Expires       string `json:"exp"`
}

// ParseAuthHeader returns the bearer token of the provided authorization
// header. If the authorization header is invalid, then "" is returned.
func ParseAuthHeader(header string) string {
	if !strings.HasPrefix(header, "Bearer ") {
		return ""
	}
	return strings.TrimPrefix(header, "Bearer ")
}

func (v *Verifier) VerifyGoogle(ctx context.Context, token string) (string, error) {
	res, err := http.Get(v.url + "?id_token=" + token)
	if err != nil {
		return "", fmt.Errorf("problem getting token info: %v", err)
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", fmt.Errorf("could not read body: %v", err)
	}
	defer res.Body.Close()

	var j jwt
	if err := json.Unmarshal(body, &j); err != nil {
		return "", fmt.Errorf("could not unmarshal response: %v", err)
	}

	if j.EmailVerified != "true" {
		return "", fmt.Errorf("invalid email: %q", j.Email)
	}
	expireSeconds, err := strconv.ParseInt(j.Expires, 10, 64)
	if err != nil {
		return "", fmt.Errorf("invalid expiry time format: %v", err)
	}
	expireTime := time.Unix(expireSeconds, 0)
	if time.Now().After(expireTime) {
		return "", fmt.Errorf("credentials expired: %v", err)
	}
	return j.Email, nil
}
