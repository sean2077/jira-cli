package jira

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type Client struct {
	BaseURL    string
	User       string
	Secret     string
	HTTPClient *http.Client
	Timeout    time.Duration
}

type Response struct {
	StatusCode int
	Body       []byte
}

func (c Client) Get(ctx context.Context, api API, segments []string, query url.Values, out any) (Response, error) {
	return c.Do(ctx, http.MethodGet, api, segments, query, nil, out)
}

func (c Client) Do(ctx context.Context, method string, api API, segments []string, query url.Values, requestBody any, out any) (Response, error) {
	rawURL, err := BuildURL(c.BaseURL, api, segments...)
	if err != nil {
		return Response{}, err
	}
	if len(query) > 0 {
		rawURL, err = AddQuery(rawURL, query)
		if err != nil {
			return Response{}, err
		}
	}

	var body io.Reader
	if requestBody != nil {
		encoded, err := json.Marshal(requestBody)
		if err != nil {
			return Response{}, err
		}
		body = bytes.NewReader(encoded)
	}

	timeout := c.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, method, rawURL, body)
	if err != nil {
		return Response{}, err
	}
	req.Header.Set("Accept", "application/json")
	if requestBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.User != "" || c.Secret != "" {
		auth, err := BasicAuthHeader(c.User, c.Secret)
		if err != nil {
			return Response{}, err
		}
		req.Header.Set("Authorization", auth)
	}

	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return Response{}, &Error{Kind: ErrorNetwork, Err: err}
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return Response{}, &Error{Kind: ErrorNetwork, Err: err}
	}
	response := Response{StatusCode: resp.StatusCode, Body: raw}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return response, ParseErrorResponse(resp, raw)
	}
	if out != nil && len(raw) > 0 {
		if err := json.Unmarshal(raw, out); err != nil {
			return response, fmt.Errorf("decode Jira response: %w", err)
		}
	}
	return response, nil
}

func (c Client) PostMultipartFile(ctx context.Context, api API, segments []string, fieldName, fileName string, file io.Reader, out any) (Response, error) {
	rawURL, err := BuildURL(c.BaseURL, api, segments...)
	if err != nil {
		return Response{}, err
	}

	auth := ""
	if c.User != "" || c.Secret != "" {
		auth, err = BasicAuthHeader(c.User, c.Secret)
		if err != nil {
			return Response{}, err
		}
	}

	timeout := c.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	bodyReader, bodyWriter := io.Pipe()
	writer := multipart.NewWriter(bodyWriter)
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, rawURL, bodyReader)
	if err != nil {
		_ = bodyReader.Close()
		_ = bodyWriter.Close()
		return Response{}, err
	}
	defer bodyReader.Close()
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-Atlassian-Token", "no-check")
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}

	writeErr := make(chan error, 1)
	go func() {
		defer close(writeErr)
		part, err := writer.CreateFormFile(fieldName, fileName)
		if err == nil {
			_, err = io.Copy(part, file)
		}
		if closeErr := writer.Close(); err == nil {
			err = closeErr
		}
		if err != nil {
			_ = bodyWriter.CloseWithError(err)
			writeErr <- err
			return
		}
		writeErr <- bodyWriter.Close()
	}()

	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		_ = bodyReader.Close()
		if pipeErr := waitMultipartWrite(reqCtx, writeErr); pipeErr != nil {
			return Response{}, pipeErr
		}
		return Response{}, &Error{Kind: ErrorNetwork, Err: err}
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return Response{}, &Error{Kind: ErrorNetwork, Err: err}
	}
	response := Response{StatusCode: resp.StatusCode, Body: raw}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return response, ParseErrorResponse(resp, raw)
	}
	if pipeErr := waitMultipartWrite(reqCtx, writeErr); pipeErr != nil {
		return response, pipeErr
	}
	if out != nil && len(raw) > 0 {
		if err := json.Unmarshal(raw, out); err != nil {
			return response, fmt.Errorf("decode Jira response: %w", err)
		}
	}
	return response, nil
}

func waitMultipartWrite(ctx context.Context, writeErr <-chan error) error {
	select {
	case err := <-writeErr:
		return err
	case <-ctx.Done():
		select {
		case err := <-writeErr:
			return err
		default:
			return ctx.Err()
		}
	}
}

