---
name: slack-users
version: 1.0.0
description: "Look up and list workspace users, resolve identity by email, read/update profile fields, check presence, and manage user groups (@-mentionable teams). Use when the user asks to find a user by name or email, look up someone's profile or timezone, check if someone is online, list a workspace's members, or create/update a user group (team mention alias)."
metadata:
  requires:
    bins: ["slack-cli"]
  cliHelp: "slack-cli users --help"
---

# slack-users: directory, profiles, presence, and user groups

Read `slack-shared` first for auth and output/error conventions.

## Core commands

| Task | Command |
|---|---|
| List all users | `slack-cli users list` |
| Get a user by id | `slack-cli users info --user U0123` |
| Find a user by email | `slack-cli users lookup-by-email --email person@example.com` |
| Read a user's profile fields | `slack-cli users profile get --user U0123` |
| Set the *calling user's* profile field | `slack-cli users profile set --name status_text --value "In a meeting"` |
| Check presence (active/away) | `slack-cli users get-presence --user U0123` |
| Set the *calling user's* presence | `slack-cli users set-presence --presence away` |
| List conversations a user can access | `slack-cli users conversations --user U0123` |
| Create a user group | `slack-cli usergroups create --name "On-call" --handle oncall` |
| List user groups | `slack-cli usergroups list --include_users true` |
| Set a user group's members | `slack-cli usergroups users update --usergroup S0123 --users U0111,U0222` |
| Enable/disable a user group | `slack-cli usergroups enable --usergroup S0123`, `slack-cli usergroups disable --usergroup S0123` |

## Key concepts

- `users.list`/`users.info` return a `deleted`/`is_bot`/`is_restricted`
  flag set — check these before assuming a result is an active human
  member (deactivated accounts and bots still show up).
- `users.profile.set` and `users.setPresence` act on **the token's own
  identity** — a bot token sets the bot's own profile/presence, not an
  arbitrary user's; to set another user's profile you need that user's own
  token (`xoxp-`) or an `admin.users.*` path.
- User groups (`usergroups.*`, ids start with `S`) are what makes
  `@handle` mention a whole team at once — distinct from IDP/SCIM groups
  used in `admin.*` access control.
- `users.identity` / `openid.connect.userInfo` are for "Sign in with
  Slack" flows, not general directory lookups — use `users.info` instead
  for that.

## Common OAuth scopes

- `users:read`, `users:read.email` — list/lookup/info
- `users.profile:read`, `users.profile:write` — profile fields
- `usergroups:read`, `usergroups:write`

## Examples

Resolve an email to a user id, then check if they're online:

```sh
uid=$(slack-cli users lookup-by-email --email person@example.com --output compact | jq -r .user.id)
slack-cli users get-presence --user "$uid" --output compact | jq -r .presence
```

## Related

- `slack-shared` — auth, output/error format
- `slack-messaging` — DMing a user once you have their id
- `slack-admin` — org-wide user provisioning/deprovisioning (`admin.users.*`), beyond a single workspace's `users.*`
