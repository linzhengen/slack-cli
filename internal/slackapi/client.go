// Package slackapi implements a thin, generic client for the Slack Web API.
//
// Rather than hand-writing a typed Go function per endpoint, the client
// exposes a single Call method that can invoke any Slack Web API method by
// name. This keeps the client correct for the entire API surface, including
// methods added by Slack after this code was written.
package slackapi

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
	"strings"
	"time"
)

// DefaultBaseURL is Slack's Web API endpoint.
const DefaultBaseURL = "https://slack.com/api"

// Client calls Slack Web API methods over HTTP.
type Client struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client

	// MaxRetries controls how many times a request is retried after a 429
	// (rate limited) response, honoring the Retry-After header.
	MaxRetries int
}

// New creates a Client for the given token. baseURL may be empty, in which
// case DefaultBaseURL is used.
func New(token, baseURL string) *Client {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	return &Client{
		BaseURL:    strings.TrimRight(baseURL, "/"),
		Token:      token,
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
		MaxRetries: 2,
	}
}

// APIError represents a Slack Web API error response (HTTP 200, ok:false).
type APIError struct {
	Method   string
	Slack    string            // the "error" field from Slack
	Warning  string            // the "warning" field from Slack, if any
	Response map[string]any    // full decoded response
	Extra    map[string]string // response_metadata.messages etc, best-effort
}

func (e *APIError) Error() string {
	return fmt.Sprintf("slack: %s: %s", e.Method, e.Slack)
}

// HTTPError represents a non-2xx transport-level failure.
type HTTPError struct {
	Method     string
	StatusCode int
	Body       string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("slack: %s: unexpected HTTP status %d: %s", e.Method, e.StatusCode, e.Body)
}

// Call invokes the given Slack Web API method (e.g. "chat.postMessage") with
// the given parameters, encoded as application/x-www-form-urlencoded. Values
// that are not plain strings are handled by the caller via ToParams; this
// method just sends whatever string map it is given. Array/object values
// destined for fields like "blocks" or "attachments" must already be
// JSON-encoded strings, which is the form Slack's Web API expects for
// form-encoded requests.
//
// The raw decoded JSON response is returned. If Slack reports ok:false, an
// *APIError is returned alongside the decoded body.
func (c *Client) Call(ctx context.Context, method string, params map[string]string) (map[string]any, error) {
	form := url.Values{}
	for k, v := range params {
		form.Set(k, v)
	}

	endpoint := c.BaseURL + "/" + method

	var lastErr error
	for attempt := 0; attempt <= c.MaxRetries; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=utf-8")
		if c.Token != "" {
			req.Header.Set("Authorization", "Bearer "+c.Token)
		}

		resp, err := c.HTTPClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			return nil, readErr
		}

		if resp.StatusCode == http.StatusTooManyRequests && attempt < c.MaxRetries {
			wait := 1 * time.Second
			if ra := resp.Header.Get("Retry-After"); ra != "" {
				if secs, perr := strconv.Atoi(ra); perr == nil {
					wait = time.Duration(secs) * time.Second
				}
			}
			select {
			case <-time.After(wait):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
			continue
		}

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, &HTTPError{Method: method, StatusCode: resp.StatusCode, Body: string(body)}
		}

		var decoded map[string]any
		if err := json.Unmarshal(body, &decoded); err != nil {
			return nil, fmt.Errorf("slack: %s: decoding response: %w (body: %s)", method, err, truncate(body, 500))
		}

		ok, _ := decoded["ok"].(bool)
		if !ok {
			apiErr := &APIError{Method: method, Response: decoded}
			if s, isStr := decoded["error"].(string); isStr {
				apiErr.Slack = s
			} else {
				apiErr.Slack = "unknown_error"
			}
			if w, isStr := decoded["warning"].(string); isStr {
				apiErr.Warning = w
			}
			return decoded, apiErr
		}

		return decoded, nil
	}
	return nil, lastErr
}

// UploadFile performs a raw multipart/form-data POST of file content to a
// pre-signed upload URL, as returned by files.getUploadURLExternal. This is
// not a normal Web API call: it does not use the "/api/" prefix, is not
// authenticated with a bearer token, and does not return a Slack-shaped
// ok/error JSON body.
func (c *Client) UploadFile(ctx context.Context, uploadURL string, filename string, content io.Reader) error {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile("file", filename)
	if err != nil {
		return err
	}
	if _, err := io.Copy(fw, content); err != nil {
		return err
	}
	if err := mw.Close(); err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, uploadURL, &buf)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &HTTPError{Method: "files.upload(raw)", StatusCode: resp.StatusCode, Body: string(body)}
	}
	return nil
}

func truncate(b []byte, n int) string {
	if len(b) <= n {
		return string(b)
	}
	return string(b[:n]) + "..."
}
