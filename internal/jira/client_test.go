package jira

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

var errMultipartRead = errors.New("multipart reader failed")
var errDownloadWrite = errors.New("download writer failed")

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) {
	return 0, errMultipartRead
}

type failingWriter struct {
	limit int
	seen  int
}

func (w *failingWriter) Write(p []byte) (int, error) {
	remaining := w.limit - w.seen
	if remaining <= 0 {
		return 0, errDownloadWrite
	}
	if len(p) > remaining {
		w.seen += remaining
		return remaining, errDownloadWrite
	}
	w.seen += len(p)
	return len(p), nil
}

func TestPostMultipartFileReturnsReaderError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[]`))
	}))
	defer server.Close()

	client := Client{BaseURL: server.URL, HTTPClient: server.Client(), Timeout: time.Second}
	_, err := client.PostMultipartFile(context.Background(), PlatformAPI, []string{"issue", "JCLI-1", "attachments"}, "file", "proof.txt", failingReader{}, nil)
	if !errors.Is(err, errMultipartRead) {
		t.Fatalf("PostMultipartFile error = %v, want %v", err, errMultipartRead)
	}
}

func TestPostMultipartFileAuthPreflightBeforeRequest(t *testing.T) {
	var called bool
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called = true
	}))
	defer server.Close()

	client := Client{BaseURL: server.URL, Secret: "secret", HTTPClient: server.Client(), Timeout: time.Second}
	_, err := client.PostMultipartFile(context.Background(), PlatformAPI, []string{"issue", "JCLI-1", "attachments"}, "file", "proof.txt", strings.NewReader("proof"), nil)
	if err == nil || !strings.Contains(err.Error(), "user is required") {
		t.Fatalf("PostMultipartFile error = %v, want missing user", err)
	}
	if called {
		t.Fatal("auth preflight should fail before sending an HTTP request")
	}
}

func TestDownloadURLReturnsWriterErrorAndPartialBytes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("proof"))
	}))
	defer server.Close()

	client := Client{HTTPClient: server.Client(), Timeout: time.Second}
	writer := &failingWriter{limit: 2}
	_, n, err := client.DownloadURL(context.Background(), server.URL+"/secure/attachment/700/proof.txt", writer)
	if !errors.Is(err, errDownloadWrite) {
		t.Fatalf("DownloadURL error = %v, want %v", err, errDownloadWrite)
	}
	if n != 2 {
		t.Fatalf("DownloadURL bytes = %d, want 2", n)
	}
}

func TestDownloadURLParsesNonSuccessJiraError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"errorMessages":["no attachment permission"]}`))
	}))
	defer server.Close()

	client := Client{HTTPClient: server.Client(), Timeout: time.Second}
	resp, n, err := client.DownloadURL(context.Background(), server.URL+"/secure/attachment/700/proof.txt", io.Discard)
	jiraErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("DownloadURL error = %T %v, want *Error", err, err)
	}
	if resp.StatusCode != http.StatusForbidden || n != 0 || jiraErr.Kind != ErrorAuth || !strings.Contains(jiraErr.Error(), "no attachment permission") {
		t.Fatalf("resp=%#v n=%d err=%#v", resp, n, jiraErr)
	}
}
