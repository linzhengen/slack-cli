package skillsfs

import (
	"testing"

	"github.com/linzhengen/slack-cli/internal/skillcontent"
)

// TestEmbeddedSkillsParse exercises the real skills/ content (not a test
// fixture): every SKILL.md under skills/ must have valid frontmatter with a
// non-empty name and description, and reference files (if any) must be
// readable. This is what would catch a YAML typo in an actual skill file.
func TestEmbeddedSkillsParse(t *testing.T) {
	r := skillcontent.New(FS)
	skills, err := r.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(skills) == 0 {
		t.Fatal("expected at least one embedded skill")
	}

	for _, s := range skills {
		if s.Description == "" {
			t.Errorf("skill %q has no description in its frontmatter", s.Name)
		}
		if s.Version == "" {
			t.Errorf("skill %q has no version in its frontmatter", s.Name)
		}

		content, err := r.ReadSkill(s.Name)
		if err != nil {
			t.Errorf("ReadSkill(%q): %v", s.Name, err)
		}
		if len(content) == 0 {
			t.Errorf("skill %q has empty SKILL.md", s.Name)
		}

		entries, _, err := r.ListPath(s.Name)
		if err != nil {
			t.Errorf("ListPath(%q): %v", s.Name, err)
			continue
		}
		for _, e := range entries {
			if e.IsDir || e.Path == s.Name+"/SKILL.md" {
				continue
			}
			t.Errorf("unexpected file %q directly in skill %q (only SKILL.md and subdirectories are expected)", e.Path, s.Name)
		}
	}
}

func TestEmbeddedReferencesReadable(t *testing.T) {
	r := skillcontent.New(FS)
	entries, _, err := r.ListPath("slack-shared/references")
	if err != nil {
		t.Fatalf("ListPath(slack-shared/references): %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("expected slack-shared/references to have at least one file")
	}
	for _, e := range entries {
		if e.IsDir {
			continue
		}
		rel := e.Path[len("slack-shared/"):]
		if _, _, err := r.ReadReference("slack-shared", rel); err != nil {
			t.Errorf("ReadReference(slack-shared, %q): %v", rel, err)
		}
	}
}
