package jira

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
)

type ErrorKind int

const (
	ErrorApplication ErrorKind = iota
	ErrorAuth
	ErrorNetwork
	ErrorCapability
)

type Error struct {
	Kind          ErrorKind
	StatusCode    int
	Status        string
	ErrorMessages []string
	Errors        map[string]string
	Body          string
	Err           error
}

func (e *Error) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	parts := make([]string, 0, 4)
	if e.Status != "" {
		parts = append(parts, e.Status)
	}
	if len(e.ErrorMessages) > 0 {
		parts = append(parts, strings.Join(e.ErrorMessages, "; "))
	}
	if len(e.Errors) > 0 {
		keys := make([]string, 0, len(e.Errors))
		for key := range e.Errors {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			parts = append(parts, fmt.Sprintf("%s: %s", key, e.Errors[key]))
		}
	}
	if len(parts) == 0 && e.Body != "" {
		parts = append(parts, strings.TrimSpace(e.Body))
	}
	if len(parts) == 0 {
		parts = append(parts, "jira error")
	}
	return strings.Join(parts, "; ")
}

func (e *Error) Unwrap() error {
	return e.Err
}

func (e *Error) ExitCode() int {
	switch e.Kind {
	case ErrorAuth:
		return 2
	case ErrorNetwork:
		return 4
	case ErrorCapability:
		return 5
	default:
		return 3
	}
}

func ParseErrorResponse(resp *http.Response, body []byte) *Error {
	kind := ErrorApplication
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		kind = ErrorAuth
	}

	parsed := struct {
		ErrorMessages []string          `json:"errorMessages"`
		Errors        map[string]string `json:"errors"`
	}{}
	_ = json.Unmarshal(body, &parsed)

	return &Error{
		Kind:          kind,
		StatusCode:    resp.StatusCode,
		Status:        resp.Status,
		ErrorMessages: parsed.ErrorMessages,
		Errors:        parsed.Errors,
		Body:          string(body),
	}
}

func AsCapabilityError(err error) error {
	var jiraErr *Error
	if !errors.As(err, &jiraErr) || jiraErr.StatusCode != http.StatusNotFound {
		return err
	}
	clone := *jiraErr
	clone.Kind = ErrorCapability
	return &clone
}

func ReadErrorResponse(resp *http.Response) *Error {
	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return &Error{Kind: ErrorNetwork, Err: readErr}
	}
	return ParseErrorResponse(resp, body)
}
