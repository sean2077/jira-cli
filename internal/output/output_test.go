package output

import (
	"bytes"
	"testing"
)

func TestWriteVersion(t *testing.T) {
	var compact bytes.Buffer
	if err := WriteVersion(&compact, Compact, "v"); err != nil {
		t.Fatalf("WriteVersion compact: %v", err)
	}
	if got, want := compact.String(), "jira v\n"; got != want {
		t.Fatalf("compact = %q, want %q", got, want)
	}

	var json bytes.Buffer
	if err := WriteVersion(&json, JSON, "v"); err != nil {
		t.Fatalf("WriteVersion JSON: %v", err)
	}
	if got, want := json.String(), "{\"ok\":true,\"kind\":\"version\",\"version\":\"v\"}\n"; got != want {
		t.Fatalf("json = %q, want %q", got, want)
	}
}

func TestWriteRawIsUnmodified(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteRaw(&buf, []byte("{\"jira\":true}")); err != nil {
		t.Fatalf("WriteRaw: %v", err)
	}
	if got, want := buf.String(), "{\"jira\":true}"; got != want {
		t.Fatalf("raw = %q, want %q", got, want)
	}
}

func TestWriteCompactLines(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteCompact(&buf, "JCLI-123 Open P2 jdoe Fix login redirect", `1 issues total=18 next="--start-at 1"`); err != nil {
		t.Fatalf("WriteCompact: %v", err)
	}
	want := "JCLI-123 Open P2 jdoe Fix login redirect\n1 issues total=18 next=\"--start-at 1\"\n"
	if got := buf.String(); got != want {
		t.Fatalf("compact = %q, want %q", got, want)
	}
}
