// Package skillcontent reads agent skill content (SKILL.md files and their
// reference material) out of an injected fs.FS rooted at the skill list,
// e.g. entries like "chat/SKILL.md" or "chat/references/blocks.md".
//
// The fs.FS is normally the skills/ directory embedded into the binary at
// build time (see the root skillsfs package), so `slack-cli skills` always
// reflects exactly the CLI version that's running it, with no separate
// install or copy-of-the-repo step required.
package skillcontent

import (
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Reader reads skill content out of fsys.
type Reader struct {
	fsys fs.FS
}

// New returns a Reader over fsys, which should have one directory per skill
// at its root (each containing at least a SKILL.md).
func New(fsys fs.FS) *Reader { return &Reader{fsys: fsys} }

// SkillInfo is the frontmatter summary of one skill, as returned by List.
type SkillInfo struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Version     string         `json:"version,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// DirEntry.Path is skill-prefixed (e.g. "chat/references/blocks.md") so it
// can be fed straight back into ReadReference/ReadSkill.
type DirEntry struct {
	Path  string `json:"path"`
	IsDir bool   `json:"is_dir"`
}

// List returns every skill's frontmatter summary, sorted by name.
func (r *Reader) List() ([]SkillInfo, error) {
	entries, err := fs.ReadDir(r.fsys, ".")
	if err != nil {
		return nil, fmt.Errorf("reading embedded skills: %w", err)
	}
	out := make([]SkillInfo, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		// Skip dirs that aren't real skills (no SKILL.md).
		if info, ok := r.skillInfo(e.Name()); ok {
			out = append(out, info)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (r *Reader) skillInfo(name string) (SkillInfo, bool) {
	data, err := fs.ReadFile(r.fsys, name+"/SKILL.md")
	if err != nil {
		return SkillInfo{}, false
	}
	desc, version, metadata := parseFrontmatter(data)
	return SkillInfo{Name: name, Description: desc, Version: version, Metadata: metadata}, true
}

// ListPath lists one directory layer (no recursion) under "<name>" or
// "<name>/<sub>", returning the entries and the cleaned path listed.
func (r *Reader) ListPath(arg string) ([]DirEntry, string, error) {
	name, sub := SplitArg(arg)
	if err := r.ensureSkill(name); err != nil {
		return nil, "", err
	}
	dir := name
	if sub != "" {
		cleaned, err := cleanSubPath(sub)
		if err != nil {
			return nil, "", err
		}
		dir = name + "/" + cleaned
		info, err := fs.Stat(r.fsys, dir)
		if err != nil {
			return nil, "", fmt.Errorf("path %q not found in skill %q (run `slack-cli skills list %s` to see files in this skill)", sub, name, name)
		}
		if !info.IsDir() {
			return nil, "", fmt.Errorf("path %q is a file, not a directory; use `slack-cli skills read %s/%s` to read it", sub, name, cleaned)
		}
	}
	entries, err := fs.ReadDir(r.fsys, dir)
	if err != nil {
		return nil, "", fmt.Errorf("reading embedded skill content: %w", err)
	}
	out := make([]DirEntry, 0, len(entries))
	for _, e := range entries {
		out = append(out, DirEntry{Path: dir + "/" + e.Name(), IsDir: e.IsDir()})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, dir, nil
}

// SplitArg splits "<name>/<rest>" at the first separator; an argument with
// no separator is a bare skill name (rest "").
func SplitArg(arg string) (name, rest string) {
	name, rest, _ = strings.Cut(arg, "/")
	return name, rest
}

// parseFrontmatter best-effort-extracts the frontmatter fields; missing or
// unparseable frontmatter yields ("", "", nil), never an error.
func parseFrontmatter(skillMD []byte) (description, version string, metadata map[string]any) {
	lines := strings.Split(string(skillMD), "\n")
	if len(lines) == 0 || strings.TrimRight(lines[0], "\r") != "---" {
		return "", "", nil
	}
	block := make([]string, 0, len(lines))
	closed := false
	for _, ln := range lines[1:] {
		if strings.TrimRight(ln, "\r") == "---" {
			closed = true
			break
		}
		block = append(block, ln)
	}
	if !closed {
		return "", "", nil
	}
	var fm struct {
		Description string         `yaml:"description"`
		Version     string         `yaml:"version"`
		Metadata    map[string]any `yaml:"metadata"`
	}
	if err := yaml.Unmarshal([]byte(strings.Join(block, "\n")), &fm); err != nil {
		return "", "", nil
	}
	return fm.Description, fm.Version, fm.Metadata
}

// ReadSkill returns the raw bytes of <name>/SKILL.md.
func (r *Reader) ReadSkill(name string) ([]byte, error) {
	if err := r.ensureSkill(name); err != nil {
		return nil, err
	}
	data, err := fs.ReadFile(r.fsys, name+"/SKILL.md")
	if err != nil {
		return nil, fmt.Errorf("reading embedded skill content: %w", err)
	}
	return data, nil
}

func (r *Reader) ensureSkill(name string) error {
	if name == "" || strings.ContainsAny(name, `/\`) || name == "." || name == ".." {
		return unknownSkill(name)
	}
	info, err := fs.Stat(r.fsys, name)
	if err != nil || !info.IsDir() {
		return unknownSkill(name)
	}
	return nil
}

func unknownSkill(name string) error {
	return fmt.Errorf("unknown skill %q (run `slack-cli skills list` to see available skills)", name)
}

// cleanSubPath returns the cleaned form of relpath, rejecting absolute paths
// and ".." escapes. relpath must be non-empty (callers handle the
// skill-root case separately).
func cleanSubPath(relpath string) (string, error) {
	cleaned := path.Clean(relpath)
	// path.Clean only treats '/' as a separator, so a Windows-style "..\"
	// prefix survives; reject it explicitly alongside "../".
	if relpath == "" || path.IsAbs(relpath) || cleaned == "." ||
		cleaned == ".." || strings.HasPrefix(cleaned, "../") || strings.HasPrefix(cleaned, `..\`) {
		return "", fmt.Errorf("invalid path %q: must be a relative path without '..'", relpath)
	}
	return cleaned, nil
}

// ReadReference returns the bytes of <name>/<relpath> and the cleaned path.
func (r *Reader) ReadReference(name, relpath string) ([]byte, string, error) {
	if err := r.ensureSkill(name); err != nil {
		return nil, "", err
	}
	cleaned, err := cleanSubPath(relpath)
	if err != nil {
		return nil, "", err
	}
	full := name + "/" + cleaned
	info, err := fs.Stat(r.fsys, full)
	if err != nil {
		return nil, "", fmt.Errorf("reference %q not found in skill %q (run `slack-cli skills list %s` to see files in this skill)", relpath, name, name)
	}
	if info.IsDir() {
		return nil, "", fmt.Errorf("reference %q is a directory, not a file", relpath)
	}
	data, err := fs.ReadFile(r.fsys, full)
	if err != nil {
		return nil, "", fmt.Errorf("reading embedded skill content: %w", err)
	}
	return data, cleaned, nil
}
