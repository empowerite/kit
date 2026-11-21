package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/empowerite/kit/internal/ports"
)

// Answer is what an address answers: a status and a body.
type Answer struct {
	Status int
	Body   string
}

// Memory is a transport that answers from memory, by address, and keeps the headers of the last request. kit's tests
// use it, since no test the gate runs opens a connection, and a test may not name net/http.
type Memory struct {
	answers map[string]Answer
	mu      sync.Mutex
	headers map[string]string
}

func memory(answers map[string]Answer) *Memory { return &Memory{answers: answers} }

// Headers gives the headers of the last request.
func (m *Memory) Headers() map[string]string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.headers
}

func (m *Memory) RoundTrip(req *http.Request) (*http.Response, error) {
	m.mu.Lock()
	m.headers = map[string]string{}
	for key := range req.Header {
		m.headers[key] = req.Header.Get(key)
	}
	m.mu.Unlock()
	a, ok := m.answers[req.URL.String()]
	if !ok {
		return nil, fmt.Errorf("nothing at %s", req.URL)
	}
	return &http.Response{StatusCode: a.Status, Body: io.NopCloser(bytes.NewReader([]byte(a.Body))), Request: req}, nil
}

// client gives the client that sends through m, or the real one where m is nil.
func client(m *Memory) *http.Client {
	if m == nil {
		return http.DefaultClient
	}
	return &http.Client{Transport: m}
}

// get asks an address, with headers, and gives its status and its body.
func get(ctx context.Context, c *http.Client, url string, headers map[string]string) (int, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, nil, err
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	resp, err := c.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	return resp.StatusCode, body, err
}

// Releases is GitHub's releases, read through its API, with a token where there is one.
type Releases struct {
	client *http.Client
	token  string
}

// NewReleases gives GitHub's releases, asked with token, or with none where it is "". m is nil for the real GitHub.
func NewReleases(token string, m *Memory) Releases { return Releases{client(m), token} }

// release is what GitHub says of a release.
type release struct {
	Tag    string `json:"tag_name"`
	Assets []struct {
		Name   string  `json:"name"`
		URL    string  `json:"browser_download_url"`
		Digest *string `json:"digest"`
	} `json:"assets"`
}

func (r Releases) release(ctx context.Context, repo, path, missing string) (release, error) {
	headers := map[string]string{"Accept": "application/vnd.github+json", "X-GitHub-Api-Version": "2022-11-28"}
	if r.token != "" {
		headers["Authorization"] = "Bearer " + r.token
	}
	var got release
	status, body, err := get(ctx, r.client, "https://api.github.com/repos/"+repo+"/releases/"+path, headers)
	switch {
	case err != nil:
		return got, fmt.Errorf("github: %w", err)
	case status == http.StatusNotFound:
		return got, fmt.Errorf("github: %s has %s", repo, missing)
	case status != http.StatusOK:
		var said struct{ Message string }
		json.Unmarshal(body, &said)
		return got, fmt.Errorf("github: %s: %d: %s", repo, status, said.Message)
	}
	if err := json.Unmarshal(body, &got); err != nil {
		return got, fmt.Errorf("github: %s: %w", repo, err)
	}
	return got, nil
}

func (r Releases) Asset(ctx context.Context, repo, tag, name string) (ports.Asset, error) {
	got, err := r.release(ctx, repo, "tags/"+tag, "no release "+tag)
	if err != nil {
		return ports.Asset{}, err
	}
	for _, a := range got.Assets {
		if a.Name == name {
			digest := ""
			if a.Digest != nil {
				digest = *a.Digest
			}
			return ports.Asset{URL: a.URL, Digest: digest}, nil
		}
	}
	return ports.Asset{}, fmt.Errorf("github: %s %s has no file %s", repo, tag, name)
}

func (r Releases) Newest(ctx context.Context, repo string) (string, error) {
	got, err := r.release(ctx, repo, "latest", "no release")
	return got.Tag, err
}

// Downloads fetches what an address holds, following GitHub's redirects to where its files are kept.
type Downloads struct{ client *http.Client }

// NewDownloads gives downloads through m, or the real ones where m is nil.
func NewDownloads(m *Memory) Downloads { return Downloads{client(m)} }

func (d Downloads) Get(ctx context.Context, url string) ([]byte, error) {
	status, body, err := get(ctx, d.client, url, nil)
	switch {
	case err != nil:
		return nil, fmt.Errorf("download: %w", err)
	case status != http.StatusOK:
		return nil, fmt.Errorf("download: %s: %d", url, status)
	}
	return body, nil
}
