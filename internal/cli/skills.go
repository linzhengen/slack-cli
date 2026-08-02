package cli

import (
	"fmt"
	"io/fs"

	"github.com/spf13/cobra"

	"github.com/linzhengen/slack-cli/internal/cliutil"
	"github.com/linzhengen/slack-cli/internal/skillcontent"
)

// skillFS holds the embedded skills/ content. main wires it in via
// SetSkillContent, before NewRootCmd is called, from a root-level go:embed
// (see the skillsfs package) — go:embed can't reach outside the directory
// tree of the file that declares it, and skills/ lives at the module root,
// not under internal/cli. It stays nil in any build that skips that wiring
// (e.g. `go build ./main.go`-style single-file builds, or tests), in which
// case "skills" reports a clear error instead of panicking.
var skillFS fs.FS

// SetSkillContent wires the embedded skills/ filesystem into the CLI. Call
// this before NewRootCmd.
func SetSkillContent(fsys fs.FS) {
	skillFS = fsys
}

func newSkillsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "skills",
		Short: "Read agent skill docs embedded in the CLI binary (list / read)",
		Long: `Read agent-readable skill content (SKILL.md and reference files) embedded
in the slack-cli binary at build time, so it always matches the installed
CLI's version — no separate install step, and no copy of this repo needed
on disk.

Each skill under skills/ documents one area of the Slack API (messaging,
channels, files, search, ...) for an AI agent: what it's for, which
slack-cli commands to reach for, the OAuth scopes it needs, and worked
examples. This is also how skills stay installable the same way
larksuite/cli's are: the skills/ directory at the repo root follows the
standard Agent Skills layout (a SKILL.md per skill directory), so tools
like "npx skills add <owner>/slack-cli" that materialize skills onto disk
for other agent runtimes work against this repo unmodified. "slack-cli
skills" is the zero-install path for agents that can just run a command.`,
	}
	cmd.AddCommand(newSkillsListCmd(), newSkillsReadCmd())
	return cmd
}

func skillReader() (*skillcontent.Reader, error) {
	if skillFS == nil {
		return nil, fmt.Errorf("skill content not embedded in this build")
	}
	return skillcontent.New(skillFS), nil
}

type skillListEnvelope struct {
	OK     bool                     `json:"ok"`
	Skills []skillcontent.SkillInfo `json:"skills"`
	Count  int                      `json:"count"`
}

type skillListPathEnvelope struct {
	OK      bool                    `json:"ok"`
	Path    string                  `json:"path"`
	Entries []skillcontent.DirEntry `json:"entries"`
	Count   int                     `json:"count"`
}

func newSkillsListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list [name[/path]]",
		Short: "List skills, or list one layer under a skill path (like ls)",
		Example: `  slack-cli skills list                            # all skills: name, description, version
  slack-cli skills list slack-messaging             # one layer under a skill (like ls)
  slack-cli skills list slack-messaging/references   # one layer under a subdirectory`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := skillReader()
			if err != nil {
				return fail(cmd, "skills.list", err.Error(), nil)
			}
			if len(args) == 0 {
				skills, err := r.List()
				if err != nil {
					return fail(cmd, "skills.list", err.Error(), nil)
				}
				return cliutil.Print(cmd.OutOrStdout(), outputFormat(cmd), skillListEnvelope{OK: true, Skills: skills, Count: len(skills)})
			}
			entries, listed, err := r.ListPath(args[0])
			if err != nil {
				return fail(cmd, "skills.list", err.Error(), nil)
			}
			return cliutil.Print(cmd.OutOrStdout(), outputFormat(cmd), skillListPathEnvelope{OK: true, Path: listed, Entries: entries, Count: len(entries)})
		},
	}
	return cmd
}

type skillReadEnvelope struct {
	Skill    string `json:"skill"`
	Path     string `json:"path"`
	Content  string `json:"content"`
	Guidance string `json:"guidance,omitempty"`
}

func newSkillsReadCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "read <name>[/<path>] [path]",
		Short: "Print a skill's SKILL.md, or a file under the skill (raw markdown by default)",
		Example: `  slack-cli skills read slack-messaging                               # the skill's SKILL.md
  slack-cli skills read slack-messaging references/blocks.md          # a file under the skill
  slack-cli skills read slack-messaging/references/blocks.md          # same, slash form
  slack-cli skills read slack-messaging --json                        # JSON envelope`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			name, relpath, err := parseSkillsReadTarget(args)
			if err != nil {
				return fail(cmd, "skills.read", err.Error(), nil)
			}
			r, err := skillReader()
			if err != nil {
				return fail(cmd, "skills.read", err.Error(), nil)
			}

			var content []byte
			var pathOut string
			if relpath == "" {
				content, err = r.ReadSkill(name)
				pathOut = "SKILL.md"
			} else {
				content, pathOut, err = r.ReadReference(name, relpath)
			}
			if err != nil {
				return fail(cmd, "skills.read", err.Error(), nil)
			}

			isMain := pathOut == "SKILL.md"
			if asJSON {
				env := skillReadEnvelope{Skill: name, Path: pathOut, Content: string(content)}
				if isMain {
					env.Guidance = skillReadGuidance(name)
				}
				return cliutil.Print(cmd.OutOrStdout(), outputFormat(cmd), env)
			}
			// Raw stdout stays byte-identical to the file; guidance goes to stderr.
			if _, err := cmd.OutOrStdout().Write(content); err != nil {
				return fail(cmd, "skills.read", err.Error(), nil)
			}
			if isMain {
				_, _ = fmt.Fprintln(cmd.ErrOrStderr(), skillReadGuidance(name))
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "output as a JSON envelope instead of raw markdown")
	return cmd
}

// parseSkillsReadTarget maps 1-or-2 positional args to (name, relpath); a
// lone "<a>/<b>" splits on the first '/', and relpath "" reads the main
// SKILL.md.
func parseSkillsReadTarget(args []string) (name, relpath string, err error) {
	switch len(args) {
	case 1:
		name, relpath = skillcontent.SplitArg(args[0])
		return name, relpath, nil
	case 2:
		return args[0], args[1], nil
	default:
		return "", "", fmt.Errorf("read requires 1 or 2 arguments: <name>[/<path>] [path]")
	}
}

// skillReadGuidance routes cross-skill "../slack-foo/..." references back
// through `skills read slack-foo/...`: the path guard rejects a literal
// "../", so the relative form must be rewritten.
func skillReadGuidance(name string) string {
	return fmt.Sprintf("> Tip: read this skill's own files (e.g. `references/...`) with "+
		"`slack-cli skills read %s <relative-path>` to keep them in sync with this CLI version. "+
		"A reference to another skill (`../slack-foo/...`) uses the same command with the "+
		"leading `../` removed: `slack-cli skills read slack-foo/...`.", name)
}
