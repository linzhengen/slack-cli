// Package skillsfs embeds skills/ (each skill's SKILL.md and references/)
// into the compiled slack-cli binary, so `slack-cli skills` always matches
// the installed CLI's version, with no separate install or copy-of-the-repo
// step needed.
//
// This file has to live at the module root, alongside skills/, because the
// embed directive below can't reach outside the directory of the file that
// declares it — patterns may not contain ".." path elements. slack-cli's
// main package lives in cmd/slack-cli instead of at the root (so `go
// install .../cmd/slack-cli@latest` names an unambiguous package), which
// is why this embed is its own small root-level package rather than living
// directly in main, the way a root-level main.go could do it.
package skillsfs

import (
	"embed"
	"io/fs"
)

// embedded holds the raw embed; each pattern must independently match at
// least one path or the build fails, so at least one skill under skills/
// must have a references/ directory.
//
//go:embed skills/*/SKILL.md skills/*/references
var embedded embed.FS

// FS is skills/, rooted so entries read like "chat/SKILL.md" instead of
// "skills/chat/SKILL.md".
var FS fs.FS = mustSub(embedded, "skills")

func mustSub(f embed.FS, dir string) fs.FS {
	sub, err := fs.Sub(f, dir)
	if err != nil {
		// Programmer error: skills/ must exist at build time for this
		// package to compile at all, so Sub can't fail here in practice.
		panic(err)
	}
	return sub
}
