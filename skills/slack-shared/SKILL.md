---
name: slack-shared
version: 1.0.0
description: "Cross-cutting slack-cli conventions: authentication, output format, error shape, Slack ID formats, pagination, rate limits, and the api call/list/describe escape hatch. Read this first, before any other slack-* skill — they all assume it. Use it when setting up auth, when unsure how to parse output or handle an error, or when a method isn't covered by a dedicated skill yet."
metadata:
  requires:
    bins: ["slack-cli"]
  cliHelp: "slack-cli --help"
---

# slack-shared: conventions every other slack-* skill assumes

This skill isn't a task area like messaging or files — it's the baseline
every other `slack-*` skill in this CLI builds on. Read it once; the other
skills reference it instead of repeating this.

## Authentication

Resolution order (first one set wins): `--token` flag > `SLACK_CLI_TOKEN`
env var > `SLACK_TOKEN` env var > `--profile <name>` (or the active stored
profile from `slack-cli auth login`).

For an agent, the simplest and most reliable path is `SLACK_CLI_TOKEN` in
the environment — no interactive step, no state on disk to get out of sync.

```sh
export SLACK_CLI_TOKEN=xoxb-...
slack-cli auth whoami   # sanity check: calls auth.test
```

Slack tokens come in a few shapes, and which one you have determines what
you can call:

- `xoxb-...` — bot token. Most methods; scopes come from the app's
  configured **Bot Token Scopes**.
- `xoxp-...` — user token. Required for a handful of methods that act as a
  real user (notably `search.*`, `stars.*`, and some `admin.*` methods).
- `xoxa-...` / `xoxe.xoxp-...` — app-level / refresh tokens, used for
  Socket Mode and OAuth token rotation, not for calling most Web API
  methods directly.

## Output format

Every command prints JSON. `--output pretty` (default) is multi-line
indented JSON; `--output compact` is single-line, best for piping to `jq`
or parsing directly. Both are valid JSON — pick based on whether a human or
another program is reading it, not based on parseability.

## Errors

On failure, a command exits non-zero and writes JSON to **stderr**, never
stdout — so `stdout` is always either valid success JSON or empty.

- A Slack API error (`ok:false`) is passed through as Slack sent it, e.g.
  `{"ok":false,"error":"channel_not_found"}`. The `error` string is one of
  Slack's documented [error codes](https://api.slack.com/methods) for that
  method.
- A client-side error (missing required parameter, bad auth, transport
  failure) uses a small stable shape:
  `{"ok":false,"error":"...","method":"..."}`.

Required parameters are validated **before** any network call, so a missing
`channel` fails fast with `missing required parameter(s): [channel]`
instead of round-tripping to Slack for a generic 400.

## Slack ID formats

IDs are opaque but their leading letter tells you what they are — useful
for sanity-checking a value before sending it:

| Prefix | Kind |
|---|---|
| `C` | public or private channel |
| `G` | private channel / MPIM (legacy naming, still appears) |
| `D` | direct message conversation |
| `U` | user |
| `B` | bot user |
| `T` | team / workspace |
| `F` | file |
| `W` | workflow / workflow step |
| `Fn` | function (workflow builder custom function) |
| `R` | canvas |

A message is identified by its **channel id + `ts`** (a string like
`"1234567890.123456"`, seconds.microseconds) — there is no separate
"message id". To reply in a thread, pass the parent message's `ts` as
`thread_ts`. `ts` also sorts chronologically as a string, so it's safe to
compare for ordering.

## Pagination

List-style methods (`conversations.list`, `users.list`, `search.messages`,
most `admin.*` listings, ...) are cursor-paginated: the response has
`response_metadata.next_cursor`; pass that back as the `cursor` parameter
to get the next page, and stop when it's empty. There is no built-in
"fetch everything" flag — loop from the caller (agent or shell) instead:

```sh
cursor=""
while :; do
  resp=$(slack-cli conversations list --output compact --param limit=200 --param cursor="$cursor")
  echo "$resp" | jq -c '.channels[]'
  cursor=$(echo "$resp" | jq -r '.response_metadata.next_cursor // empty')
  [ -z "$cursor" ] && break
done
```

## Rate limits

slack-cli automatically retries once on HTTP 429, honoring Slack's
`Retry-After` header. If you still hit a rate limit in a tight loop
(common with `admin.*` and bulk `conversations.history` calls), add your
own backoff between calls rather than retrying immediately.

## The escape hatch: methods without a dedicated skill

Every Slack Web API method — 200+, including ones added after this CLI was
built — is reachable even without a task-specific skill:

```sh
slack-cli api list                          # every known method, JSON
slack-cli api list --category admin.        # filter by prefix
slack-cli api describe chat.postMessage     # params, required flags, notes
slack-cli api call some.new.method --json '{"foo":"bar"}'
```

If you're not sure which skill covers a task, `slack-cli api list --category
<prefix>` plus `api describe` is always a valid fallback — every dedicated
command (e.g. `slack-cli chat post-message`) is equivalent to `api call
chat.postMessage` with named flags instead of `--param`/`--json`.

## Discovering skills

This mechanism you're reading right now — `slack-cli skills list` /
`slack-cli skills read <name>` — works the same way for every skill in this
CLI. See `references/conventions.md` in this skill for the full command
reference, or just run `slack-cli skills --help`.
