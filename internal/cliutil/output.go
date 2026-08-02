// Package cliutil holds small helpers shared across CLI commands:
// output formatting and structured error printing, both designed to be
// easy for a script or an AI agent to parse reliably.
package cliutil

import (
	"encoding/json"
	"fmt"
	"io"
)

// Format selects how a value is written to stdout.
type Format string

const (
	FormatPretty  Format = "pretty"  // multi-line, 2-space indented JSON (default)
	FormatCompact Format = "compact" // single-line JSON, best for piping to jq/agents
)

// Print writes v as JSON to w in the requested format.
func Print(w io.Writer, format Format, v any) error {
	switch format {
	case FormatCompact:
		enc := json.NewEncoder(w)
		return enc.Encode(v)
	case FormatPretty, "":
		data, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(w, string(data))
		return err
	default:
		return fmt.Errorf("unknown output format %q (want pretty or compact)", format)
	}
}

// ErrorPayload is the shape written to stderr on failure, kept stable so
// scripts and agents can rely on it.
type ErrorPayload struct {
	Ok     bool   `json:"ok"`
	Error  string `json:"error"`
	Method string `json:"method,omitempty"`
	Detail any    `json:"detail,omitempty"`
}

// PrintError writes a structured error to w.
func PrintError(w io.Writer, format Format, method, message string, detail any) error {
	return Print(w, format, ErrorPayload{Ok: false, Error: message, Method: method, Detail: detail})
}
