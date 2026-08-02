package cli

import (
	"github.com/spf13/cobra"

	"github.com/linzhengen/slack-cli/internal/cliutil"
	"github.com/linzhengen/slack-cli/internal/slackapi"
)

// performCall resolves credentials, invokes the given Slack Web API method,
// and prints either the JSON response (stdout, exit 0) or a structured
// error (stderr, non-zero exit). It is the single place that decides how
// every command in the tree talks to Slack and reports back.
func performCall(cmd *cobra.Command, method string, params map[string]string) error {
	client, err := resolveClient(cmd)
	if err != nil {
		return fail(cmd, method, err.Error(), nil)
	}

	resp, callErr := client.Call(cmd.Context(), method, params)
	if callErr != nil {
		switch e := callErr.(type) {
		case *slackapi.APIError:
			// Slack answered with ok:false; its response body already is
			// the most useful error description, so print it as-is.
			_ = cliutil.Print(cmd.ErrOrStderr(), outputFormat(cmd), e.Response)
			return errSilent
		case *slackapi.HTTPError:
			return fail(cmd, method, e.Error(), map[string]any{
				"status_code": e.StatusCode,
				"body":        e.Body,
			})
		default:
			return fail(cmd, method, callErr.Error(), nil)
		}
	}

	return cliutil.Print(cmd.OutOrStdout(), outputFormat(cmd), resp)
}
