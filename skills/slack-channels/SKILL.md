---
name: slack-channels
version: 1.0.0
description: "Create, archive, rename, and configure channels; manage membership (invite/kick/join/leave); list channels and their members; set topic/purpose. Use when the user asks to create a channel, invite or remove someone from a channel, archive/rename/unarchive a channel, list channels or their members, or change a channel's topic or purpose."
metadata:
  requires:
    bins: ["slack-cli"]
  cliHelp: "slack-cli conversations --help"
---

# slack-channels: channel lifecycle and membership

Read `slack-shared` first for auth and output/error conventions.

In the Slack API, channels, DMs, and group DMs are all "conversations" —
one `conversations.*` namespace covers all of them, and `slack-cli
conversations --help` lists every subcommand.

## Core commands

| Task | Command |
|---|---|
| Create a public channel | `slack-cli conversations create --name my-channel` |
| Create a private channel | `slack-cli conversations create --name my-channel --is_private true` |
| List channels | `slack-cli conversations list --types public_channel,private_channel` |
| Get channel info | `slack-cli conversations info --channel <id>` |
| List a channel's members | `slack-cli conversations members --channel <id>` |
| Invite users | `slack-cli conversations invite --channel <id> --users U0123,U0456` |
| Remove a user | `slack-cli conversations kick --channel <id> --user <id>` |
| Join a channel (as the bot/user) | `slack-cli conversations join --channel <id>` |
| Leave a channel | `slack-cli conversations leave --channel <id>` |
| Rename | `slack-cli conversations rename --channel <id> --name new-name` |
| Set topic | `slack-cli conversations set-topic --channel <id> --topic "..."` |
| Set purpose | `slack-cli conversations set-purpose --channel <id> --purpose "..."` |
| Archive | `slack-cli conversations archive --channel <id>` |
| Unarchive | `slack-cli conversations unarchive --channel <id>` |
| Open/resume a DM or group DM | `slack-cli conversations open --users U0123,U0456` |
| Mark as read up to a point | `slack-cli conversations mark --channel <id> --ts <ts>` |

## Key concepts

- Channel names must be lowercase, no spaces (use hyphens), max 80 chars.
- A bot must **join** a public channel (or be **invited** to a private one)
  before it can post or read history there — `channel_not_found` or
  `not_in_channel` errors usually mean this step was skipped.
- `conversations.list` defaults to public channels only; pass `--types` to
  include private channels, IMs, or MPIMs (comma-separated, no spaces).
- Renaming/archiving requires the acting token to be a channel member with
  sufficient permission (often the creator or a workspace admin) —
  workspace-wide overrides for any channel live in `slack-admin`
  (`admin.conversations.*`), not here.
- `conversations.open` with a single user id opens/resumes a 1:1 DM; with
  multiple, an MPIM (group DM).

## Common OAuth scopes

- `channels:read` / `groups:read` / `im:read` / `mpim:read` — list/info
- `channels:manage` / `groups:write` — create, rename, archive, set topic/purpose
- `channels:join` — join a public channel

## Examples

Create a private incident channel and invite the on-call team:

```sh
ch=$(slack-cli conversations create --name incident-2026-08-02 --is_private true --output compact | jq -r .channel.id)
slack-cli conversations invite --channel "$ch" --users U0111,U0222,U0333
slack-cli conversations set-topic --channel "$ch" --topic "Tracking the 2026-08-02 incident"
```

## Related

- `slack-shared` — auth, output/error format
- `slack-messaging` — once you have a channel, sending into it
- `slack-admin` — org-wide channel administration (Enterprise Grid), beyond a single workspace's `conversations.*`
