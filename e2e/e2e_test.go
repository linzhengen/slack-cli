//go:build e2e

package e2e

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/linzhengen/slack-cli/internal/slacktest"
)

func mustJSON(t *testing.T, s string) map[string]any {
	t.Helper()
	var v map[string]any
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, s)
	}
	return v
}

func TestE2E_AuthWhoami(t *testing.T) {
	srv := slacktest.New()
	defer srv.Close()

	res := runCLI(t, srv, "auth", "whoami")
	if res.ExitCode != 0 {
		t.Fatalf("unexpected exit %d, stderr=%s", res.ExitCode, res.Stderr)
	}
	got := mustJSON(t, res.Stdout)
	if got["user"] != "testbot" {
		t.Fatalf("unexpected response: %v", got)
	}
	if len(srv.CallsFor("auth.test")) != 1 {
		t.Fatalf("expected exactly 1 auth.test call, got %d", len(srv.CallsFor("auth.test")))
	}
}

func TestE2E_PostMessageAndThreadReply(t *testing.T) {
	srv := slacktest.New()
	defer srv.Close()

	res := runCLI(t, srv, "chat", "post-message", "--channel", "C123", "--text", "hello", "--output", "compact")
	if res.ExitCode != 0 {
		t.Fatalf("unexpected exit %d, stderr=%s", res.ExitCode, res.Stderr)
	}
	got := mustJSON(t, res.Stdout)
	ts, _ := got["ts"].(string)
	if ts == "" {
		t.Fatalf("expected a ts in response: %v", got)
	}

	res2 := runCLI(t, srv, "chat", "post-message", "--channel", "C123", "--thread_ts", ts, "--text", "reply")
	if res2.ExitCode != 0 {
		t.Fatalf("unexpected exit %d, stderr=%s", res2.ExitCode, res2.Stderr)
	}

	calls := srv.CallsFor("chat.postMessage")
	if len(calls) != 2 {
		t.Fatalf("expected 2 chat.postMessage calls, got %d", len(calls))
	}
	if calls[1].Form["thread_ts"] != ts {
		t.Fatalf("expected second call's thread_ts to be %q, got %q", ts, calls[1].Form["thread_ts"])
	}
}

func TestE2E_MissingRequiredParam_NeverCallsServer(t *testing.T) {
	srv := slacktest.New()
	defer srv.Close()

	res := runCLI(t, srv, "chat", "post-message", "--text", "hi")
	if res.ExitCode == 0 {
		t.Fatalf("expected non-zero exit, stdout=%s", res.Stdout)
	}
	if !strings.Contains(res.Stderr, "channel") {
		t.Fatalf("expected stderr to mention missing 'channel', got: %s", res.Stderr)
	}
	if len(srv.CallsFor("chat.postMessage")) != 0 {
		t.Fatal("server should never have been called for a client-side validation failure")
	}
}

func TestE2E_SlackAPIError_PassesThroughWithNonZeroExit(t *testing.T) {
	srv := slacktest.New()
	defer srv.Close()
	srv.QueueError("chat.postMessage", "channel_not_found")

	res := runCLI(t, srv, "chat", "post-message", "--channel", "bad", "--text", "hi")
	if res.ExitCode == 0 {
		t.Fatalf("expected non-zero exit, stdout=%s", res.Stdout)
	}
	if !strings.Contains(res.Stderr, "channel_not_found") {
		t.Fatalf("expected stderr to contain the Slack error, got: %s", res.Stderr)
	}
	if res.Stdout != "" {
		t.Fatalf("expected empty stdout on failure, got: %s", res.Stdout)
	}
}

func TestE2E_RateLimitedRequestRetriesTransparently(t *testing.T) {
	srv := slacktest.New()
	defer srv.Close()
	srv.QueueRateLimited("auth.test", 0) // Retry-After: 0 keeps the test fast

	res := runCLI(t, srv, "auth", "whoami")
	if res.ExitCode != 0 {
		t.Fatalf("unexpected exit %d, stderr=%s", res.ExitCode, res.Stderr)
	}
	if got := len(srv.CallsFor("auth.test")); got != 2 {
		t.Fatalf("expected 2 attempts (1 rate-limited + 1 success), got %d", got)
	}
}