func (c Client) DownloadURL(ctx context.Context, rawURL string, out io.Writer) (Response, int64, error) {
	timeout := c.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, rawURL, nil)
	if err != nil {
		return Response{}, 0, err
	}
	req.Header.Set("Accept", "*/*")
	if c.User != "" || c.Secret != "" {
		auth, err := BasicAuthHeader(c.User, c.Secret)
		if err != nil {
			return Response{}, 0, err
		}
		req.Header.Set("Authorization", auth)
	}

	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return Response{}, 0, &Error{Kind: ErrorNetwork, Err: err}
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		raw, err := io.ReadAll(resp.Body)
		if err != nil {
			return Response{}, 0, &Error{Kind: ErrorNetwork, Err: err}
		}
		response := Response{StatusCode: resp.StatusCode, Body: raw}
		return response, 0, ParseErrorResponse(resp, raw)
	}
	n, err := io.Copy(out, resp.Body)
	if err != nil {
		return Response{}, n, &Error{Kind: ErrorNetwork, Err: err}
	}
	return Response{StatusCode: resp.StatusCode}, n, nil
}

type ServerInfo struct {
	Version string `json:"version"`
}

type User struct {
	Name         string `json:"name"`
	Key          string `json:"key"`
	DisplayName  string `json:"displayName"`
	EmailAddress string `json:"emailAddress"`
}

type NamedValue struct {
	ID          string `json:"id,omitempty"`
	Key         string `json:"key,omitempty"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
}

type Project struct {
	ID   string `json:"id,omitempty"`
	Key  string `json:"key,omitempty"`
	Name string `json:"name,omitempty"`
	Lead *User  `json:"lead,omitempty"`
}

type Field struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Custom bool   `json:"custom"`
}

type Issue struct {
	ID     string      `json:"id,omitempty"`
	Key    string      `json:"key"`
	Fields IssueFields `json:"fields"`
}

type IssueFields struct {
	Summary     string       `json:"summary,omitempty"`
	Status      *NamedValue  `json:"status,omitempty"`
	Priority    *NamedValue  `json:"priority,omitempty"`
	Assignee    *User        `json:"assignee,omitempty"`
	Project     *Project     `json:"project,omitempty"`
	IssueType   *NamedValue  `json:"issuetype,omitempty"`
	Created     string       `json:"created,omitempty"`
	Updated     string       `json:"updated,omitempty"`
	Description string       `json:"description,omitempty"`
	IssueLinks  []IssueLink  `json:"issuelinks,omitempty"`
	Attachments []Attachment `json:"attachment,omitempty"`
}

type IssueLinkType struct {
	ID      string `json:"id,omitempty"`
	Name    string `json:"name,omitempty"`
	Inward  string `json:"inward,omitempty"`
	Outward string `json:"outward,omitempty"`
}

type IssueLink struct {
	ID           string         `json:"id,omitempty"`
	Type         *IssueLinkType `json:"type,omitempty"`
	InwardIssue  *Issue         `json:"inwardIssue,omitempty"`
	OutwardIssue *Issue         `json:"outwardIssue,omitempty"`
}

type SearchResult struct {
	StartAt    int     `json:"startAt"`
	MaxResults int     `json:"maxResults"`
	Total      int     `json:"total"`
	Issues     []Issue `json:"issues"`
}

type AgilePage[T any] struct {
	StartAt    int  `json:"startAt"`
	MaxResults int  `json:"maxResults"`
	Total      int  `json:"total"`
	IsLast     bool `json:"isLast,omitempty"`
	Values     []T  `json:"values"`
}

type Board struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Type string `json:"type,omitempty"`
}

type Sprint struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	State string `json:"state,omitempty"`
}

type Epic struct {
	ID      int    `json:"id,omitempty"`
	Key     string `json:"key,omitempty"`
	Name    string `json:"name,omitempty"`
	Summary string `json:"summary,omitempty"`
	Done    bool   `json:"done,omitempty"`
}

type Dashboard struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Self string `json:"self,omitempty"`
	View string `json:"view,omitempty"`
}

type DashboardsResult struct {
	StartAt    int         `json:"startAt"`
	MaxResults int         `json:"maxResults"`
	Total      int         `json:"total"`
	Prev       string      `json:"prev,omitempty"`
	Next       string      `json:"next,omitempty"`
	Dashboards []Dashboard `json:"dashboards"`
}

type EntityPropertyKey struct {
	Self string `json:"self,omitempty"`
	Key  string `json:"key"`
}

type EntityPropertyKeys struct {
	Keys []EntityPropertyKey `json:"keys"`
}

type EntityProperty struct {
	Key   string          `json:"key"`
	Value json.RawMessage `json:"value"`
}

type Transition struct {
	ID   string      `json:"id"`
	Name string      `json:"name"`
	To   *NamedValue `json:"to,omitempty"`
}

type TransitionsResult struct {
	Transitions []Transition `json:"transitions"`
}

type Watchers struct {
	IsWatching bool   `json:"isWatching"`
	WatchCount int    `json:"watchCount"`
	Watchers   []User `json:"watchers"`
}

type Comment struct {
	ID           string `json:"id,omitempty"`
	Body         string `json:"body,omitempty"`
	Author       *User  `json:"author,omitempty"`
	UpdateAuthor *User  `json:"updateAuthor,omitempty"`
	Created      string `json:"created,omitempty"`
	Updated      string `json:"updated,omitempty"`
}

type CommentsResult struct {
	StartAt    int       `json:"startAt"`
	MaxResults int       `json:"maxResults"`
	Total      int       `json:"total"`
	Comments   []Comment `json:"comments"`
}

type Worklog struct {
	ID               string `json:"id,omitempty"`
	Author           *User  `json:"author,omitempty"`
	UpdateAuthor     *User  `json:"updateAuthor,omitempty"`
	Comment          string `json:"comment,omitempty"`
	Created          string `json:"created,omitempty"`
	Updated          string `json:"updated,omitempty"`
	Started          string `json:"started,omitempty"`
	TimeSpent        string `json:"timeSpent,omitempty"`
	TimeSpentSeconds int    `json:"timeSpentSeconds,omitempty"`
}

type WorklogsResult struct {
	StartAt    int       `json:"startAt"`
	MaxResults int       `json:"maxResults"`
	Total      int       `json:"total"`
	Worklogs   []Worklog `json:"worklogs"`
}

type Filter struct {
	ID          string `json:"id,omitempty"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	JQL         string `json:"jql,omitempty"`
	Self        string `json:"self,omitempty"`
	ViewURL     string `json:"viewUrl,omitempty"`
	SearchURL   string `json:"searchUrl,omitempty"`
	Favourite   bool   `json:"favourite,omitempty"`
	Owner       *User  `json:"owner,omitempty"`
}

