package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"testing/fstest"
)

func testSkillFS() fstest.MapFS {
	return fstest.MapFS{
		"demo/SKILL.md": &fstest.MapFile{Data: []byte(`---
name: demo
version: 1.0.0
description: "Demo skill for tests"
---

# demo skill body
`)},
		"demo/references/notes.md": &fstest.MapFile{Data: []byte("demo reference notes")},
	}
}

// runSkills executes root against args with the given skill content wired
// in, restoring the previous skillFS afterward so tests don't leak state.
func runSkills(t *testing.T, fsys fstest.MapFS, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	prev := skillFS
	SetSkillContent(fsys)
	t.Cleanup(func() { skillFS = prev })

	root := NewRootCmd()
	var outBuf, errBuf bytes.Buffer
	root.SetOut(&outBuf)
	root.SetErr(&errBuf)
	root.SetArgs(args)
	err = root.Execute()
	return outBuf.String(), errBuf.String(), err
}

func TestSkillsList_All(t *testing.T) {
	stdout, _, err := runSkills(t, testSkillFS(), "skills", "list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var got struct {
		OK     bool `json:"ok"`
		Count  int  `json:"count"`
		Skills []struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		} `json:"skills"`
	}
	if jerr := json.Unmarshal([]byte(stdout), &got); jerr != nil {
		t.Fatalf("stdout not valid JSON: %v\n%s", jerr, stdout)
	}
	if !got.OK || got.Count != 1 || len(got.Skills) != 1 || got.Skills[0].Name != "demo" {
		t.Fatalf("unexpected response: %+v", got)
	}
}

func TestSkillsRead_RawWithGuidanceOnStderr(t *testing.T) {
	stdout, stderr, err := runSkills(t, testSkillFS(), "skills", "read", "demo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "demo skill body") {
		t.Fatalf("unexpected stdout: %s", stdout)
	}
	if !strings.Contains(stderr, "slack-cli skills read demo") {
		t.Fatalf("expected guidance tip on stderr, got: %s", stderr)
	}
}

func TestSkillsRead_JSONEnvelope(t *testing.T) {
	stdout, _, err := runSkills(t, testSkillFS(), "skills", "read", "demo", "--json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var got struct {
		Skill    string `json:"skill"`
		Path     string `json:"path"`
		Content  string `json:"content"`
		Guidance string `json:"guidance"`
	}
	if jerr := json.Unmarshal([]byte(stdout), &got); jerr != nil {
		t.Fatalf("stdout not valid JSON: %v\n%s", jerr, stdout)
	}
	if got.Skill != "demo" || got.Path != "SKILL.md" || !strings.Contains(got.Content, "demo skill body") || got.Guidance == "" {
		t.Fatalf("unexpected envelope: %+v", got)
	}
}

func TestSkillsRead_Reference(t *testing.T) {
	stdout, stderr, err := runSkills(t, testSkillFS(), "skills", "read", "demo/references/notes.md")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stdout != "demo reference notes" {
		t.Fatalf("unexpected stdout: %q", stdout)
	}
	if stderr != "" {
		t.Fatalf("expected no guidance tip for a reference file, got: %s", stderr)
	}
}

func TestSkillsRead_UnknownSkill(t *testing.T) {
	_, stderr, err := runSkills(t, testSkillFS(), "skills", "read", "nope")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(stderr, "unknown skill") {
		t.Fatalf("unexpected stderr: %s", stderr)
	}
}

func TestSkillsRead_PathTraversalRejected(t *testing.T) {
	_, stderr, err := runSkills(t, testSkillFS(), "skills", "read", "demo/../../etc/passwd")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(stderr, "must be a relative path") {
		t.Fatalf("unexpected stderr: %s", stderr)
	}
}

func TestSkillsCommands_NotEmbedded(t *testing.T) {
	prev := skillFS
	SetSkillContent(nil)
	t.Cleanup(func() { skillFS = prev })

	root := NewRootCmd()
	var outBuf, errBuf bytes.Buffer
	root.SetOut(&outBuf)
	root.SetErr(&errBuf)
	root.SetArgs([]string{"skills", "list"})
	if err := root.Execute(); err == nil {
		t.Fatal("expected error when skill content isn't embedded")
	}
	if !strings.Contains(errBuf.String(), "not embedded") {
		t.Fatalf("unexpected stderr: %s", errBuf.String())
	}
}
