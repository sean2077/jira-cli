package jira

import (
	"errors"
	"fmt"
	"net/url"
	"path"
	"strings"
)

type API string

const (
	PlatformAPI API = "platform"
	AgileAPI    API = "agile"
)

func BuildURL(baseURL string, api API, segments ...string) (string, error) {
	if baseURL == "" {
		return "", errors.New("base URL is required")
	}
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", errors.New("base URL must include scheme and host")
	}

	apiSegments, err := apiPathSegments(api)
	if err != nil {
		return "", err
	}
	if err := validatePathSegments(segments); err != nil {
		return "", err
	}
	parts := append([]string{parsed.Path}, apiSegments...)
	parts = append(parts, segments...)
	parsed.Path = path.Join(parts...)
	if parsed.Path == "." {
		parsed.Path = "/"
	}
	if parsed.Path[0] != '/' {
		parsed.Path = "/" + parsed.Path
	}
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String(), nil
}

func validatePathSegments(segments []string) error {
	for _, segment := range segments {
		if segment == "" || segment == "." || segment == ".." || strings.ContainsAny(segment, `/\`) {
			return fmt.Errorf("unsafe URL path segment %q", segment)
		}
	}
	return nil
}

func AddQuery(rawURL string, values url.Values) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	parsed.RawQuery = values.Encode()
	return parsed.String(), nil
}

func apiPathSegments(api API) ([]string, error) {
	switch api {
	case PlatformAPI:
		return []string{"rest", "api", "2"}, nil
	case AgileAPI:
		return []string{"rest", "agile", "1.0"}, nil
	default:
		return nil, errors.New("unknown Jira API")
	}
}
