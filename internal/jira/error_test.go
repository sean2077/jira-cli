package jira

import (
	"net/http"
	"strings"
	"testing"
)

func TestParseErrorResponseMapsAuthAndMessages(t *testing.T) {
	resp := &http.Response{StatusCode: http.StatusForbidden, Status: "403 Forbidden"}
	err := ParseErrorResponse(resp, []byte(`{"errorMessages":["no permission"],"errors":{"assignee":"unknown user"}}`))
	if err.Kind != ErrorAuth {
		t.Fatalf("Kind = %v, want ErrorAuth", err.Kind)
	}
	if err.ExitCode() != 2 {
		t.Fatalf("ExitCode = %d, want 2", err.ExitCode())
	}
	text := err.Error()
	for _, want := range []string{"403 Forbidden", "no permission", "assignee: unknown user"} {
		if !strings.Contains(text, want) {
			t.Fatalf("Error() missing %q in %q", want, text)
		}
	}
}

func TestParseErrorResponseMaps404ToApplicationByDefault(t *testing.T) {
	resp := &http.Response{StatusCode: http.StatusNotFound, Status: "404 Not Found"}
	err := ParseErrorResponse(resp, []byte(`{"errorMessages":["Not Found"]}`))
	if err.Kind != ErrorApplication {
		t.Fatalf("Kind = %v, want ErrorApplication", err.Kind)
	}
	if err.ExitCode() != 3 {
		t.Fatalf("ExitCode = %d, want 3", err.ExitCode())
	}

	capabilityErr := AsCapabilityError(err)
	if capabilityErr.(*Error).Kind != ErrorCapability {
		t.Fatalf("AsCapabilityError kind = %v, want ErrorCapability", capabilityErr.(*Error).Kind)
	}
	if capabilityErr.(*Error).ExitCode() != 5 {
		t.Fatalf("AsCapabilityError exit = %d, want 5", capabilityErr.(*Error).ExitCode())
	}
}
