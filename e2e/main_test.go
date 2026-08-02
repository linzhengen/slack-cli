//go:build e2e

// Package e2e is a true black-box test suite: it builds the actual
// slack-cli binary once and execs it as a subprocess against a fake Slack
// API server (internal/slacktest) for each test, asserting on stdout,
// stderr, and the process exit code exactly as a real caller would see
// them.
//
// This is deliberately separate from the in-process cobra tests in
// internal/cli (which call root.Execute() directly): those are fast and
// good for command-tree/flag-parsing coverage, but they never exercise
// main.go, the compiled binary's go:embed skill content, or real process
// exit codes. This package does, at the cost of a `go build` per test run
// (once, in TestMain) instead of zero — hence the build tag: it's opt-in
// via `make e2e` / `go test -tags=e2e ./e2e/...`, not part of the default
// `go test ./...` / `make test`.
package e2e

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/linzhengen/slack-cli/internal/slacktest"
)

// defaultTestToken is the token runCLI passes by default. Tests that care
// about auth failure set srv.Token to something else first, so the
// mismatch is exercised deliberately rather than by accident.
const defaultTestToken = "xoxb-e2e-test"

// binPath is the compiled slack-cli binary under test, built once in
// TestMain.
var binPath string

func TestMain(m *testing.M) {
	os.Exit(run(m))
}

func run(m *testing.M) int {
	tmpDir, err := os.MkdirTemp("", "slack-cli-e2e-*")
	if err != nil {
		fmt.Fprintln(os.Stderr, "e2e: creating temp dir:", err)
		return 1
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	binPath = filepath.Join(tmpDir, "slack-cli")

	// This package lives at <module root>/e2e, so the module root — where
	// `go build ./cmd/slack-cli` needs to run — is always its parent.
	build := exec.Command("go", "build", "-o", binPath, "./cmd/slack-cli")
	build.Dir = ".."
	var stderr bytes.Buffer
	build.Stderr = &stderr
	if err := build.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "e2e: building slack-cli: %v\n%s", err, stderr.String())
		return 1
	}

	return m.Run()
}

// cliResult is what a real caller of slack-cli sees: the two output
// streams, kept separate exactly like they'd be over a pipe, and the exit
// code.
type cliResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// runCLI execs the built slack-cli binary against srv, with a token srv
// will accept by default (tests that care about auth failure set
// srv.Token to something else first).
func runCLI(t *testing.T, srv *slacktest.Server, args ...string) cliResult {
	t.Helper()
	full := append([]string{"--base-url", srv.URL, "--token", defaultTestToken}, args...)
	cmd := exec.Command(binPath, full...)
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	exitCode := 0
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
		} else {
			t.Fatalf("running slack-cli: %v", err)
		}
	}
	return cliResult{Stdout: outBuf.String(), Stderr: errBuf.String(), ExitCode: exitCode}
}
