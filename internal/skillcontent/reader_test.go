package skillcontent

import (
	"testing"
	"testing/fstest"
)

func testFS() fstest.MapFS {
	return fstest.MapFS{
		"alpha/SKILL.md": &fstest.MapFile{Data: []byte(`---
name: alpha
version: 1.2.3
description: "Alpha skill description"
metadata:
  requires:
    bins: ["slack-cli"]
---

# alpha

body text.
`)},
		"alpha/references/notes.md": &fstest.MapFile{Data: []byte("alpha notes")},
		"beta/SKILL.md":             &fstest.MapFile{Data: []byte("---\ndescription: \"Beta skill\"\n---\nbeta body\n")},
		"not-a-skill/readme.txt":    &fstest.MapFile{Data: []byte("no SKILL.md here")},
	}
}

func TestList(t *testing.T) {
	r := New(testFS())
	skills, err := r.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(skills) != 2 {
		t.Fatalf("expected 2 skills (not-a-skill excluded), got %d: %+v", len(skills), skills)
	}
	if skills[0].Name != "alpha" || skills[1].Name != "beta" {
		t.Fatalf("expected sorted [alpha, beta], got %+v", skills)
	}
	if skills[0].Description != "Alpha skill description" || skills[0].Version != "1.2.3" {
		t.Fatalf("unexpected alpha frontmatter: %+v", skills[0])
	}
	if got := skills[0].Metadata["requires"]; got == nil {
		t.Fatalf("expected metadata.requires to survive parsing, got %+v", skills[0].Metadata)
	}
	if skills[1].Version != "" {
		t.Fatalf("expected beta to have no version, got %q", skills[1].Version)
	}
}

func TestReadSkill(t *testing.T) {
	r := New(testFS())
	data, err := r.ReadSkill("alpha")
	if err != nil {
		t.Fatalf("ReadSkill: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("expected non-empty content")
	}

	if _, err := r.ReadSkill("nope"); err == nil {
		t.Fatal("expected error for unknown skill")
	}
	for _, bad := range []string{"", ".", "..", "alpha/../beta", "a/b"} {
		if _, err := r.ReadSkill(bad); err == nil {
			t.Errorf("expected error for invalid skill name %q", bad)
		}
	}
}

func TestReadReference(t *testing.T) {
	r := New(testFS())
	data, cleaned, err := r.ReadReference("alpha", "references/notes.md")
	if err != nil {
		t.Fatalf("ReadReference: %v", err)
	}
	if string(data) != "alpha notes" {
		t.Fatalf("unexpected content: %q", data)
	}
	if cleaned != "references/notes.md" {
		t.Fatalf("unexpected cleaned path: %q", cleaned)
	}

	// Missing file.
	if _, _, err := r.ReadReference("alpha", "references/missing.md"); err == nil {
		t.Fatal("expected error for missing reference")
	}

	// Directory, not a file.
	if _, _, err := r.ReadReference("alpha", "references"); err == nil {
		t.Fatal("expected error reading a directory as a reference")
	}

	// Path traversal attempts must be rejected.
	for _, bad := range []string{"../beta/SKILL.md", "..", "/etc/passwd", `..\beta\SKILL.md`, ""} {
		if _, _, err := r.ReadReference("alpha", bad); err == nil {
			t.Errorf("expected error for traversal attempt %q", bad)
		}
	}
}

func TestListPath(t *testing.T) {
	r := New(testFS())

	entries, listed, err := r.ListPath("alpha")
	if err != nil {
		t.Fatalf("ListPath: %v", err)
	}
	if listed != "alpha" {
		t.Fatalf("expected listed=alpha, got %q", listed)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries (SKILL.md, references), got %+v", entries)
	}

	entries, listed, err = r.ListPath("alpha/references")
	if err != nil {
		t.Fatalf("ListPath subdir: %v", err)
	}
	if listed != "alpha/references" || len(entries) != 1 || entries[0].Path != "alpha/references/notes.md" {
		t.Fatalf("unexpected result: listed=%q entries=%+v", listed, entries)
	}

	if _, _, err := r.ListPath("alpha/references/notes.md"); err == nil {
		t.Fatal("expected error listing a file as if it were a directory")
	}

	if _, _, err := r.ListPath("nope"); err == nil {
		t.Fatal("expected error for unknown skill")
	}
}

func TestSplitArg(t *testing.T) {
	cases := []struct {
		in, name, rest string
	}{
		{"alpha", "alpha", ""},
		{"alpha/references/notes.md", "alpha", "references/notes.md"},
	}
	for _, c := range cases {
		name, rest := SplitArg(c.in)
		if name != c.name || rest != c.rest {
			t.Errorf("SplitArg(%q) = (%q, %q), want (%q, %q)", c.in, name, rest, c.name, c.rest)
		}
	}
}
