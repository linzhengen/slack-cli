---
name: slack-messaging
version: 1.0.0
description: "Send, edit, delete, and schedule Slack messages; reply in threads; react with emoji; pin and bookmark; read channel/thread history. Use when the user asks to send or post a message, reply to a thread, edit or delete something already posted, react with an emoji, pin a message, add a bookmark, or read/summarize recent messages in a channel or thread."
metadata:
  requires:
    bins: ["slack-cli"]
  cliHelp: "slack-cli chat --help"
---

# slack-messaging: send, read, and react to messages

Read `slack-shared` first (`slack-cli skills read slack-shared`) for auth,
output/error shape, and Block Kit basics — this skill assumes it.

## Core commands

| Task | Command |
|---|---|
| Send a message | `slack-cli chat post-message --channel <id> --text "..."` |
| Send with rich formatting | add `--blocks '[...]'` (see slack-shared Block Kit reference) |
| Reply in a thread | add `--thread_ts <parent ts>` |
| Send visible only to one user | `slack-cli chat post-ephemeral --channel <id> --user <id> --text "..."` |
| Edit a message | `slack-cli chat update --channel <id> --ts <ts> --text "..."` |
| Delete a message | `slack-cli chat delete --channel <id> --ts <ts>` |
| Schedule for later | `slack-cli chat schedule-message --channel <id> --post_at <unix ts> --text "..."` |
| List/cancel scheduled messages | `slack-cli chat scheduled-messages list`, `slack-cli chat delete-scheduled-message --channel <id> --scheduled_message_id <id>` |
| Get a shareable link to a message | `slack-cli chat get-permalink --channel <id> --message_ts <ts>` |
| Read channel history | `slack-cli conversations history --channel <id> --limit 50` |
| Read a thread | `slack-cli conversations replies --channel <id> --ts <parent ts>` |
| Add an emoji reaction | `slack-cli reactions add --channel <id> --timestamp <ts> --name thumbsup` |
| Remove a reaction | `slack-cli reactions remove --channel <id> --timestamp <ts> --name thumbsup` |
| See who reacted / with what | `slack-cli reactions get --channel <id> --timestamp <ts> --full true` |
| Pin a message | `slack-cli pins add --channel <id> --timestamp <ts>` |
| List pinned items | `slack-cli pins list --channel <id>` |
| Add a bookmark (link/message) to a channel | `slack-cli bookmarks add --channel_id <id> --title "..." --type link --link https://...` |

## Key concepts

- A message is identified by **channel id + `ts`**, not a separate message
  id. Every edit/delete/react/pin call needs both.
- To start a thread, just post normally, note the `ts` from the response,
  and pass it as `thread_ts` on later replies to that same message. There's
  no separate "create thread" call.
- `chat.postMessage` requires either `text` or `blocks` (or both — always
  include `text` as the notification fallback even when using `blocks`).
- Bots can only edit or delete messages **they posted**. A user token
  (`xoxp-`) can edit/delete that user's own messages, subject to workspace
  message-editing settings.
- `conversations.history`/`conversations.replies` are cursor-paginated (see
  slack-shared) and return newest-first by default.

## Common OAuth scopes

Exact requirements vary by method — verify at
[api.slack.com/scopes](https://api.slack.com/scopes) — but as a bot these
are typical:

- `chat:write` — post/update/delete messages
- `channels:history` / `groups:history` / `im:history` / `mpim:history` —
  read history, per conversation type
- `reactions:write`, `reactions:read`
- `pins:write`, `pins:read`
- `bookmarks:write`, `bookmarks:read`

## Examples

Post, then reply in a thread:

```sh
resp=$(slack-cli chat post-message --channel C0123456 --text "Kicking off the release" --output compact)
ts=$(echo "$resp" | jq -r .ts)
slack-cli chat post-message --channel C0123456 --thread_ts "$ts" --text "Build passed ✅"
```

Summarize the last 20 messages in a channel:

```sh
slack-cli conversations history --channel C0123456 --limit 20 --output compact | jq -r '.messages[] | "\(.user // .bot_id): \(.text)"'
```

## Related

- `slack-shared` — auth, output/error format, Block Kit primer (this skill assumes it)
- `slack-channels` — creating/archiving/managing the channels you post into
- `slack-search` — finding messages across the whole workspace, not just one channel
- `slack-files` — sharing files rather than text
