package cli

import (
	"testing"

	"github.com/linzhengen/slack-cli/internal/slackapi"
)

func TestKebab(t *testing.T) {
	cases := map[string]string{
		"chat":                 "chat",
		"postMessage":          "post-message",
		"authPolicy":           "auth-policy",
		"getUploadURLExternal": "get-upload-url-external",
		"usergroups":           "usergroups",
		"setAdmin":             "set-admin",
	}
	for in, want := range cases {
		if got := kebab(in); got != want {
			t.Errorf("kebab(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestToParamString(t *testing.T) {
	s, err := toParamString("hello")
	if err != nil || s != "hello" {
		t.Fatalf("string passthrough failed: %q, %v", s, err)
	}

	s, err = toParamString(true)
	if err != nil || s != "true" {
		t.Fatalf("bool encoding failed: %q, %v", s, err)
	}

	s, err = toParamString([]any{"a", "b"})
	if err != nil || s != `["a","b"]` {
		t.Fatalf("array encoding failed: %q, %v", s, err)
	}
}

func TestMissingRequired(t *testing.T) {
	declared := []slackapi.Param{
		{Name: "channel", Required: true},
		{Name: "text", Required: false},
	}

	got := missingRequired(declared, map[string]string{"text": "hi"})
	if len(got) != 1 || got[0] != "channel" {
		t.Fatalf("expected [channel], got %v", got)
	}

	got = missingRequired(declared, map[string]string{"channel": "C1", "text": "hi"})
	if len(got) != 0 {
		t.Fatalf("expected no missing params, got %v", got)
	}
}
