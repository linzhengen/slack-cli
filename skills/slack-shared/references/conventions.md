# Shared reference: skills command, and Block Kit basics

## `slack-cli skills` in full

Skill content (this file included) is embedded into the `slack-cli` binary
at build time. That's deliberate: an agent that only has the compiled
binary — no clone of this repo — can still discover and read every skill.

```sh
slack-cli skills list                                # all skills: name, description, version
slack-cli skills list slack-messaging                # ls-style: files directly under a skill
slack-cli skills list slack-messaging/references      # ls-style: one layer under a subdirectory

slack-cli skills read slack-messaging                          # the skill's SKILL.md, raw
slack-cli skills read slack-messaging references/blocks.md     # a reference file, raw
slack-cli skills read slack-messaging/references/blocks.md     # same, slash form
slack-cli skills read slack-messaging --json                   # JSON envelope instead of raw text
```

`read` writes the file's exact bytes to stdout (nothing else — safe to
redirect straight to a file); for the main `SKILL.md` it also writes a
one-line tip to **stderr** pointing out that a reference to another skill
written as `../slack-foo/...` inside a skill file should be read back as
`slack-cli skills read slack-foo/...` (the leading `../` stripped) — the
reader deliberately rejects literal `..` path segments, since it's an
embedded virtual filesystem, not a real one.

## Block Kit quick reference

Rich messages (`chat.postMessage`, `chat.postEphemeral`, `chat.update`,
`views.open`/`push`/`update`/`publish`) take a `blocks` array instead of, or
alongside, plain `text`. Always still send `text` too — it's the fallback
shown in notifications and by clients that don't render blocks.

A minimal message with one section block:

```sh
slack-cli chat post-message \
  --channel C0123456 \
  --text "Deploy finished" \
  --blocks '[
    {"type":"section","text":{"type":"mrkdwn","text":"*Deploy finished* :white_check_mark:"}}
  ]'
```

Common block types:

| Type | Use for |
|---|---|
| `section` | The main text/fields block; supports `mrkdwn` or `plain_text`, and an optional `accessory` (button, image, ...) |
| `divider` | A horizontal rule, no fields |
| `context` | Small muted text/images, e.g. timestamps or attribution |
| `actions` | A row of interactive elements (buttons, select menus) |
| `header` | A large plain-text heading (24 char limit) |
| `image` | A standalone image block |
| `input` | Modal-only: a labeled form field (`views.*`) |

`mrkdwn` text supports `*bold*`, `_italic_`, `~strike~`, `` `code` ``,
`` ```code block``` ``, `<https://example.com|link text>`, and user/channel
mentions as `<@U0123>` / `<#C0123>` — note this is **not** standard Markdown
(no `**bold**`, no `[text](url)` link syntax).

Validate a `blocks` payload without sending anything by checking it against
Slack's [Block Kit Builder](https://app.slack.com/block-kit-builder) before
scripting it, especially for anything with `input` blocks or multiple
interactive elements — malformed blocks fail the whole `chat.postMessage`
call with `invalid_blocks`.