type RemoteIssueLink struct {
	ID           int               `json:"id,omitempty"`
	Self         string            `json:"self,omitempty"`
	GlobalID     string            `json:"globalId,omitempty"`
	Relationship string            `json:"relationship,omitempty"`
	Object       RemoteLinkObject  `json:"object,omitempty"`
	Application  map[string]string `json:"application,omitempty"`
}

type RemoteLinkObject struct {
	URL     string `json:"url,omitempty"`
	Title   string `json:"title,omitempty"`
	Summary string `json:"summary,omitempty"`
}

type Version struct {
	ID          string `json:"id,omitempty"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Archived    bool   `json:"archived,omitempty"`
	Released    bool   `json:"released,omitempty"`
	Overdue     bool   `json:"overdue,omitempty"`
}

type Attachment struct {
	ID       string `json:"id,omitempty"`
	Filename string `json:"filename,omitempty"`
	Size     int64  `json:"size,omitempty"`
	MimeType string `json:"mimeType,omitempty"`
	Content  string `json:"content,omitempty"`
	Self     string `json:"self,omitempty"`
	Author   *User  `json:"author,omitempty"`
	Created  string `json:"created,omitempty"`
}

type WriteResult struct {
	ID   string `json:"id,omitempty"`
	Key  string `json:"key,omitempty"`
	Self string `json:"self,omitempty"`
}

func SearchQuery(jql string, startAt, maxResults int) url.Values {
	query := url.Values{}
	query.Set("jql", jql)
	query.Set("startAt", strconv.Itoa(startAt))
	query.Set("maxResults", strconv.Itoa(maxResults))
	query.Set("fields", "summary,status,priority,assignee,project,created,updated")
	return query
}
