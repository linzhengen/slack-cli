package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/linzhengen/slack-cli/internal/slackapi"
)

// buildGeneratedCommands turns slackapi.All() into a nested cobra command
// tree, e.g. method "admin.users.session.reset" becomes
// `slack-cli admin users session reset`, and "chat.postMessage" becomes
// `slack-cli chat post-message`. It returns the top-level commands to
// attach to root, plus every intermediate group command keyed by its
// dotted API path (e.g. "admin.users"), so other files (like the
// hand-written files-upload helper) can attach extra children to a group
// the registry also populates.
func buildGeneratedCommands() ([]*cobra.Command, map[string]*cobra.Command) {
	groups := map[string]*cobra.Command{}
	var top []*cobra.Command

	for _, meth := range slackapi.All() {
		segments := strings.Split(meth.Name, ".")
		groupPath := segments[:len(segments)-1]

		var parent *cobra.Command
		var pathKey string
		for _, seg := range groupPath {
			if pathKey == "" {
				pathKey = seg
			} else {
				pathKey = pathKey + "." + seg
			}
			node, ok := groups[pathKey]
			if !ok {
				node = &cobra.Command{
					Use:   kebab(seg),
					Short: fmt.Sprintf("%s.* methods", pathKey),
				}
				groups[pathKey] = node
				if parent == nil {
					top = append(top, node)
				} else {
					parent.AddCommand(node)
				}
			}
			parent = node
		}

		leaf := buildLeafCommand(meth)
		if parent == nil {
			// No Slack Web API method is dot-free, but keep this safe.
			top = append(top, leaf)
		} else {
			parent.AddCommand(leaf)
		}
	}

	return top, groups
}

func buildLeafCommand(meth slackapi.Method) *cobra.Command {
	segments := strings.Split(meth.Name, ".")
	action := segments[len(segments)-1]

	short := meth.Desc
	if meth.Deprecated != "" {
		short = fmt.Sprintf("%s (deprecated: use %s)", short, meth.Deprecated)
	}

	cmd := &cobra.Command{
		Use:   kebab(action),
		Short: short,
		Long:  buildLongHelp(meth),
		RunE: func(cmd *cobra.Command, args []string) error {
			params, err := buildParams(cmd, meth.Params)
			if err != nil {
				return fail(cmd, meth.Name, err.Error(), nil)
			}
			if missing := missingRequired(meth.Params, params); len(missing) > 0 {
				return fail(cmd, meth.Name, fmt.Sprintf("missing required parameter(s): %v", missing), nil)
			}
			return performCall(cmd, meth.Name, params)
		},
	}

	for _, prm := range meth.Params {
		desc := prm.Desc
		if prm.Required {
			desc += " (required)"
		}
		cmd.Flags().String(prm.Name, "", desc)
	}
	addParamEscapeHatchFlags(cmd)

	return cmd
}

func buildLongHelp(meth slackapi.Method) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Calls the Slack Web API method %q.\n\n%s", meth.Name, meth.Desc)
	if meth.Deprecated != "" {
		fmt.Fprintf(&b, "\n\nDeprecated: use %s instead.", meth.Deprecated)
	}
	fmt.Fprint(&b, "\n\nUse --param key=value or --json '{...}' for fields not listed as flags above.")
	return b.String()
}
