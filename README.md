# slack-cli

A command-line client for the [Slack Web API](https://api.slack.com/web), written in Go.

It's modeled on [larksuite/cli](https://github.com/larksuite/cli): one binary, one
consistent command shape per API method, structured JSON everywhere. The
whole point is to make Slack scriptable — by you, in a shell, and by an AI
agent that needs a predictable, self-describing tool to act on your behalf.

- **Nearly the entire Web API is covered**, including Enterprise Grid
  `admin.*` methods — see [Method coverage](#method-coverage).
- **Any method can be called, known or not**, via `slack-cli api call`, so
  the CLI never lags behind new Slack API methods.
- **JSON in, JSON out.** Every command prints JSON to stdout on success and
  structured JSON to stderr (with a non-zero exit code) on failure.
- **Self-describing**, so an agent can discover what's available without
  reading Slack's docs: `slack-cli api list`, `slack-cli api describe <method>`.

## Install

```sh
go install github.com/linzhengen/slack-cli/cmd/slack-cli@latest
```

Or build from source:

```sh
git clone https://github.com/linzhengen/slack-cli.git
cd slack-cli
make build   # -> bin/slack-cli
```

## Authenticate

Create a Slack app at https://api.slack.com/apps, add the OAuth scopes you
need, install it to a workspace, and copy the resulting bot token
(`xoxb-...`) or user token (`xoxp-...`).

Then either:

```sh
# store it as a named profile (~/.config/slack-cli/config.yaml, 0600)
slack-cli auth login --token xoxb-... --name default

# or skip profiles entirely — good for CI / agents
export SLACK_CLI_TOKEN=xoxb-...
```

Token resolution order: `--token` flag > `SLACK_CLI_TOKEN` env >
`SLACK_TOKEN` env > `--profile <name>` (or the active profile).

```sh
slack-cli auth whoami                 # calls auth.test
slack-cli auth list                   # list stored profiles (tokens masked)
slack-cli auth use <name>             # switch the active profile
slack-cli auth logout <name>          # remove a stored profile
```

## Usage

Every Slack Web API method is reachable two ways.

**1. As a dedicated command**, generated from the method name
(`chat.postMessage` → `chat post-message`, `admin.users.session.reset` →
`admin users session reset`), with a named flag per documented parameter:

```sh
slack-cli chat post-message --channel C0123 --text "hello from slack-cli"

slack-cli conversations list --types public_channel,private_channel

slack-cli chat post-message \
  --channel C0123 \
  --blocks '[{"type":"section","text":{"type":"mrkdwn","text":"*bold*"}}]'
```

Fields not exposed as a named flag still work via `--param key=value`
(repeatable) or `--json '{...}'` (merged in last, so it always wins):

```sh
slack-cli chat post-message --channel C0123 --param text=hi --param thread_ts=169...
```

**2. Through the generic escape hatch**, `api call`, which can invoke any
method by name — including ones this CLI has no dedicated command for yet:

```sh
slack-cli api call chat.postMessage --param channel=C0123 --param text=hi
slack-cli api call some.brand-new.method --json '{"foo":"bar"}'
```

### Discovery (for humans and agents)

```sh
slack-cli api list                          # every known method + description, as JSON
slack-cli api list --category admin.        # filter by prefix
slack-cli api describe chat.postMessage     # params, required flags, deprecation notes
```

### File uploads

`files.upload` is deprecated by Slack; `slack-cli files upload` implements
the current three-step flow (`getUploadURLExternal` → raw multipart POST →
`completeUploadExternal`) as one command:

```sh
slack-cli files upload --file ./report.csv --channel C0123 --title "Weekly report"
```

### Output

```sh
slack-cli --output pretty  ...   # multi-line indented JSON (default)
slack-cli --output compact ...   # single-line JSON, best for piping to jq or an agent
```

Errors are printed to stderr as JSON and exit non-zero — Slack API errors
(`ok:false`) are passed through as Slack sent them; client-side errors
(missing required params, auth failures, transport errors) use a small
stable shape: `{"ok":false,"error":"...","method":"..."}`.

## Design for AI agents

- Deterministic, parseable output on both streams — no partial/mixed
  human+JSON output, no prompts, no interactive fallback.
- `api list` / `api describe` let an agent enumerate capabilities and
  required parameters at runtime instead of needing them baked into a
  system prompt.
- Required parameters are validated client-side before any network call, so
  failures are fast and specific (`missing required parameter(s): [channel]`)
  rather than a generic Slack 400.
- Non-interactive by design: pass `--token` (or `SLACK_CLI_TOKEN`) and every
  command is safe to run unattended.

## Method coverage

Run `slack-cli api list` for the full, current list (200+ methods). Broadly:

`admin.*` (apps, auth policy, barriers, conversations, emoji, invite
requests, roles, teams, usergroups, users, workflows) · `api.test` ·
`apps.*` · `assistant.threads.*` · `auth.*` · `bookmarks.*` · `bots.*` ·
`calls.*` · `canvases.*` · `chat.*` · `conversations.*` · `dnd.*` ·
`emoji.*` · `files.*` (incl. `files.remote.*`) · `functions.*` ·
`migration.*` · `oauth.*` · `openid.connect.*` · `pins.*` · `reactions.*` ·
`reminders.*` · `rtm.connect` · `search.*` · `stars.*` · `team.*` ·
`usergroups.*` · `users.*` (incl. `users.profile.*`) · `views.*` ·
`workflows.*`

Even methods not in this list can be called via `slack-cli api call
<method>` — the registry is a documentation/UX layer, not a limit.

## Architecture

```
cmd/slack-cli/          entry point
internal/slackapi/      generic Web API HTTP client + the method registry
internal/config/        auth profile storage (~/.config/slack-cli/config.yaml)
internal/cliutil/       JSON output + structured error formatting
internal/cli/           cobra command tree:
  root.go                 wires everything together
  auth.go                 profile management commands
  api.go                  api call / api list / api describe
  generated.go             builds the whole method-registry command tree
  files.go                 friendly files-upload wrapper
  params.go, call.go       shared flag parsing + Slack call/print logic
```

Commands are generated from `internal/slackapi/methods.go`, a single
declarative table of `{name, description, params}` per Slack Web API
method. Adding a method — including brand-new ones Slack ships after this
was written — is a one-line addition to that table; the command tree,
help text, and flags follow automatically.

## Development

Requires Go 1.25+ (pinned via the `go` directive in `go.mod`) and
[golangci-lint](https://golangci-lint.run/) for `make lint`.

```sh
make build
make test
make vet
make lint
make fmt-check
```
