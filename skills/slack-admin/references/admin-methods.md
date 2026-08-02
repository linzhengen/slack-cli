# admin.* method categories

The full, current list (with per-method descriptions and parameters) is
always available live: `slack-cli api list --category admin.` and
`slack-cli api describe admin.<method>`. This is a category map to help you
find the right subgroup quickly; treat it as an index, not the source of
truth.

| CLI path | Slack method prefix | Covers |
|---|---|---|
| `admin apps` | `admin.apps.*` | Approve/restrict app installs, list requests, uninstall, activity log |
| `admin auth-policy` | `admin.authPolicy.*` | Assign/query authentication policies (e.g. required auth method) to entities |
| `admin barriers` | `admin.barriers.*` | Information barriers between user groups |
| `admin conversations` | `admin.conversations.*` | Org-wide channel CRUD, Slack Connect (shared channel) management, access restriction by IDP group, bulk archive/delete/move |
| `admin emoji` | `admin.emoji.*` | Org-wide custom emoji add/remove/rename/alias |
| `admin invite-requests` | `admin.inviteRequests.*` | Approve/deny pending workspace invite requests |
| `admin roles` | `admin.roles.*` | Assign/list/remove custom role assignments |
| `admin teams` | `admin.teams.*` | List/create workspaces, list admins/owners, workspace-level settings (name, icon, description, discoverability, default channels) |
| `admin usergroups` | `admin.usergroups.*` | Org-wide user group channel/workspace associations |
| `admin users` | `admin.users.*` | Provision/deprovision, set role (admin/owner/regular), set expiration (guests), session management (list/reset/invalidate) |
| `admin workflows` | `admin.workflows.*` | Search workflows, manage collaborators, permission lookups, unpublish |

## Frequently needed, easy to miss

- **Deactivating a user org-wide** (not just removing from one workspace)
  isn't a single dedicated method — the closest built-in path is
  `admin.users.remove` per workspace, or resetting/invalidating sessions
  (`admin.users.session.reset` / `admin.users.session.invalidate`) to
  immediately cut off access while an actual deactivation happens through
  SCIM or the admin console.
- **Finding a channel's id from its name org-wide**: `admin.conversations.search`
  (`slack-cli admin conversations search --query <name>`), not
  `conversations.list`, which is scoped to one workspace and only sees
  channels the calling token is a member of.
- **Bulk operations** (`bulkArchive`, `bulkDelete`, `bulkMove`) take
  `channel_ids` as a **JSON array**, unlike most other multi-id parameters
  in this CLI which take comma-separated strings — check
  `slack-cli api describe admin.conversations.bulkDelete` before calling.
