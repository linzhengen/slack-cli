---
name: slack-admin
version: 1.0.0
description: "Enterprise Grid organization administration: provision/deprovision users across workspaces, manage channels org-wide, control app installation approval, manage custom emoji, user groups, and workspace settings at the org level. Use when the user asks to do something 'across the whole org' or 'for all workspaces', approve or restrict an app, provision or offboard a user org-wide, or any task naming 'admin' explicitly. Requires an Enterprise Grid org and an admin-level token."
metadata:
  requires:
    bins: ["slack-cli"]
  cliHelp: "slack-cli admin --help"
---

# slack-admin: Enterprise Grid organization administration

Read `slack-shared` first for auth and output/error conventions.

`admin.*` is Slack's Enterprise Grid org-management surface — a different
scale from every other `slack-*` skill here, which operate within a single
workspace. It only works against an Enterprise Grid org with a token that
has org admin privileges; on a standard workspace, every `admin.*` call
fails with `not_allowed` or `org_login_required` regardless of scopes.

This skill covers the most common tasks directly. `admin.*` has ~90
methods total; see `references/admin-methods.md` in this skill for the
full category breakdown, or `slack-cli api list --category admin.` for the
live list with descriptions.

## Core commands

| Task | Command |
|---|---|
| List workspaces in the org | `slack-cli admin teams list` |
| List users on a workspace | `slack-cli admin users list --team_id T0123` |
| Add an existing org user to a workspace | `slack-cli admin users assign --team_id T0123 --user_id U0456` |
| Invite a new user to a workspace | `slack-cli admin users invite --team_id T0123 --email person@example.com` |
| Remove a user from a workspace | `slack-cli admin users remove --team_id T0123 --user_id U0456` |
| Deactivate a user org-wide | see `references/admin-methods.md` (`admin.users.*` session/session-reset paths) |
| Create an org-wide channel | `slack-cli admin conversations create --name org-announcements --team_id T0123` |
| Search channels org-wide | `slack-cli admin conversations search --query "incident"` |
| Archive/rename any channel | `slack-cli admin conversations archive --channel_id C0123`, `slack-cli admin conversations rename --channel_id C0123 --name new-name` |
| Approve a pending app install | `slack-cli admin apps approve --app_id A0123 --team_id T0123` |
| Restrict an app | `slack-cli admin apps restrict --app_id A0123 --team_id T0123` |
| Manage org-wide custom emoji | `slack-cli admin emoji add --name party --url https://...`, `slack-cli admin emoji list` |
| Manage org-wide user groups | `slack-cli admin usergroups add-channels --usergroup_id S0123 --channel_ids C0111,C0222` |

## Key concepts

- Most `admin.*` write operations require `team_id` (target workspace)
  even when the action conceptually applies "everywhere" — get it from
  `admin teams list` if you don't already have it.
- `admin.*` bypasses normal workspace-membership requirements — an admin
  token can archive or rename a channel it isn't a member of, unlike the
  plain `conversations.*` equivalents in `slack-channels`.
- These are high-blast-radius operations (user removal, channel deletion,
  app restriction affect real people and data immediately, with no
  built-in undo for most of them). Confirm scope and target explicitly
  before calling anything under `admin.conversations.delete`,
  `admin.conversations.bulkDelete`, `admin.users.remove`, or
  `admin.apps.uninstall` — there is no dry-run flag.
- Bulk operations (`admin.conversations.bulkArchive/bulkDelete/bulkMove`)
  take a JSON array of channel ids, not a comma-separated list — see
  `slack-cli api describe admin.conversations.bulkDelete`.

## Common OAuth scopes

Admin scopes are their own namespace, separate from regular bot/user
scopes, e.g. `admin.users:write`, `admin.conversations:write`,
`admin.apps:write`, `admin.usergroups:write` — granted only to apps
installed org-wide with admin approval. See
[api.slack.com/scopes](https://api.slack.com/scopes) (search "admin.") for
the exact scope each method needs.

## Examples

Offboard a user from one workspace (not the whole org):

```sh
slack-cli admin users remove --team_id T0123456 --user_id U0456789
```

List every workspace, then every user in the first one:

```sh
team_id=$(slack-cli admin teams list --output compact | jq -r '.teams[0].id')
slack-cli admin users list --team_id "$team_id"
```

## Related

- `slack-shared` — auth, output/error format
- `slack-channels` — single-workspace channel management for the common (non-Enterprise-Grid) case
- `slack-users` — single-workspace user lookup/profile for the common case
