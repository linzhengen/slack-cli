package cli

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode"

	"github.com/spf13/cobra"

	"github.com/linzhengen/slack-cli/internal/slackapi"
)

// addParamEscapeHatchFlags adds --param and --json to cmd. Every command
// that calls the Slack API exposes these, so arbitrary or newly-added
// fields can always be sent even without a dedicated named flag.
func addParamEscapeHatchFlags(cmd *cobra.Command) {
	cmd.Flags().StringArray("param", nil, "Extra parameter as key=value (repeatable)")
	cmd.Flags().String("json", "", "Extra parameters as a JSON object, merged in last (highest priority)")
}

// buildParams assembles the Slack API parameter map for a call from three
// layers, applied in increasing priority: named flags for declared params,
// then repeated --param key=value flags, then a single --json object. Later
// layers overwrite earlier ones, so --json is always the final override.
func buildParams(cmd *cobra.Command, declared []slackapi.Param) (map[string]string, error) {
	params := map[string]string{}

	for _, prm := range declared {
		if !cmd.Flags().Changed(prm.Name) {
			continue
		}
		v, err := cmd.Flags().GetString(prm.Name)
		if err != nil {
			continue
		}
		params[prm.Name] = v
	}

	extra, _ := cmd.Flags().GetStringArray("param")
	for _, kv := range extra {
		k, v, ok := strings.Cut(kv, "=")
		if !ok {
			return nil, fmt.Errorf("invalid --param %q: expected key=value", kv)
		}
		params[k] = v
	}

	jsonStr, _ := cmd.Flags().GetString("json")
	if jsonStr != "" {
		var obj map[string]any
		if err := json.Unmarshal([]byte(jsonStr), &obj); err != nil {
			return nil, fmt.Errorf("invalid --json: %w", err)
		}
		for k, v := range obj {
			s, err := toParamString(v)
			if err != nil {
				return nil, fmt.Errorf("--json field %q: %w", k, err)
			}
			params[k] = s
		}
	}

	return params, nil
}

// toParamString renders a decoded JSON value as the string Slack's
// form-encoded Web API expects: plain strings pass through unchanged (no
// extra quoting), everything else (bool, number, array, object) is
// re-encoded as a JSON string, which is exactly how Slack wants complex
// fields like "blocks" or "attachments" when sent form-encoded.
func toParamString(v any) (string, error) {
	if s, ok := v.(string); ok {
		return s, nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// missingRequired returns the names of any declared required params not
// present in params, so we can fail fast client-side with a clear message
// instead of round-tripping to Slack for an error it already documents.
func missingRequired(declared []slackapi.Param, params map[string]string) []string {
	var missing []string
	for _, prm := range declared {
		if !prm.Required {
			continue
		}
		if _, ok := params[prm.Name]; !ok {
			missing = append(missing, prm.Name)
		}
	}
	return missing
}

// kebab converts a camelCase or PascalCase API segment (e.g. "postMessage",
// "authPolicy", "getUploadURLExternal") into a kebab-case CLI token
// ("post-message", "auth-policy", "get-upload-url-external"). Runs of
// uppercase letters are treated as a single acronym: a hyphen is only
// inserted where a new word starts, i.e. before an uppercase letter that
// follows a lowercase/digit, or before the last letter of an acronym when
// it's immediately followed by a lowercase letter. Segments that are
// already lowercase (the common case: "chat", "conversations", "admin")
// pass through unchanged.
func kebab(s string) string {
	runes := []rune(s)
	var b strings.Builder
	for i, r := range runes {
		if unicode.IsUpper(r) && i > 0 {
			prev := runes[i-1]
			startsNewWord := unicode.IsLower(prev) || unicode.IsDigit(prev)
			endsAcronymBeforeWord := unicode.IsUpper(prev) && i+1 < len(runes) && unicode.IsLower(runes[i+1])
			if startsNewWord || endsAcronymBeforeWord {
				b.WriteByte('-')
			}
		}
		b.WriteRune(unicode.ToLower(r))
	}
	return b.String()
}
