---
name: slack-search
version: 1.0.0
description: "Search across the whole workspace for messages and files matching a query, using Slack's search operators (from:, in:, before:, after:, has:, is:). Use when the user asks to find, search for, or locate a message or file anywhere in Slack, rather than in one specific channel or thread they've already named."
metadata:
  requires:
    bins: ["slack-cli"]
  cliHelp: "slack-cli search --help"
---

# slack-search: workspace-wide search

Read `slack-shared` first for auth and output/error conventions.

## This one needs a user token

`search.*` methods require a **user token** (`xoxp-...`), not a bot token —
Slack search is scoped to what a specific person can see, and bots don't
have that notion. If `slack-cli search messages ...` fails with
`not_allowed_token_type` or `user_is_bot`, the active token is a bot token;
switch to a user-token profile (`slack-cli auth login --token xoxp-... --name user`)
or set `SLACK_CLI_TOKEN` to a user token for that call.

## Core commands

| Task | Command |
|---|---|
| Search messages | `slack-cli search messages --query "deploy failed"` |
| Search files | `slack-cli search files --query "quarterly report"` |
| Search both | `slack-cli search all --query "..."` |
| Sort by recency instead of relevance | add `--sort timestamp` (messages only) |

## Key concepts

- The `query` string supports Slack's search operators directly, e.g.:
  - `from:@alice` — sent by a user
  - `in:#general` / `in:@alice` (DM) — scoped to a conversation
  - `before:2026-08-01`, `after:2026-07-01`, `on:2026-08-02`
  - `has:link`, `has:star`, `has:reaction`
  - `is:thread` — only thread replies
  - Quote a phrase with `"..."` for an exact match.
  - Combine operators by spacing them: `--query 'from:@alice in:#eng after:2026-07-01 "rollback"'`
- Results are paginated the same as any other list method (see
  slack-shared) — `--param count=100` and follow `response_metadata.next_cursor`.
- Search only covers conversations the token's user is actually a member
  of (or public channels, if the workspace allows search across those) —
  a message existing doesn't guarantee it's findable by every token.

## Common OAuth scopes (user token)

- `search:read`

## Examples

Find the most recent message from a specific person mentioning an incident:

```sh
slack-cli search messages --query 'from:@alice "incident" ' --sort timestamp --output compact | jq -r '.messages.matches[0].text'
```

## Related

- `slack-shared` — auth, output/error format
- `slack-messaging` — reading history within one already-known channel/thread (works with a bot token; use this instead of search when you already have the channel)
