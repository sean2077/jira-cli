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
