// Package cli wires together slack-cli's cobra command tree: auth/profile
// management, the generic `api` escape hatch, and the full generated
// command tree covering the Slack Web API.
package cli

import (
	"github.com/spf13/cobra"
)

// Version is set by main via ldflags at build time.
var Version = "dev"

// NewRootCmd builds the top-level slack-cli command.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "slack-cli",
		Short: "A Slack Web API client for the terminal and for AI agents",
		Long: `slack-cli is a command-line client for the Slack Web API.

It is built to be equally usable by hand and by AI agents: every command
prints structured JSON to stdout, errors are structured JSON on stderr with
a non-zero exit code, and the full API surface is reachable even for
methods this CLI doesn't have a dedicated command for yet (see
"slack-cli api call --help").

Authentication: run "slack-cli auth login" once, or set SLACK_CLI_TOKEN /
SLACK_TOKEN, or pass --token on any command.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       Version,
	}

	addPersistentFlags(root)

	genTop, groups := buildGeneratedCommands()

	// Some hand-written command groups share a name with a top-level API
	// namespace the registry also generates (api.test lives under "api",
	// auth.test/auth.revoke live under "auth", files.upload's siblings live
	// under "files"). Rather than register two commands with the same
	// name, fold the hand-written children into the generated group node.
	mergeGroup(root, groups, "api", newAPICmd())
	mergeGroup(root, groups, "auth", newAuthCmd())
	mergeGroup(root, groups, "files", filesGroupWrapper())

	root.AddCommand(genTop...)

	return root
}

// mergeGroup attaches custom's subcommands (and help text) onto the
// generated group node at path, if one exists; otherwise custom is added to
// root directly, since there was nothing generated to merge into. genTop/
// groups must already be populated by buildGeneratedCommands before calling
// this, and genTop must not yet have been added to root.
func mergeGroup(root *cobra.Command, groups map[string]*cobra.Command, path string, custom *cobra.Command) {
	node, ok := groups[path]
	if !ok {
		// No generated methods share this namespace; custom stands alone.
		// (Not expected today, but keeps this helper safe if the registry
		// ever loses every method under a namespace slack-cli hand-writes
		// commands for.)
		root.AddCommand(custom)
		return
	}
	if custom.Short != "" {
		node.Short = custom.Short
	}
	if custom.Long != "" {
		node.Long = custom.Long
	}
	for _, child := range custom.Commands() {
		node.AddCommand(child)
	}
}

// filesGroupWrapper adapts the single hand-written files-upload command to
// the mergeGroup(groups, "files", ...) shape used for "api" and "auth".
func filesGroupWrapper() *cobra.Command {
	wrapper := &cobra.Command{Use: "files"}
	wrapper.AddCommand(newFilesUploadCmd())
	return wrapper
}
