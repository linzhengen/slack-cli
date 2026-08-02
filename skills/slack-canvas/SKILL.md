---
name: slack-canvas
version: 1.0.0
description: "Create and edit Slack canvases (rich collaborative documents attached to a channel), manage sections, and control who can view or edit them. Use when the user asks to create a doc, write-up, or canvas in Slack, attach a document to a channel, or share/restrict access to one."
metadata:
  requires:
    bins: ["slack-cli"]
  cliHelp: "slack-cli canvases --help"
---

# slack-canvas: Slack's built-in documents

Read `slack-shared` first for auth and output/error conventions.

Canvases are Slack's persistent rich-text documents, either standalone or
attached to a channel (shown as a tab in the channel).

## Core commands

| Task | Command |
|---|---|
| Create a standalone canvas | `slack-cli canvases create --title "Runbook"` |
| Create a canvas attached to a channel | `slack-cli canvases create --title "Meeting notes" --channel_id C0123` |
| Create with initial content | add `--document_content '{"type":"markdown","markdown":"# Title\n..."}'` |
| Edit an existing canvas | `slack-cli canvases edit --canvas_id R0123 --changes '[{"operation":"insert_at_end","document_content":{"type":"markdown","markdown":"\n## New section"}}]'` |
| Delete a canvas | `slack-cli canvases delete --canvas_id R0123` |
| Grant access | `slack-cli canvases access set --canvas_id R0123 --access_level write --user_ids '["U0123"]'` |
| Revoke access | `slack-cli canvases access delete --canvas_id R0123 --user_ids '["U0123"]'` |
| Find a section to edit | `slack-cli canvases sections lookup --canvas_id R0123 --criteria '{"contains_text":"Action items"}'` |

## Key concepts

- `document_content` (on create) and each entry's content in `changes` (on
  edit) use Slack's markdown-flavored canvas format, not full Block Kit.
- `canvases.edit`'s `changes` array applies operations in order —
  `insert_at_start`, `insert_at_end`, `insert_after`/`insert_before` (needs
  a `section_id` from `sections.lookup`), and `replace`/`delete` for
  existing sections. Look up the target section first if you're not
  appending to the end.
- A canvas attached to a channel (`channel_id` at creation) inherits that
  channel's membership for read access by default; `canvases.access.*`
  layers finer-grained overrides on top.
- Canvas ids start with `R`.

## Common OAuth scopes

- `canvases:write` — create, edit, delete, manage access
- `canvases:read` — implied for most read paths alongside channel access

## Examples

Create a channel canvas and append a section to it later:

```sh
rid=$(slack-cli canvases create --title "Incident runbook" --channel_id C0123456 \
  --document_content '{"type":"markdown","markdown":"# Incident runbook\n\n## Steps\n"}' \
  --output compact | jq -r .canvas_id)

slack-cli canvases edit --canvas_id "$rid" --changes '[
  {"operation":"insert_at_end","document_content":{"type":"markdown","markdown":"\n1. Page on-call\n2. Open a war room\n"}}
]'
```

## Related

- `slack-shared` — auth, output/error format
- `slack-messaging` — canvases pair well with posting a link to one, or bookmarking it (`bookmarks.add` with `type=link`)
