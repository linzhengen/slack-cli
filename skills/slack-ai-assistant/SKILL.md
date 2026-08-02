---
name: slack-ai-assistant
version: 1.0.0
description: "Build and drive a Slack AI Assistant app: set the 'is typing...' style status on an assistant thread, offer suggested prompts, set a thread title, and reply via chat.postMessage in the assistant surface. Use when the user is building a Slack bot that responds in Slack's native AI Assistant panel (assistant_thread_started events), not a plain bot posting into channels."
metadata:
  requires:
    bins: ["slack-cli"]
  cliHelp: "slack-cli assistant --help"
---

# slack-ai-assistant: Slack's native AI Assistant surface

Read `slack-shared` first for auth, output/error, and Block Kit basics.

This skill is specifically about the `assistant.threads.*` methods used by
apps that appear in Slack's dedicated AI Assistant panel (as opposed to a
regular bot that just posts into channels — for that, use
`slack-messaging` instead). An assistant app receives
`assistant_thread_started` / `assistant_thread_context_changed` /
`message.im` events (via Events API or Socket Mode, outside this CLI's
scope) and responds using the commands below plus ordinary
`chat.postMessage`.

## Core commands

| Task | Command |
|---|---|
| Show a "thinking" / working status | `slack-cli assistant threads set-status --channel_id D0123 --thread_ts <ts> --status "Looking that up..."` |
| Clear the status (response ready) | same command with `--status ""` |
| Offer suggested follow-up prompts | `slack-cli assistant threads set-suggested-prompts --channel_id D0123 --thread_ts <ts> --prompts '[{"title":"Summarize this thread","message":"Summarize this thread"}]'` |
| Set the thread's title | `slack-cli assistant threads set-title --channel_id D0123 --thread_ts <ts> --title "Deploy question"` |
| Actually send the reply | `slack-cli chat post-message --channel D0123 --thread_ts <ts> --text "..."` (see `slack-messaging`) |

## Key concepts

- An assistant thread lives in a DM-like conversation (`channel_id`
  starting with `D`) between the user and the app; `thread_ts` is the
  thread's parent message timestamp, same concept as any other Slack
  thread.
- Typical turn sequence: on `assistant_thread_started`, optionally set a
  title and suggested prompts immediately; on each incoming user message,
  set a status while working, then clear the status and `chat.postMessage`
  the actual reply.
- `set-suggested-prompts` prompts are what the user sees as clickable
  chips before they've typed anything — keep `message` identical to what
  you'd want sent if they click it verbatim.
- This surface still uses ordinary `chat.postMessage` to actually deliver
  content, including Block Kit — everything in `slack-messaging` and the
  Block Kit reference in `slack-shared` applies directly here.

## Common OAuth scopes

- `assistant:write` — all `assistant.threads.*` methods
- `chat:write`, `im:history` — replying and reading the DM thread

## Examples

A minimal "working → answer" turn:

```sh
slack-cli assistant threads set-status --channel_id D0123456 --thread_ts 1722600000.000100 --status "Checking the deploy log..."
# ... do the actual work ...
slack-cli assistant threads set-status --channel_id D0123456 --thread_ts 1722600000.000100 --status ""
slack-cli chat post-message --channel D0123456 --thread_ts 1722600000.000100 \
  --text "The last deploy finished 12 minutes ago and is healthy."
```

See `references/turn-example.md` in this skill for a fuller worked example
including suggested prompts and title-setting on thread start.

## Related

- `slack-shared` — auth, output/error format, Block Kit primer
- `slack-messaging` — the underlying send/edit/react commands used to actually reply
- `slack-workflows` — for non-conversational automation (workflow steps, custom functions) rather than a chat-style assistant
