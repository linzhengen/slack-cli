---
name: slack-files
version: 1.0.0
description: "Upload local files into Slack and share them to a channel, list/search team files, get file info, delete files, and manage public sharing links. Use when the user asks to upload, attach, or share a file (image, document, CSV, log, etc.) to Slack, find a previously shared file, or revoke/enable a public link for one."
metadata:
  requires:
    bins: ["slack-cli"]
  cliHelp: "slack-cli files --help"
---

# slack-files: uploads, listing, and sharing

Read `slack-shared` first for auth and output/error conventions.

## Core commands

| Task | Command |
|---|---|
| Upload a local file and share it | `slack-cli files upload --file ./report.csv --channel C0123 --title "Weekly report"` |
| Upload without sharing to a channel yet | `slack-cli files upload --file ./notes.txt` |
| Upload as a reply in a thread | add `--thread-ts <parent ts>` (with `--channel`) |
| Get file info | `slack-cli files info --file F0123456` |
| List/filter team files | `slack-cli files list --channel C0123 --types images,pdfs` |
| Delete a file | `slack-cli files delete --file F0123456` |
| Enable a public link | `slack-cli files shared-public-url --file F0123456` |
| Revoke a public link | `slack-cli files revoke-public-url --file F0123456` |

## Key concepts

- **Use `slack-cli files upload` for anything on local disk.** It wraps
  Slack's current 3-step upload flow (`files.getUploadURLExternal` → raw
  POST → `files.completeUploadExternal`) as one command — don't call those
  three methods by hand unless you specifically need to interleave other
  work between the steps. The legacy single-call `files.upload` method is
  deprecated by Slack and not the primary path here, though still reachable
  via `slack-cli api call files.upload` for older workspaces if truly
  needed.
- Files aren't posted "into" a channel like a message — `--channel`
  attaches/shares the already-uploaded file to that channel's feed, and
  `--initial-comment` (via `--param initial_comment=...` or as a plain
  `chat.postMessage` afterward) adds context around it.
- To attach a file that already exists remotely (e.g. a Google Drive doc)
  rather than uploading bytes, use `files.remote.*` (`slack-cli api call
  files.remote.add ...`) instead of `files upload`.
- Public links (`files.sharedPublicURL`) are workspace-policy-gated — some
  workspaces disable external file sharing entirely, in which case this
  call fails with `public_url_disabled` (not a bug, and not overridable
  from a single-workspace token — that's an `admin.*` / workspace settings
  change).

## Common OAuth scopes

- `files:write` — upload, delete, revoke/enable public URL
- `files:read` — info, list

## Examples

Upload a CSV and post a comment introducing it, in one flow:

```sh
slack-cli files upload \
  --file ./weekly-report.csv \
  --channel C0123456 \
  --title "Weekly report" \
  --initial-comment "Here's this week's numbers"
```

## Related

- `slack-shared` — auth, output/error format
- `slack-messaging` — posting text alongside or instead of a file
