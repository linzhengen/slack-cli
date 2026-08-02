package slackapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCall_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat.postMessage" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer xoxb-test" {
			t.Fatalf("unexpected auth header %q", got)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		if r.PostForm.Get("channel") != "C123" {
			t.Fatalf("unexpected channel %q", r.PostForm.Get("channel"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok":true,"ts":"123.456","channel":"C123"}`))
	}))
	defer srv.Close()

	c := New("xoxb-test", srv.URL)
	resp, err := c.Call(context.Background(), "chat.postMessage", map[string]string{"channel": "C123", "text": "hi"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp["ts"] != "123.456" {
		t.Fatalf("unexpected response: %v", resp)
	}
}

func TestCall_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok":false,"error":"channel_not_found"}`))
	}))
	defer srv.Close()

	c := New("xoxb-test", srv.URL)
	_, err := c.Call(context.Background(), "chat.postMessage", map[string]string{"channel": "bad"})
	if err == nil {
		t.Fatal("expected error")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T: %v", err, err)
	}
	if apiErr.Slack != "channel_not_found" {
		t.Fatalf("unexpected slack error: %q", apiErr.Slack)
	}
}

func TestCall_RetriesOn429(t *testing.T) {
	attempts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c := New("xoxb-test", srv.URL)
	resp, err := c.Call(context.Background(), "auth.test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if attempts != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempts)
	}
	if resp["ok"] != true {
		t.Fatalf("unexpected response: %v", resp)
	}
}

func TestCall_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("boom"))
	}))
	defer srv.Close()

	c := New("xoxb-test", srv.URL)
	_, err := c.Call(context.Background(), "auth.test", nil)
	if err == nil {
		t.Fatal("expected error")
	}
	httpErr, ok := err.(*HTTPError)
	if !ok {
		t.Fatalf("expected *HTTPError, got %T: %v", err, err)
	}
	if httpErr.StatusCode != 500 || !strings.Contains(httpErr.Body, "boom") {
		t.Fatalf("unexpected HTTPError: %+v", httpErr)
	}
}
