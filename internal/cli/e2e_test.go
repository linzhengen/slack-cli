package cli

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// run executes root against the given args, with --token/--base-url wired
// to a local test server, and returns stdout, stderr, and any error.
func run(t *testing.T, srv *httptest.Server, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	root := NewRootCmd()
	var outBuf, errBuf bytes.Buffer
	root.SetOut(&outBuf)
	root.SetErr(&errBuf)
	full := append([]string{"--token", "xoxb-test", "--base-url", srv.URL}, args...)
	root.SetArgs(full)
	err = root.Execute()
	return outBuf.String(), errBuf.String(), err
}

func TestE2E_GeneratedCommand_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat.postMessage" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"ok":true,"ts":"1.1","channel":"C1"}`))
	}))
	defer srv.Close()

	stdout, _, err := run(t, srv, "chat", "post-message", "--channel", "C1", "--text", "hi")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var got map[string]any
	if jerr := json.Unmarshal([]byte(stdout), &got); jerr != nil {
		t.Fatalf("stdout not valid JSON: %v\n%s", jerr, stdout)
	}
	if got["ts"] != "1.1" {
		t.Fatalf("unexpected response: %v", got)
	}
}

func TestE2E_MissingRequiredParam(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server should not be called when a required param is missing")
	}))
	defer srv.Close()

	_, stderr, err := run(t, srv, "chat", "post-message", "--text", "hi")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(stderr, "channel") {
		t.Fatalf("expected stderr to mention missing 'channel', got: %s", stderr)
	}
}

func TestE2E_APICall_UnknownMethod(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/some.newMethod" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"ok":true,"custom":"field"}`))
	}))
	defer srv.Close()

	stdout, _, err := run(t, srv, "api", "call", "some.newMethod", "--param", "foo=bar")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, `"custom": "field"`) {
		t.Fatalf("unexpected stdout: %s", stdout)
	}
}

func TestE2E_APIError_SurfacesSlackBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"ok":false,"error":"channel_not_found"}`))
	}))
	defer srv.Close()

	_, stderr, err := run(t, srv, "chat", "post-message", "--channel", "bad", "--text", "hi")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(stderr, "channel_not_found") {
		t.Fatalf("expected stderr to contain slack error, got: %s", stderr)
	}
}

func TestE2E_AdminNestedGroup(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/admin.users.session.reset" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	_, _, err := run(t, srv, "admin", "users", "session", "reset", "--user_id", "U1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestE2E_AuthWhoami(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/auth.test" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"ok":true,"user":"tester"}`))
	}))
	defer srv.Close()

	stdout, _, err := run(t, srv, "auth", "whoami")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "tester") {
		t.Fatalf("unexpected stdout: %s", stdout)
	}
}
