// Command slack-cli is a command-line client for the Slack Web API.
package main

import (
	"context"
	"fmt"
	"os"

	skillsfs "github.com/linzhengen/slack-cli"
	"github.com/linzhengen/slack-cli/internal/cli"
)

// version is set via -ldflags "-X main.version=..." at release build time.
var version = "dev"

func main() {
	cli.Version = version
	cli.SetSkillContent(skillsfs.FS)

	root := cli.NewRootCmd()
	ctx := context.Background()

	if err := root.ExecuteContext(ctx); err != nil {
		if err.Error() != "" {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(1)
	}
}
