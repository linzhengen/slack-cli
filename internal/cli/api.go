package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/linzhengen/slack-cli/internal/cliutil"
	"github.com/linzhengen/slack-cli/internal/slackapi"
)

// newAPICmd builds the `api` command group: a generic escape hatch that can
// call any Slack Web API method by name, plus self-describing helpers
// (`list`, `describe`) an AI agent can use to discover what's available
// without reading Slack's docs.
func newAPICmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "api",
		Short: "Call any Slack Web API method directly, and discover available methods",
	}
	cmd.AddCommand(newAPICallCmd(), newAPIListCmd(), newAPIDescribeCmd())
	return cmd
}

func newAPICallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "call <method>",
		Short: "Call any Slack Web API method by name (e.g. chat.postMessage)",
		Long: `Call any Slack Web API method by name.

This is the universal fallback: every method the dedicated command tree
covers (and any method it doesn't, including ones added to Slack's API
after this CLI was built) can be called this way.

Parameters are given with repeated --param key=value flags, or as a single
--json object for complex/array fields such as "blocks":

  slack-cli api call chat.postMessage --param channel=C0123 --param text="hello"
  slack-cli api call chat.postMessage --json '{"channel":"C0123","blocks":[...]}'

If the method is known to slack-cli's registry (see "api list"), required
parameters are validated client-side before the request is sent.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			method := args[0]

			var declared []slackapi.Param
			if meth, ok := slackapi.Lookup(method); ok {
				declared = meth.Params
			}

			params, err := buildParams(cmd, declared)
			if err != nil {
				return fail(cmd, method, err.Error(), nil)
			}

			if missing := missingRequired(declared, params); len(missing) > 0 {
				return fail(cmd, method, fmt.Sprintf("missing required parameter(s): %v", missing), nil)
			}

			return performCall(cmd, method, params)
		},
	}
	addParamEscapeHatchFlags(cmd)
	return cmd
}

func newAPIListCmd() *cobra.Command {
	var category string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all Slack Web API methods slack-cli knows about",
		Long: `List all Slack Web API methods slack-cli's registry knows about, as JSON.

This does not limit what "api call" can invoke — it can call any method by
name, known or not — it's a discovery aid, useful for an AI agent deciding
which method and parameters to use.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			all := slackapi.All()
			type entry struct {
				Name       string `json:"name"`
				Desc       string `json:"description"`
				Deprecated string `json:"deprecated,omitempty"`
			}
			out := make([]entry, 0, len(all))
			for _, meth := range all {
				if category != "" && !hasPrefix(meth.Name, category) {
					continue
				}
				out = append(out, entry{Name: meth.Name, Desc: meth.Desc, Deprecated: meth.Deprecated})
			}
			return cliutil.Print(cmd.OutOrStdout(), outputFormat(cmd), out)
		},
	}
	cmd.Flags().StringVar(&category, "category", "", `Only list methods whose name starts with this prefix (e.g. "admin.", "chat")`)
	return cmd
}

func newAPIDescribeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "describe <method>",
		Short: "Show parameters and documentation for a Slack Web API method, as JSON",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			method := args[0]
			meth, ok := slackapi.Lookup(method)
			if !ok {
				return cliutil.Print(cmd.OutOrStdout(), outputFormat(cmd), map[string]any{
					"name":  method,
					"known": false,
					"note":  "not in slack-cli's registry; it can still be called with `api call`, and all parameters you supply will be sent as-is",
				})
			}
			return cliutil.Print(cmd.OutOrStdout(), outputFormat(cmd), map[string]any{
				"name":       meth.Name,
				"known":      true,
				"desc":       meth.Desc,
				"deprecated": meth.Deprecated,
				"params":     meth.Params,
			})
		},
	}
	return cmd
}

func hasPrefix(name, prefix string) bool {
	if len(name) < len(prefix) {
		return false
	}
	return name[:len(prefix)] == prefix
}
