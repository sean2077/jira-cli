package jira

import (
	"encoding/base64"
	"errors"
)

func BasicAuthHeader(user, secret string) (string, error) {
	if user == "" {
		return "", errors.New("user is required")
	}
	if secret == "" {
		return "", errors.New("secret is required")
	}
	encoded := base64.StdEncoding.EncodeToString([]byte(user + ":" + secret))
	return "Basic " + encoded, nil
}

// BearerAuthHeader builds an Authorization header for Jira Server/Data Center
// Personal Access Tokens (Jira 8.14+), which are presented as a bearer token
// rather than HTTP Basic.
func BearerAuthHeader(secret string) (string, error) {
	if secret == "" {
		return "", errors.New("secret is required")
	}
	return "Bearer " + secret, nil
}