func TestE2E_FilesUpload_FullThreeStepFlow(t *testing.T) {
	srv := slacktest.New()
	defer srv.Close()

	dir := t.TempDir()
	path := filepath.Join(dir, "report.csv")
	content := []byte("a,b,c\n1,2,3\n")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("writing test file: %v", err)
	}

	res := runCLI(t, srv, "files", "upload", "--file", path, "--channel", "C123", "--title", "Report")
	if res.ExitCode != 0 {
		t.Fatalf("unexpected exit %d, stderr=%s", res.ExitCode, res.Stderr)
	}
	got := mustJSON(t, res.Stdout)
	files, _ := got["files"].([]any)
	if len(files) != 1 {
		t.Fatalf("unexpected completeUploadExternal response: %v", got)
	}

	getCalls := srv.CallsFor("files.getUploadURLExternal")
	if len(getCalls) != 1 || getCalls[0].Form["filename"] != "report.csv" {
		t.Fatalf("unexpected files.getUploadURLExternal calls: %+v", getCalls)
	}

	completeCalls := srv.CallsFor("files.completeUploadExternal")
	if len(completeCalls) != 1 {
		t.Fatalf("expected exactly 1 files.completeUploadExternal call, got %d", len(completeCalls))
	}
	if completeCalls[0].Form["channel_id"] != "C123" {
		t.Fatalf("expected channel_id=C123, got %+v", completeCalls[0].Form)
	}

	var requested []struct {
		ID    string `json:"id"`
		Title string `json:"title"`
	}
	if err := json.Unmarshal([]byte(completeCalls[0].Form["files"]), &requested); err != nil || len(requested) != 1 {
		t.Fatalf("unexpected 'files' param: %q (err=%v)", completeCalls[0].Form["files"], err)
	}

	// The important bit: the actual file bytes made it through the raw
	// multipart POST to the upload URL, not just the metadata calls.
	uploaded, ok := srv.UploadedFile(requested[0].ID)
	if !ok || string(uploaded) != string(content) {
		t.Fatalf("uploaded content mismatch: got %q (ok=%v), want %q", uploaded, ok, content)
	}
}

func TestE2E_SkillsListAndRead(t *testing.T) {
	srv := slacktest.New()
	defer srv.Close()

	res := runCLI(t, srv, "skills", "list", "--output", "compact")
	if res.ExitCode != 0 {
		t.Fatalf("unexpected exit %d, stderr=%s", res.ExitCode, res.Stderr)
	}
	got := mustJSON(t, res.Stdout)
	count, _ := got["count"].(float64)
	if count < 5 {
		t.Fatalf("expected several embedded skills, got: %v", got)
	}

	res2 := runCLI(t, srv, "skills", "read", "slack-shared")
	if res2.ExitCode != 0 {
		t.Fatalf("unexpected exit %d, stderr=%s", res2.ExitCode, res2.Stderr)
	}
	if !strings.Contains(res2.Stdout, "name: slack-shared") {
		t.Fatalf("unexpected stdout: %s", res2.Stdout)
	}
	// This assertion is what an in-process test (internal/skillsfs_test.go)
	// can't give us: proof the *compiled binary* actually embeds skills/,
	// not just that the source tree parses under `go test`.
}

func TestE2E_AdminNestedCommandGroup(t *testing.T) {
	srv := slacktest.New()
	defer srv.Close()

	res := runCLI(t, srv, "admin", "users", "session", "reset", "--user_id", "U1", "--output", "compact")
	if res.ExitCode != 0 {
		t.Fatalf("unexpected exit %d, stderr=%s", res.ExitCode, res.Stderr)
	}
	calls := srv.CallsFor("admin.users.session.reset")
	if len(calls) != 1 || calls[0].Form["user_id"] != "U1" {
		t.Fatalf("unexpected admin.users.session.reset calls: %+v", calls)
	}
}

func TestE2E_TokenIsSentAsBearerAuthorizationHeader(t *testing.T) {
	srv := slacktest.New()
	srv.Token = defaultTestToken // matches what runCLI passes
	defer srv.Close()

	res := runCLI(t, srv, "auth", "whoami")
	if res.ExitCode != 0 {
		t.Fatalf("expected success with a matching token, stderr=%s", res.Stderr)
	}
}

func TestE2E_MismatchedTokenIsRejected(t *testing.T) {
	srv := slacktest.New()
	srv.Token = "xoxb-some-other-token"
	defer srv.Close()

	res := runCLI(t, srv, "auth", "whoami")
	if res.ExitCode == 0 {
		t.Fatalf("expected failure with a mismatched token, stdout=%s", res.Stdout)
	}
	if !strings.Contains(res.Stderr, "invalid_auth") {
		t.Fatalf("unexpected stderr: %s", res.Stderr)
	}
}
