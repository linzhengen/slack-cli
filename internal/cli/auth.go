package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/linzhengen/slack-cli/internal/cliutil"
	"github.com/linzhengen/slack-cli/internal/config"
)

func newAuthCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage stored Slack auth profiles",
		Long: `Manage named Slack auth profiles stored at ~/.config/slack-cli/config.yaml.

A profile is only needed for convenience: every command also accepts
--token directly, or reads SLACK_CLI_TOKEN / SLACK_TOKEN from the
environment, which take priority over stored profiles and are usually the
better choice for AI agents and CI.`,
	}
	cmd.AddCommand(
		newAuthLoginCmd(),
		newAuthLogoutCmd(),
		newAuthListCmd(),
		newAuthUseCmd(),
		newAuthWhoamiCmd(),
	)
	return cmd
}

func newAuthLoginCmd() *cobra.Command {
	var name, team, baseURL string
	var activate bool
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Store a Slack token as a named profile",
		Long: `Store a Slack token as a named profile.

Slack tokens start with xoxb- (bot), xoxp- (user), or xoxa-/xoxr- (app-level
/ refresh). Create one at https://api.slack.com/apps by adding OAuth
scopes to an app and installing it to a workspace.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			token, _ := cmd.Flags().GetString(flagToken)
			if token == "" {
				return fail(cmd, "", "pass the token to store with --token", nil)
			}
			cfg, err := config.Load()
			if err != nil {
				return fail(cmd, "", err.Error(), nil)
			}
			cfg.SetProfile(name, config.Profile{Token: token, Team: team, BaseURL: baseURL}, activate)
			if err := cfg.Save(); err != nil {
				return fail(cmd, "", err.Error(), nil)
			}
			path, _ := config.Path()
			return cliutil.Print(cmd.OutOrStdout(), outputFormat(cmd), map[string]any{
				"ok":      true,
				"profile": name,
				"active":  cfg.Active == name,
				"path":    path,
			})
		},
	}
	cmd.Flags().StringVar(&name, "name", "default", "Profile name to store the token under")
	cmd.Flags().StringVar(&team, "team", "", "Optional label for the workspace this token belongs to")
	cmd.Flags().StringVar(&baseURL, "base-url-override", "", "Optional API base URL to always use with this profile")
	cmd.Flags().BoolVar(&activate, "activate", true, "Make this the active profile")
	return cmd
}

func newAuthLogoutCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logout <name>",
		Short: "Remove a stored profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return fail(cmd, "", err.Error(), nil)
			}
			name := args[0]
			if _, ok := cfg.Profiles[name]; !ok {
				return fail(cmd, "", fmt.Sprintf("no such profile %q", name), nil)
			}
			cfg.RemoveProfile(name)
			if err := cfg.Save(); err != nil {
				return fail(cmd, "", err.Error(), nil)
			}
			return cliutil.Print(cmd.OutOrStdout(), outputFormat(cmd), map[string]any{"ok": true, "removed": name})
		},
	}
	return cmd
}

func newAuthListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List stored profiles",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return fail(cmd, "", err.Error(), nil)
			}
			type entry struct {
				Name   string `json:"name"`
				Team   string `json:"team,omitempty"`
				Active bool   `json:"active"`
				Token  string `json:"token"` // masked
			}
			out := make([]entry, 0, len(cfg.Profiles))
			for name, prof := range cfg.Profiles {
				out = append(out, entry{Name: name, Team: prof.Team, Active: name == cfg.Active, Token: maskToken(prof.Token)})
			}
			return cliutil.Print(cmd.OutOrStdout(), outputFormat(cmd), out)
		},
	}
	return cmd
}

func newAuthUseCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "use <name>",
		Short: "Set the active profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return fail(cmd, "", err.Error(), nil)
			}
			name := args[0]
			if _, ok := cfg.Profiles[name]; !ok {
				return fail(cmd, "", fmt.Sprintf("no such profile %q", name), nil)
			}
			cfg.Active = name
			if err := cfg.Save(); err != nil {
				return fail(cmd, "", err.Error(), nil)
			}
			return cliutil.Print(cmd.OutOrStdout(), outputFormat(cmd), map[string]any{"ok": true, "active": name})
		},
	}
	return cmd
}

func newAuthWhoamiCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "whoami",
		Short: "Show identity information for the resolved token (calls auth.test)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return performCall(cmd, "auth.test", map[string]string{})
		},
	}
	return cmd
}

func maskToken(t string) string {
	if len(t) <= 8 {
		return "****"
	}
	return t[:8] + "…" + fmt.Sprintf("(%d chars)", len(t))
}
