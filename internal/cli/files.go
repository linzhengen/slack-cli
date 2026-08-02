package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/linzhengen/slack-cli/internal/cliutil"
)

// newFilesUploadCmd implements the modern three-step file upload flow
// (files.getUploadURLExternal -> raw multipart POST ->
// files.completeUploadExternal) as a single friendly command. The legacy
// files.upload method is deprecated by Slack and is intentionally not the
// primary path here, though it's still reachable via `api call files.upload`
// for older workspaces/apps.
func newFilesUploadCmd() *cobra.Command {
	var (
		path           string
		filename       string
		title          string
		channel        string
		initialComment string
		threadTS       string
		altText        string
		snippetType    string
	)

	cmd := &cobra.Command{
		Use:   "upload",
		Short: "Upload a local file to Slack (getUploadURLExternal + completeUploadExternal)",
		Long: `Uploads a local file using Slack's current upload flow:

  1. files.getUploadURLExternal - reserves an upload URL
  2. a raw multipart POST of the file content to that URL
  3. files.completeUploadExternal - finalizes the upload, optionally sharing
     it to a channel

Equivalent to calling those three methods yourself; use this when you just
want to get a local file into Slack in one step.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if path == "" {
				return fail(cmd, "files.upload", "--file is required", nil)
			}
			f, err := os.Open(path)
			if err != nil {
				return fail(cmd, "files.upload", err.Error(), nil)
			}
			defer f.Close()

			info, err := f.Stat()
			if err != nil {
				return fail(cmd, "files.upload", err.Error(), nil)
			}

			name := filename
			if name == "" {
				name = filepath.Base(path)
			}

			client, err := resolveClient(cmd)
			if err != nil {
				return fail(cmd, "files.upload", err.Error(), nil)
			}
			ctx := cmd.Context()

			getParams := map[string]string{
				"filename": name,
				"length":   fmt.Sprintf("%d", info.Size()),
			}
			if altText != "" {
				getParams["alt_text"] = altText
			}
			if snippetType != "" {
				getParams["snippet_type"] = snippetType
			}

			step1, err := client.Call(ctx, "files.getUploadURLExternal", getParams)
			if err != nil {
				return handleUploadErr(cmd, "files.getUploadURLExternal", err, step1)
			}

			uploadURL, _ := step1["upload_url"].(string)
			fileID, _ := step1["file_id"].(string)
			if uploadURL == "" || fileID == "" {
				return fail(cmd, "files.getUploadURLExternal", "unexpected response: missing upload_url/file_id", step1)
			}

			if err := client.UploadFile(ctx, uploadURL, name, f); err != nil {
				return fail(cmd, "files.upload(raw)", err.Error(), nil)
			}

			fileEntry := map[string]string{"id": fileID}
			if title != "" {
				fileEntry["title"] = title
			}
			filesJSON, err := json.Marshal([]map[string]string{fileEntry})
			if err != nil {
				return fail(cmd, "files.completeUploadExternal", err.Error(), nil)
			}

			completeParams := map[string]string{"files": string(filesJSON)}
			if channel != "" {
				completeParams["channel_id"] = channel
			}
			if initialComment != "" {
				completeParams["initial_comment"] = initialComment
			}
			if threadTS != "" {
				completeParams["thread_ts"] = threadTS
			}

			step3, err := client.Call(ctx, "files.completeUploadExternal", completeParams)
			if err != nil {
				return handleUploadErr(cmd, "files.completeUploadExternal", err, step3)
			}

			return cliutil.Print(cmd.OutOrStdout(), outputFormat(cmd), step3)
		},
	}

	cmd.Flags().StringVar(&path, "file", "", "Path to the local file to upload (required)")
	cmd.Flags().StringVar(&filename, "filename", "", "Filename to use in Slack (default: base name of --file)")
	cmd.Flags().StringVar(&title, "title", "", "Title for the uploaded file")
	cmd.Flags().StringVar(&channel, "channel", "", "Channel id to share the file to")
	cmd.Flags().StringVar(&initialComment, "initial-comment", "", "Comment to post along with the file")
	cmd.Flags().StringVar(&threadTS, "thread-ts", "", "Thread timestamp to reply in, if sharing to a channel")
	cmd.Flags().StringVar(&altText, "alt-text", "", "Alt text, for image files")
	cmd.Flags().StringVar(&snippetType, "snippet-type", "", "Syntax highlighting language, for snippet files")
	return cmd
}

func handleUploadErr(cmd *cobra.Command, method string, err error, resp map[string]any) error {
	if resp != nil {
		_ = cliutil.Print(cmd.ErrOrStderr(), outputFormat(cmd), resp)
		return errSilent
	}
	return fail(cmd, method, err.Error(), nil)
}
