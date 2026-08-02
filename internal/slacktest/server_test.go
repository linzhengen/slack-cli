package slacktest

import (
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func post(t *testing.T, srv *Server, method string, form url.Values) map[string]any {
	t.Helper()
	resp, err := http.PostForm(srv.URL+"/"+method, form)
	if err != nil {
		t.Fatalf("POST %s: %v", method, err)
	}
	defer func() { _ = resp.Body.Close() }()
	var out map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	out["_status"] = resp.StatusCode
	return out
}

func TestDefaultResponses_AuthTest(t *testing.T) {
	srv := New()
	defer srv.Close()

	resp := post(t, srv, "auth.test", nil)
	if resp["ok"] != true || resp["user"] != "testbot" {
		t.Fatalf("unexpected response: %v", resp)
	}
}

func TestDefaultResponses_PostMessageGeneratesTS(t *testing.T) {
	srv := New()
	defer srv.Close()

	resp := post(t, srv, "chat.postMessage", url.Values{"channel": {"C123"}, "text": {"hi"}})
	if resp["ok"] != true || resp["channel"] != "C123" {
		t.Fatalf("unexpected response: %v", resp)
	}
	ts, _ := resp["ts"].(string)
	if ts == "" {
		t.Fatal("expected a non-empty ts")
	}

	resp2 := post(t, srv, "chat.postMessage", url.Values{"channel": {"C123"}, "text": {"hi again"}})
	if resp2["ts"] == ts {
		t.Fatal("expected a different ts for a second message")
	}
}

func TestCallRecording(t *testing.T) {
	srv := New()
	defer srv.Close()

	post(t, srv, "auth.test", nil)
	post(t, srv, "chat.postMessage", url.Values{"channel": {"C1"}, "text": {"hi"}})
	post(t, srv, "chat.postMessage", url.Values{"channel": {"C2"}, "text": {"bye"}})

	if len(srv.Calls()) != 3 {
		t.Fatalf("expected 3 recorded calls, got %d", len(srv.Calls()))
	}
	pm := srv.CallsFor("chat.postMessage")
	if len(pm) != 2 || pm[0].Form["channel"] != "C1" || pm[1].Form["channel"] != "C2" {
		t.Fatalf("unexpected chat.postMessage calls: %+v", pm)
	}
}

func TestQueueError(t *testing.T) {
	srv := New()
	defer srv.Close()

	srv.QueueError("chat.postMessage", "channel_not_found")

	resp := post(t, srv, "chat.postMessage", url.Values{"channel": {"bad"}, "text": {"hi"}})
	if resp["ok"] != false || resp["error"] != "channel_not_found" {
		t.Fatalf("unexpected response: %v", resp)
	}

	// Queue is one-shot: the next call falls back to the default.
	resp2 := post(t, srv, "chat.postMessage", url.Values{"channel": {"C1"}, "text": {"hi"}})
	if resp2["ok"] != true {
		t.Fatalf("expected default response after queue drained, got %v", resp2)
	}
}

func TestQueueRateLimited(t *testing.T) {
	srv := New()
	defer srv.Close()

	srv.QueueRateLimited("auth.test", 1)

	resp, err := http.PostForm(srv.URL+"/auth.test", nil)
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", resp.StatusCode)
	}
	if got := resp.Header.Get("Retry-After"); got != "1" {
		t.Fatalf("expected Retry-After: 1, got %q", got)
	}

	// Next call succeeds normally.
	resp2 := post(t, srv, "auth.test", nil)
	if resp2["ok"] != true {
		t.Fatalf("expected success after rate limit cleared, got %v", resp2)
	}
}

func TestHandlePermanentOverride(t *testing.T) {
	srv := New()
	defer srv.Close()

	srv.Handle("custom.method", func(r *http.Request) (int, any) {
		return http.StatusOK, map[string]any{"ok": true, "custom": "value"}
	})

	for i := 0; i < 3; i++ {
		resp := post(t, srv, "custom.method", nil)
		if resp["custom"] != "value" {
			t.Fatalf("call %d: unexpected response: %v", i, resp)
		}
	}
}

func TestTokenValidation(t *testing.T) {
	srv := New()
	srv.Token = "xoxb-expected"
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/auth.test", strings.NewReader(""))
	req.Header.Set("Authorization", "Bearer xoxb-wrong")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	var out map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&out)
	if out["ok"] != false || out["error"] != "invalid_auth" {
		t.Fatalf("expected invalid_auth, got %v", out)
	}
}

func TestUploadFlow(t *testing.T) {
	srv := New()
	defer srv.Close()

	getResp := post(t, srv, "files.getUploadURLExternal", url.Values{"filename": {"a.txt"}, "length": {"5"}})
	uploadURL, _ := getResp["upload_url"].(string)
	fileID, _ := getResp["file_id"].(string)
	if uploadURL == "" || fileID == "" {
		t.Fatalf("unexpected getUploadURLExternal response: %v", getResp)
	}

	var buf strings.Builder
	mw := multipart.NewWriter(&buf)
	fw, _ := mw.CreateFormFile("file", "a.txt")
	_, _ = fw.Write([]byte("hello"))
	_ = mw.Close()

	req, _ := http.NewRequest(http.MethodPost, uploadURL, strings.NewReader(buf.String()))
	req.Header.Set("Content-Type", mw.FormDataContentType())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("upload POST: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected upload status %d: %s", resp.StatusCode, body)
	}

	data, ok := srv.UploadedFile(fileID)
	if !ok || string(data) != "hello" {
		t.Fatalf("expected uploaded content 'hello', got %q (ok=%v)", data, ok)
	}

	filesJSON := `[{"id":"` + fileID + `","title":"A"}]`
	completeResp := post(t, srv, "files.completeUploadExternal", url.Values{"files": {filesJSON}})
	files, _ := completeResp["files"].([]any)
	if len(files) != 1 {
		t.Fatalf("unexpected completeUploadExternal response: %v", completeResp)
	}
}
