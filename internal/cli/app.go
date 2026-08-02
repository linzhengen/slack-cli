package cli

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/linzhengen/slack-cli/internal/cliutil"
	"github.com/linzhengen/slack-cli/internal/config"
	"github.com/linzhengen/slack-cli/internal/slackapi"
)

// persistent flag names, shared by root.go and every subcommand that needs
// to resolve credentials or output formatting.
const (
	flagToken   = "token"
	flagProfile = "profile"
	flagBaseURL = "base-url"
	flagOutput  = "output"
	flagTimeout = "timeout"
)

func addPersistentFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().String(flagToken, "", "Slack token (overrides profile/env; env: SLACK_CLI_TOKEN, SLACK_TOKEN)")
	cmd.PersistentFlags().String(flagProfile, "", "Named auth profile to use (default: active profile)")
	cmd.PersistentFlags().String(flagBaseURL, "", "Override the Slack API base URL")
	cmd.PersistentFlags().String(flagOutput, string(cliutil.FormatPretty), "Output format: pretty|compact")
	cmd.PersistentFlags().Duration(flagTimeout, 30*time.Second, "Per-request HTTP timeout")
}

// outputFormat reads the resolved --output flag from cmd or any parent.
func outputFormat(cmd *cobra.Command) cliutil.Format {
	v, _ := cmd.Flags().GetString(flagOutput)
	return cliutil.Format(v)
}

// resolveClient builds a slackapi.Client using, in priority order: the
// --token flag, the SLACK_CLI_TOKEN env var, the SLACK_TOKEN env var, then
// the named (--profile) or active profile from the config file.
func resolveClient(cmd *cobra.Command) (*slackapi.Client, error) {
	token, _ := cmd.Flags().GetString(flagToken)
	baseURL, _ := cmd.Flags().GetString(flagBaseURL)
	profileName, _ := cmd.Flags().GetString(flagProfile)
	timeout, _ := cmd.Flags().GetDuration(flagTimeout)

	if token == "" {
		token = os.Getenv("SLACK_CLI_TOKEN")
	}
	if token == "" {
		token = os.Getenv("SLACK_TOKEN")
	}

	if token == "" {
		cfg, err := config.Load()
		if err != nil {
			return nil, fmt.Errorf("loading config: %w", err)
		}
		prof, ok := cfg.Get(profileName)
		if !ok {
			if profileName != "" {
				return nil, fmt.Errorf("no such profile %q; run `slack-cli auth list`", profileName)
			}
			return nil, fmt.Errorf("no Slack token available: pass --token, set SLACK_CLI_TOKEN, or run `slack-cli auth login`")
		}
		token = prof.Token
		if baseURL == "" {
			baseURL = prof.BaseURL
		}
	}

	client := slackapi.New(token, baseURL)
	client.HTTPClient.Timeout = timeout
	return client, nil
}

// fail prints a structured error to stderr and returns an error that makes
// cobra exit non-zero without also printing cobra's own usage/error text
// (each subcommand sets SilenceUsage/SilenceErrors and calls this instead).
func fail(cmd *cobra.Command, method, message string, detail any) error {
	_ = cliutil.PrintError(cmd.ErrOrStderr(), outputFormat(cmd), method, message, detail)
	return errSilent
}

// errSilent is returned by RunE implementations after they've already
// printed a structured error, so cobra doesn't print a second, differently
// shaped one.
var errSilent = &silentError{}

type silentError struct{}

func (*silentError) Error() string { return "" }
