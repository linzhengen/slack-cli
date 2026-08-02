package slackapi

import "sort"

// Param documents one field a Method accepts. The registry is descriptive,
// not enforcing: any method can still be called with arbitrary extra
// parameters via --param/--json, so the CLI stays correct even for fields
// this table doesn't yet know about.
type Param struct {
	Name     string `json:"name"`
	Required bool   `json:"required"`
	Desc     string `json:"description"`
}

// Method describes one Slack Web API method for command generation, help
// text, and agent self-discovery (see `slack-cli api describe`).
type Method struct {
	Name       string  `json:"name"`
	Desc       string  `json:"description"`
	Params     []Param `json:"params"`
	Deprecated string  `json:"deprecated,omitempty"` // non-empty explains what to use instead
}

func p(name string, required bool, desc string) Param {
	return Param{Name: name, Required: required, Desc: desc}
}

func m(name, desc string, params ...Param) Method {
	return Method{Name: name, Desc: desc, Params: params}
}

func deprecated(name, desc, note string, params ...Param) Method {
	return Method{Name: name, Desc: desc, Params: params, Deprecated: note}
}

// Methods is the registry of known Slack Web API methods. It intentionally
// covers the whole public method surface (including Enterprise Grid
// admin.* methods) so that both the generated command tree and
// `slack-cli api describe` are useful without consulting Slack's docs.
//
// This list documents required and commonly-used parameters; any method,
// including ones not listed here at all, can still be invoked with
// `slack-cli api call <method>`.
var Methods = []Method{
	// --- api ---
	m("api.test", "Checks API calling code, without a token.",
		p("error", false, "Error response to mimic"),
		p("foo", false, "Example property to echo back"),
	),

	// --- auth ---
	m("auth.test", "Checks authentication & identity."),
	m("auth.revoke", "Revokes a token.", p("test", false, "Dry run without revoking")),

	// --- apps ---
	m("apps.uninstall", "Uninstalls your app from a workspace.",
		p("client_id", true, "App's client id"),
		p("client_secret", true, "App's client secret"),
	),
	m("apps.event.authorizations.list", "Returns list of authorizations for an event trigger.",
		p("event_context", true, "Event context token"),
		p("cursor", false, "Pagination cursor"),
		p("limit", false, "Max results"),
	),

	// --- assistant.threads (AI Assistant apps) ---
	m("assistant.threads.setStatus", "Sets a status on an assistant thread (e.g. 'is typing…').",
		p("channel_id", true, "Channel/conversation id"),
		p("thread_ts", true, "Thread timestamp"),
		p("status", false, "Status text; empty clears it"),
	),
	m("assistant.threads.setSuggestedPrompts", "Sets suggested prompts for an assistant thread.",
		p("channel_id", true, "Channel/conversation id"),
		p("thread_ts", true, "Thread timestamp"),
		p("prompts", true, "JSON array of {title, message}"),
		p("title", false, "Prompts section title"),
	),
	m("assistant.threads.setTitle", "Sets the title of an assistant thread.",
		p("channel_id", true, "Channel/conversation id"),
		p("thread_ts", true, "Thread timestamp"),
		p("title", true, "New title"),
	),

	// --- bookmarks ---
	m("bookmarks.add", "Adds a bookmark to a channel.",
		p("channel_id", true, "Channel id"),
		p("title", true, "Bookmark title"),
		p("type", true, "link | message"),
		p("link", false, "URL (required for type=link)"),
		p("entity_id", false, "Message id (for type=message)"),
	),
	m("bookmarks.edit", "Edits a bookmark.",
		p("bookmark_id", true, "Bookmark id"),
		p("channel_id", true, "Channel id"),
		p("title", false, "New title"),
		p("link", false, "New URL"),
	),
	m("bookmarks.list", "Lists bookmarks for a channel.", p("channel_id", true, "Channel id")),
	m("bookmarks.remove", "Removes a bookmark.", p("bookmark_id", true, "Bookmark id"), p("channel_id", true, "Channel id")),

	// --- bots ---
	m("bots.info", "Gets information about a bot user.", p("bot", false, "Bot user id")),

	// --- calls ---
	m("calls.add", "Registers a new Call.",
		p("external_unique_id", true, "Unique id for the call"),
		p("join_url", true, "URL to join the call"),
		p("desktop_app_join_url", false, "Deep link URL"),
		p("external_display_id", false, "Display id"),
		p("title", false, "Call title"),
		p("users", false, "JSON array of call participants"),
	),
	m("calls.end", "Ends a Call.", p("id", true, "Call id"), p("duration", false, "Call duration in seconds")),
	m("calls.info", "Returns information about a Call.", p("id", true, "Call id")),
	m("calls.update", "Updates a Call.", p("id", true, "Call id"), p("title", false, "New title"), p("join_url", false, "New join URL")),
	m("calls.participants.add", "Registers new participants added to a Call.", p("id", true, "Call id"), p("users", true, "JSON array of participants")),
	m("calls.participants.remove", "Registers participants removed from a Call.", p("id", true, "Call id"), p("users", true, "JSON array of participants")),

	// --- canvases ---
	m("canvases.create", "Creates a new canvas.",
		p("title", false, "Canvas title"),
		p("document_content", false, "JSON document content"),
		p("channel_id", false, "Channel to attach the canvas to"),
	),
	m("canvases.edit", "Edits an existing canvas.", p("canvas_id", true, "Canvas id"), p("changes", true, "JSON array of change operations")),
	m("canvases.delete", "Deletes a canvas.", p("canvas_id", true, "Canvas id")),
	m("canvases.access.set", "Sets canvas access for users/channels.",
		p("canvas_id", true, "Canvas id"), p("access_level", true, "read | write"),
		p("channel_ids", false, "JSON array of channel ids"), p("user_ids", false, "JSON array of user ids"),
	),
	m("canvases.access.delete", "Removes canvas access.", p("canvas_id", true, "Canvas id"),
		p("channel_ids", false, "JSON array of channel ids"), p("user_ids", false, "JSON array of user ids"),
	),
	m("canvases.sections.lookup", "Finds sections matching criteria within a canvas.", p("canvas_id", true, "Canvas id"), p("criteria", true, "JSON search criteria")),

	// --- chat ---
	m("chat.delete", "Deletes a message.", p("channel", true, "Channel id"), p("ts", true, "Message timestamp")),
	m("chat.deleteScheduledMessage", "Deletes a pending scheduled message.",
		p("channel", true, "Channel id"), p("scheduled_message_id", true, "Scheduled message id"),
	),
	m("chat.getPermalink", "Retrieves a permalink URL for a message.", p("channel", true, "Channel id"), p("message_ts", true, "Message timestamp")),
	m("chat.meMessage", "Shares a /me style message.", p("channel", true, "Channel id"), p("text", true, "Message text")),
	m("chat.postEphemeral", "Sends an ephemeral message to a user in a channel.",
		p("channel", true, "Channel id"), p("user", true, "User id to see the message"),
		p("text", false, "Message text (or use blocks)"), p("blocks", false, "JSON array of Block Kit blocks"),
	),
	m("chat.postMessage", "Sends a message to a channel.",
		p("channel", true, "Channel, private group, or IM id"),
		p("text", false, "Message text (fallback if blocks used)"),
		p("blocks", false, "JSON array of Block Kit blocks"),
		p("attachments", false, "JSON array of legacy attachments"),
		p("thread_ts", false, "Parent message ts to reply in a thread"),
		p("unfurl_links", false, "true/false"),
		p("mrkdwn", false, "true/false"),
	),
	m("chat.scheduleMessage", "Schedules a message to be sent in the future.",
		p("channel", true, "Channel id"), p("post_at", true, "Unix timestamp to send at"),
		p("text", false, "Message text"), p("blocks", false, "JSON array of Block Kit blocks"),
	),
	m("chat.scheduledMessages.list", "Lists pending scheduled messages.",
		p("channel", false, "Filter by channel id"), p("cursor", false, "Pagination cursor"), p("limit", false, "Max results"),
	),
	m("chat.unfurl", "Provides custom unfurl behavior for user-posted URLs.",
		p("channel", true, "Channel id"), p("ts", true, "Message timestamp"), p("unfurls", true, "JSON object of url -> unfurl attachment"),
	),
	m("chat.update", "Edits an existing message.",
		p("channel", true, "Channel id"), p("ts", true, "Message timestamp"),
		p("text", false, "New text"), p("blocks", false, "JSON array of Block Kit blocks"),
	),

	// --- conversations ---
	m("conversations.acceptSharedInvite", "Accepts an invitation to a Slack Connect channel.",
		p("channel_name", true, "Name for the channel"), p("invite_id", false, "Invite id"), p("channel_id", false, "Channel id"),
	),
	m("conversations.approveSharedInvite", "Approves a Slack Connect channel invite.", p("invite_id", true, "Invite id")),
	m("conversations.declineSharedInvite", "Declines a Slack Connect channel invite.", p("invite_id", true, "Invite id")),
	m("conversations.archive", "Archives a conversation.", p("channel", true, "Channel id")),
	m("conversations.close", "Closes a direct message or multi-person direct message.", p("channel", true, "Channel id")),
	m("conversations.create", "Initiates a public or private channel-based conversation.",
		p("name", true, "Channel name"), p("is_private", false, "true for a private channel"), p("team_id", false, "Team id (org installs)"),
	),
	m("conversations.externalInvitePermissions.set", "Configures Slack Connect invite permissions for a channel.",
		p("channel", true, "Channel id"), p("action", true, "invites_open | approval_required | invites_restricted"),
	),
	m("conversations.history", "Fetches a conversation's history of messages.",
		p("channel", true, "Channel id"), p("cursor", false, "Pagination cursor"), p("limit", false, "Max results"),
		p("oldest", false, "Only messages after this ts"), p("latest", false, "Only messages before this ts"),
	),
	m("conversations.info", "Retrieves information about a conversation.", p("channel", true, "Channel id"), p("include_locale", false, "true/false")),
	m("conversations.invite", "Invites users to a channel.", p("channel", true, "Channel id"), p("users", true, "Comma-separated user ids")),
	m("conversations.inviteShared", "Sends a Slack Connect invitation to a channel.",
		p("channel", true, "Channel id"), p("emails", false, "Comma-separated emails"), p("user_ids", false, "Comma-separated user ids"),
	),
	m("conversations.join", "Joins an existing conversation.", p("channel", true, "Channel id")),
	m("conversations.kick", "Removes a user from a conversation.", p("channel", true, "Channel id"), p("user", true, "User id")),
	m("conversations.leave", "Leaves a conversation.", p("channel", true, "Channel id")),
	m("conversations.list", "Lists all channels in a workspace.",
		p("cursor", false, "Pagination cursor"), p("limit", false, "Max results"),
		p("exclude_archived", false, "true/false"), p("types", false, "public_channel,private_channel,mpim,im"),
	),
	m("conversations.listConnectInvites", "Lists Slack Connect invitations for a team.", p("cursor", false, "Pagination cursor"), p("count", false, "Max results")),
	m("conversations.mark", "Sets the read cursor in a channel.", p("channel", true, "Channel id"), p("ts", true, "Message ts to mark read up to")),
	m("conversations.members", "Retrieves members of a conversation.", p("channel", true, "Channel id"), p("cursor", false, "Pagination cursor"), p("limit", false, "Max results")),
	m("conversations.open", "Opens or resumes a direct or multi-person direct message.",
		p("channel", false, "Existing conversation id to resume"), p("users", false, "Comma-separated user ids"), p("return_im", false, "true/false"),
	),
	m("conversations.rename", "Renames a conversation.", p("channel", true, "Channel id"), p("name", true, "New name")),
	m("conversations.replies", "Retrieves a thread of messages.",
		p("channel", true, "Channel id"), p("ts", true, "Parent message ts"), p("cursor", false, "Pagination cursor"), p("limit", false, "Max results"),
	),
	m("conversations.requestSharedInvite.approve", "Approves a Slack Connect invite request.", p("invite_id", true, "Invite request id")),
	m("conversations.requestSharedInvite.deny", "Denies a Slack Connect invite request.", p("invite_id", true, "Invite request id")),
	m("conversations.requestSharedInvite.list", "Lists Slack Connect invite requests.", p("cursor", false, "Pagination cursor"), p("limit", false, "Max results")),
	m("conversations.setPurpose", "Sets the purpose of a conversation.", p("channel", true, "Channel id"), p("purpose", true, "New purpose text")),
	m("conversations.setTopic", "Sets the topic of a conversation.", p("channel", true, "Channel id"), p("topic", true, "New topic text")),
	m("conversations.unarchive", "Reverses conversation archival.", p("channel", true, "Channel id")),

	// --- dnd ---
	m("dnd.endDnd", "Ends the current user's Do Not Disturb session."),
	m("dnd.endSnooze", "Ends the current user's snooze mode."),
	m("dnd.info", "Retrieves a user's current Do Not Disturb status.", p("user", false, "User id")),
	m("dnd.setSnooze", "Turns on Do Not Disturb snooze mode.", p("num_minutes", true, "Snooze duration in minutes")),
	m("dnd.teamInfo", "Retrieves Do Not Disturb status for multiple users.", p("users", false, "Comma-separated user ids")),

	// --- emoji ---
	m("emoji.list", "Lists custom emoji for a team."),

	// --- files ---
	m("files.completeUploadExternal", "Finalizes an upload started with files.getUploadURLExternal.",
		p("files", true, "JSON array of {id, title}"), p("channel_id", false, "Channel to share to"),
		p("initial_comment", false, "Comment to add"), p("thread_ts", false, "Thread ts to reply in"),
	),
	m("files.getUploadURLExternal", "Gets a URL to upload a file to.",
		p("filename", true, "Filename"), p("length", true, "File size in bytes"), p("alt_text", false, "Image alt text"), p("snippet_type", false, "Syntax highlighting language"),
	),
	m("files.delete", "Deletes a file.", p("file", true, "File id")),
	m("files.info", "Gets information about a file.", p("file", true, "File id"), p("cursor", false, "Pagination cursor for comments"), p("limit", false, "Max results")),
	m("files.list", "Lists & filters team files.",
		p("channel", false, "Filter by channel"), p("user", false, "Filter by user"), p("ts_from", false, "Filter from ts"),
		p("ts_to", false, "Filter to ts"), p("types", false, "Comma-separated file types"), p("cursor", false, "Pagination cursor"),
	),
	m("files.revokePublicURL", "Revokes public/external sharing for a file.", p("file", true, "File id")),
	m("files.sharedPublicURL", "Enables a file for public/external sharing.", p("file", true, "File id")),
	m("files.remote.add", "Adds a file from a remote service.",
		p("external_id", true, "Id from the remote service"), p("external_url", true, "URL to the remote file"), p("title", true, "File title"),
	),
	m("files.remote.info", "Retrieves info about a remote file.", p("file", false, "File id"), p("external_id", false, "External id")),
	m("files.remote.list", "Lists remote files.", p("channel", false, "Filter by channel"), p("cursor", false, "Pagination cursor")),
	m("files.remote.remove", "Removes a remote file.", p("file", false, "File id"), p("external_id", false, "External id")),
	m("files.remote.share", "Shares a remote file into a channel.", p("channels", true, "Comma-separated channel ids"), p("file", false, "File id"), p("external_id", false, "External id")),
	m("files.remote.update", "Updates info about a remote file.", p("file", false, "File id"), p("external_id", false, "External id"), p("title", false, "New title")),

	// --- functions (workflow builder custom functions) ---
	m("functions.completeSuccess", "Signals a function execution succeeded.", p("function_execution_id", true, "Execution id"), p("outputs", true, "JSON object of output values")),
	m("functions.completeError", "Signals a function execution failed.", p("function_execution_id", true, "Execution id"), p("error", true, "Error message")),

	// --- migration ---
	m("migration.exchange", "Exchanges a legacy user id for a new-format id.", p("users", true, "Comma-separated user ids"), p("to_old", false, "true/false")),

	// --- oauth ---
	m("oauth.v2.access", "Exchanges a temporary OAuth verifier code for an access token.",
		p("client_id", true, "App client id"), p("client_secret", true, "App client secret"),
		p("code", false, "Verifier code from the OAuth flow"), p("redirect_uri", false, "Redirect URI used"),
		p("grant_type", false, "authorization_code | refresh_token"), p("refresh_token", false, "Refresh token"),
	),
	m("oauth.v2.exchange", "Exchanges a legacy access token for a new expiring token pair.", p("client_id", true, "App client id"), p("client_secret", true, "App client secret"), p("token", true, "Legacy token")),
	deprecated("oauth.access", "Exchanges a temporary OAuth verifier code for a workspace token.", "use oauth.v2.access",
		p("client_id", true, "App client id"), p("client_secret", true, "App client secret"), p("code", true, "Verifier code"),
	),

	// --- openid.connect ---
	m("openid.connect.token", "Exchanges a code for an OpenID Connect token.",
		p("client_id", true, "App client id"), p("client_secret", true, "App client secret"), p("code", false, "Verifier code"), p("grant_type", false, "Grant type"),
	),
	m("openid.connect.userInfo", "Gets identity information for the authenticated user."),

	// --- pins ---
	m("pins.add", "Pins an item to a channel.", p("channel", true, "Channel id"), p("timestamp", false, "Message ts to pin")),
	m("pins.list", "Lists items pinned to a channel.", p("channel", true, "Channel id")),
	m("pins.remove", "Un-pins an item from a channel.", p("channel", true, "Channel id"), p("timestamp", false, "Message ts to unpin")),

	// --- reactions ---
	m("reactions.add", "Adds a reaction to an item.", p("channel", true, "Channel id"), p("timestamp", true, "Message ts"), p("name", true, "Emoji name")),
	m("reactions.get", "Gets reactions for an item.", p("channel", false, "Channel id"), p("timestamp", false, "Message ts"), p("full", false, "true/false")),
	m("reactions.list", "Lists reactions made by a user.", p("user", false, "User id"), p("cursor", false, "Pagination cursor"), p("limit", false, "Max results")),
	m("reactions.remove", "Removes a reaction from an item.", p("channel", true, "Channel id"), p("timestamp", true, "Message ts"), p("name", true, "Emoji name")),

	// --- reminders ---
	m("reminders.add", "Creates a reminder.", p("text", true, "Reminder text"), p("time", true, "Unix ts or natural-language time"), p("user", false, "User id to remind")),
	m("reminders.complete", "Marks a reminder as complete.", p("reminder", true, "Reminder id")),
	m("reminders.delete", "Deletes a reminder.", p("reminder", true, "Reminder id")),
	m("reminders.info", "Gets information about a reminder.", p("reminder", true, "Reminder id")),
	m("reminders.list", "Lists reminders for the calling user."),

	// --- rtm ---
	m("rtm.connect", "Starts a Real Time Messaging session.", p("batch_presence_aware", false, "true/false"), p("presence_sub", false, "true/false")),

	// --- search ---
	m("search.all", "Searches for messages and files matching a query.", p("query", true, "Search query"), p("cursor", false, "Pagination cursor"), p("count", false, "Max results per page")),
	m("search.files", "Searches for files matching a query.", p("query", true, "Search query"), p("cursor", false, "Pagination cursor"), p("count", false, "Max results per page")),
	m("search.messages", "Searches for messages matching a query.", p("query", true, "Search query"), p("cursor", false, "Pagination cursor"), p("count", false, "Max results per page"), p("sort", false, "score | timestamp")),

	// --- stars ---
	m("stars.add", "Adds a star to an item.", p("channel", false, "Channel id"), p("timestamp", false, "Message ts"), p("file", false, "File id")),
	m("stars.list", "Lists items starred by a user.", p("cursor", false, "Pagination cursor"), p("count", false, "Max results")),
	m("stars.remove", "Removes a star from an item.", p("channel", false, "Channel id"), p("timestamp", false, "Message ts"), p("file", false, "File id")),

	// --- team ---
	m("team.accessLogs", "Gets the access logs for users on a team.", p("before", false, "Unix ts upper bound"), p("cursor", false, "Pagination cursor")),
	m("team.billableInfo", "Gets billable users information for a team.", p("user", false, "Filter by user id")),
	m("team.billing.info", "Reports billing plan information for a team."),
	m("team.info", "Gets information about the current team.", p("team", false, "Team id")),
	m("team.integrationLogs", "Gets the integration logs for a team.", p("app_id", false, "Filter by app id"), p("user", false, "Filter by user id")),
	m("team.preferences.list", "Retrieves a team's Slack Connect preferences."),
	m("team.profile.get", "Retrieves a team's profile field definitions.", p("visibility", false, "all | visible | hidden")),

	// --- usergroups ---
	m("usergroups.create", "Creates a user group.", p("name", true, "Group name"), p("channels", false, "Comma-separated default channel ids"), p("handle", false, "Mention handle")),
	m("usergroups.disable", "Disables an existing user group.", p("usergroup", true, "User group id")),
	m("usergroups.enable", "Enables a previously disabled user group.", p("usergroup", true, "User group id")),
	m("usergroups.list", "Lists all user groups for a team.", p("include_disabled", false, "true/false"), p("include_users", false, "true/false")),
	m("usergroups.update", "Updates properties of a user group.", p("usergroup", true, "User group id"), p("name", false, "New name"), p("handle", false, "New handle")),
	m("usergroups.users.list", "Lists all users in a user group.", p("usergroup", true, "User group id")),
	m("usergroups.users.update", "Updates the list of users for a user group.", p("usergroup", true, "User group id"), p("users", true, "Comma-separated user ids")),

	// --- users ---
	m("users.conversations", "Lists conversations the calling user may access.",
		p("user", false, "User id"), p("types", false, "public_channel,private_channel,mpim,im"), p("cursor", false, "Pagination cursor"),
	),
	m("users.deletePhoto", "Deletes the current user's profile photo."),
	m("users.getPresence", "Gets user presence information.", p("user", true, "User id")),
	m("users.identity", "Gets a user's identity.", p("user", false, "User id (with identity.basic scope, ignored)")),
	m("users.info", "Gets information about a user.", p("user", true, "User id"), p("include_locale", false, "true/false")),
	m("users.list", "Lists all users in a workspace.", p("cursor", false, "Pagination cursor"), p("limit", false, "Max results")),
	m("users.lookupByEmail", "Finds a user by email.", p("email", true, "Email address")),
	m("users.setActive", "Marks a user as active.", p("user", false, "User id")),
	m("users.setPhoto", "Sets the current user's profile photo.", p("image", true, "Path to an image file to upload")),
	m("users.setPresence", "Manually sets user presence.", p("presence", true, "auto | away")),
	m("users.profile.get", "Retrieves a user's profile information.", p("user", false, "User id"), p("include_labels", false, "true/false")),
	m("users.profile.set", "Sets a user's profile fields.", p("user", false, "User id"), p("profile", false, "JSON profile object"), p("name", false, "Single field name"), p("value", false, "Single field value")),

	// --- views ---
	m("views.open", "Opens a modal view.", p("trigger_id", true, "Trigger id from an interaction"), p("view", true, "JSON view payload")),
	m("views.publish", "Publishes a static view for a user's App Home.", p("user_id", true, "User id"), p("view", true, "JSON view payload"), p("hash", false, "Expected hash for optimistic concurrency")),
	m("views.push", "Pushes a new view onto a modal stack.", p("trigger_id", true, "Trigger id"), p("view", true, "JSON view payload")),
	m("views.update", "Updates an existing view.", p("view", true, "JSON view payload"), p("view_id", false, "View id"), p("external_id", false, "External id"), p("hash", false, "Expected hash")),

	// --- workflows ---
	m("workflows.stepCompleted", "Signals a workflow extension step finished successfully.", p("workflow_step_execute_id", true, "Step execution id"), p("outputs", false, "JSON object of output values")),
	m("workflows.stepFailed", "Signals a workflow extension step failed.", p("workflow_step_execute_id", true, "Step execution id"), p("error", true, "JSON {message}")),
	m("workflows.updateStep", "Updates configuration for a workflow extension step.", p("workflow_step_edit_id", true, "Step edit id"), p("inputs", false, "JSON object"), p("outputs", false, "JSON array")),

	// ==================== admin.* (Enterprise Grid) ====================
	m("admin.apps.approve", "Approves an app for installation.", p("app_id", false, "App id"), p("request_id", false, "Request id"), p("team_id", false, "Team id")),
	m("admin.apps.approved.list", "Lists approved apps for an org/workspace.", p("cursor", false, "Pagination cursor"), p("limit", false, "Max results")),
	m("admin.apps.clearResolution", "Clears an app resolution restriction/approval.", p("app_id", true, "App id")),
	m("admin.apps.requests.list", "Lists app installation requests.", p("cursor", false, "Pagination cursor"), p("limit", false, "Max results")),
	m("admin.apps.restrict", "Restricts an app from installation.", p("app_id", false, "App id"), p("request_id", false, "Request id"), p("team_id", false, "Team id")),
	m("admin.apps.restricted.list", "Lists restricted apps for an org/workspace.", p("cursor", false, "Pagination cursor"), p("limit", false, "Max results")),
	m("admin.apps.uninstall", "Uninstalls an app from one or more workspaces.", p("app_id", true, "App id"), p("team_ids", false, "Comma-separated team ids")),
	m("admin.apps.activities.list", "Lists activities for a given app.", p("app_id", false, "App id"), p("cursor", false, "Pagination cursor")),

	m("admin.authPolicy.assignEntities", "Assigns entities to a specific authentication policy.", p("entity_ids", true, "JSON array of ids"), p("entity_type", true, "Entity type"), p("policy_name", true, "Policy name")),
	m("admin.authPolicy.getEntities", "Fetches entities assigned to an authentication policy.", p("policy_name", true, "Policy name"), p("cursor", false, "Pagination cursor")),
	m("admin.authPolicy.removeEntities", "Removes entities from an authentication policy.", p("entity_ids", true, "JSON array of ids"), p("entity_type", true, "Entity type"), p("policy_name", true, "Policy name")),

	m("admin.barriers.create", "Creates an information barrier.", p("barriered_from_usergroup_ids", true, "JSON array"), p("primary_usergroup_id", true, "Usergroup id"), p("restricted_subjects", true, "JSON array")),
	m("admin.barriers.delete", "Deletes an information barrier.", p("barrier_id", true, "Barrier id")),
	m("admin.barriers.list", "Lists information barriers."),
	m("admin.barriers.update", "Updates an information barrier.", p("barrier_id", true, "Barrier id")),

	m("admin.conversations.archive", "Archives a channel.", p("channel_id", true, "Channel id")),
	m("admin.conversations.convertToPrivate", "Converts a public channel to private.", p("channel_id", true, "Channel id")),
	m("admin.conversations.create", "Creates a channel for an org/workspace.", p("name", true, "Channel name"), p("is_private", false, "true/false"), p("team_id", false, "Team id")),
	m("admin.conversations.delete", "Deletes a channel.", p("channel_id", true, "Channel id")),
	m("admin.conversations.disconnectShared", "Disconnects a shared Slack Connect channel.", p("channel_id", true, "Channel id"), p("leaving_team_ids", false, "Comma-separated team ids")),
	m("admin.conversations.getConversationPrefs", "Gets channel preferences (posting/threads permissions).", p("channel_id", true, "Channel id")),
	m("admin.conversations.getTeams", "Gets workspaces a channel is shared with.", p("channel_id", true, "Channel id"), p("cursor", false, "Pagination cursor")),
	m("admin.conversations.invite", "Invites users to a channel.", p("channel_id", true, "Channel id"), p("user_ids", true, "Comma-separated user ids")),
	m("admin.conversations.rename", "Renames a channel.", p("channel_id", true, "Channel id"), p("name", true, "New name")),
	m("admin.conversations.search", "Searches org/workspace channels.", p("query", false, "Search text"), p("cursor", false, "Pagination cursor")),
	m("admin.conversations.setConversationPrefs", "Sets channel preferences.", p("channel_id", true, "Channel id"), p("prefs", true, "JSON preferences object")),
	m("admin.conversations.setTeams", "Sets which workspaces can access a channel.", p("channel_id", true, "Channel id"), p("team_id", false, "Team id"), p("target_team_ids", false, "Comma-separated team ids")),
	m("admin.conversations.unarchive", "Unarchives a channel.", p("channel_id", true, "Channel id")),
	m("admin.conversations.bulkArchive", "Archives multiple channels.", p("channel_ids", true, "JSON array of channel ids")),
	m("admin.conversations.bulkDelete", "Deletes multiple channels.", p("channel_ids", true, "JSON array of channel ids")),
	m("admin.conversations.bulkMove", "Moves multiple channels to another workspace.", p("channel_ids", true, "JSON array of channel ids"), p("target_team_id", true, "Destination team id")),
	m("admin.conversations.lookup", "Looks up channels matching filter criteria.", p("cursor", false, "Pagination cursor"), p("limit", false, "Max results")),
	m("admin.conversations.restrictAccess.addGroup", "Adds an allowlisted group to a channel.", p("channel_id", true, "Channel id"), p("group_id", true, "IDP group id"), p("team_id", false, "Team id")),
	m("admin.conversations.restrictAccess.listGroups", "Lists allowlisted groups for a channel.", p("channel_id", true, "Channel id"), p("team_id", false, "Team id")),
	m("admin.conversations.restrictAccess.removeGroup", "Removes an allowlisted group from a channel.", p("channel_id", true, "Channel id"), p("group_id", true, "IDP group id"), p("team_id", true, "Team id")),
	m("admin.conversations.ekm.listOriginalConnectedChannelInfo", "Lists EKM info for connected channels.", p("channel_ids", false, "Comma-separated channel ids"), p("team_ids", false, "Comma-separated team ids"), p("cursor", false, "Pagination cursor")),

	m("admin.emoji.add", "Adds a custom emoji.", p("name", true, "Emoji name"), p("url", true, "Image URL")),
	m("admin.emoji.addAlias", "Adds an alias for a custom emoji.", p("alias_for", true, "Existing emoji name"), p("name", true, "New alias name")),
	m("admin.emoji.list", "Lists custom emoji for an org/workspace.", p("cursor", false, "Pagination cursor")),
	m("admin.emoji.remove", "Removes a custom emoji.", p("name", true, "Emoji name")),
	m("admin.emoji.rename", "Renames a custom emoji.", p("name", true, "Current name"), p("new_name", true, "New name")),

	m("admin.inviteRequests.approve", "Approves a workspace invite request.", p("invite_request_id", true, "Invite request id")),
	m("admin.inviteRequests.deny", "Denies a workspace invite request.", p("invite_request_id", true, "Invite request id")),
	m("admin.inviteRequests.list", "Lists pending workspace invite requests.", p("cursor", false, "Pagination cursor"), p("team_id", false, "Team id")),
	m("admin.inviteRequests.approved.list", "Lists approved invite requests.", p("cursor", false, "Pagination cursor")),
	m("admin.inviteRequests.denied.list", "Lists denied invite requests.", p("cursor", false, "Pagination cursor")),

	m("admin.roles.addAssignments", "Adds users to a role.", p("role_id", true, "Role id"), p("entity_ids", true, "JSON array of entity ids"), p("user_ids", true, "JSON array of user ids")),
	m("admin.roles.listAssignments", "Lists role assignments.", p("role_ids", false, "JSON array of role ids"), p("cursor", false, "Pagination cursor")),
	m("admin.roles.removeAssignments", "Removes users from a role.", p("role_id", true, "Role id"), p("entity_ids", true, "JSON array of entity ids"), p("user_ids", true, "JSON array of user ids")),

	m("admin.teams.admins.list", "Lists all admins on a workspace.", p("team_id", true, "Team id"), p("cursor", false, "Pagination cursor")),
	m("admin.teams.create", "Creates a Grid workspace.", p("team_domain", true, "Workspace domain"), p("team_name", true, "Workspace name")),
	m("admin.teams.list", "Lists all workspaces in an org.", p("cursor", false, "Pagination cursor"), p("limit", false, "Max results")),
	m("admin.teams.owners.list", "Lists all owners on a workspace.", p("team_id", true, "Team id"), p("cursor", false, "Pagination cursor")),
	m("admin.teams.settings.info", "Fetches workspace settings.", p("team_id", true, "Team id")),
	m("admin.teams.settings.setDefaultChannels", "Sets default channels new members join.", p("team_id", true, "Team id"), p("channel_ids", true, "Comma-separated channel ids")),
	m("admin.teams.settings.setDescription", "Sets a workspace's description.", p("team_id", true, "Team id"), p("description", true, "New description")),
	m("admin.teams.settings.setDiscoverability", "Sets a workspace's discoverability.", p("team_id", true, "Team id"), p("discoverability", true, "open | invite_only | closed | unlisted")),
	m("admin.teams.settings.setIcon", "Sets a workspace's icon.", p("team_id", true, "Team id"), p("image_url", true, "Icon image URL")),
	m("admin.teams.settings.setName", "Sets a workspace's name.", p("team_id", true, "Team id"), p("name", true, "New name")),

	m("admin.usergroups.addChannels", "Adds channels to a user group.", p("usergroup_id", true, "Usergroup id"), p("channel_ids", true, "Comma-separated channel ids")),
	m("admin.usergroups.addTeams", "Associates workspaces with a user group.", p("usergroup_id", true, "Usergroup id"), p("team_ids", true, "Comma-separated team ids")),
	m("admin.usergroups.listChannels", "Lists channels associated with a user group.", p("usergroup_id", true, "Usergroup id"), p("team_id", false, "Team id")),
	m("admin.usergroups.removeChannels", "Removes channels from a user group.", p("usergroup_id", true, "Usergroup id"), p("channel_ids", true, "Comma-separated channel ids")),

	m("admin.users.assign", "Adds a user to a workspace.", p("team_id", true, "Team id"), p("user_id", true, "User id")),
	m("admin.users.invite", "Invites a user to a workspace.", p("team_id", true, "Team id"), p("email", true, "Email address"), p("channel_ids", false, "Comma-separated channel ids")),
	m("admin.users.list", "Lists users on a workspace.", p("team_id", true, "Team id"), p("cursor", false, "Pagination cursor")),
	m("admin.users.remove", "Removes a user from a workspace.", p("team_id", true, "Team id"), p("user_id", true, "User id")),
	m("admin.users.setAdmin", "Sets a user as an admin.", p("team_id", true, "Team id"), p("user_id", true, "User id")),
	m("admin.users.setExpiration", "Sets an expiration for a guest user.", p("team_id", true, "Team id"), p("user_id", true, "User id"), p("expiration_ts", true, "Unix expiration timestamp")),
	m("admin.users.setOwner", "Sets a user as an owner.", p("team_id", true, "Team id"), p("user_id", true, "User id")),
	m("admin.users.setRegular", "Sets a user as a regular member.", p("team_id", true, "Team id"), p("user_id", true, "User id")),
	m("admin.users.session.getSettings", "Gets session settings for users.", p("user_ids", true, "JSON array of user ids")),
	m("admin.users.session.invalidate", "Invalidates a session for a user.", p("session_id", true, "Session id"), p("team_id", true, "Team id")),
	m("admin.users.session.list", "Lists active user sessions.", p("team_id", false, "Team id"), p("cursor", false, "Pagination cursor")),
	m("admin.users.session.reset", "Resets sessions for a user.", p("user_id", true, "User id"), p("mobile_only", false, "true/false"), p("web_only", false, "true/false")),
	m("admin.users.session.resetBulk", "Resets sessions for multiple users.", p("user_ids", true, "JSON array of user ids")),
	m("admin.users.session.setSettings", "Sets session settings for users.", p("user_ids", true, "JSON array of user ids"), p("desktop_app_browser_quit", false, "true/false")),
	m("admin.users.unsupportedVersions.export", "Starts an export of users on unsupported app versions.", p("date_end_of_support", false, "Unix ts"), p("date_sessions_started", false, "Unix ts")),

	m("admin.workflows.collaborators.add", "Adds collaborators to workflows.", p("collaborator_ids", true, "JSON array of user ids"), p("workflow_ids", true, "JSON array of workflow ids")),
	m("admin.workflows.collaborators.remove", "Removes collaborators from workflows.", p("collaborator_ids", true, "JSON array of user ids"), p("workflow_ids", true, "JSON array of workflow ids")),
	m("admin.workflows.permissions.lookup", "Looks up run permissions for workflows.", p("workflow_ids", true, "JSON array of workflow ids")),
	m("admin.workflows.search", "Searches workflows in an org/workspace.", p("cursor", false, "Pagination cursor"), p("query", false, "Search text")),
	m("admin.workflows.triggers.types.permissions.lookup", "Looks up trigger type permissions for workflows.", p("trigger_type", true, "Trigger type"), p("workflow_ids", true, "JSON array of workflow ids")),
	m("admin.workflows.unpublish", "Unpublishes workflows.", p("workflow_ids", true, "JSON array of workflow ids")),
}

// registry provides fast lookups over Methods.
type registry struct {
	byName map[string]Method
}

var reg = buildRegistry()

func buildRegistry() *registry {
	r := &registry{byName: make(map[string]Method, len(Methods))}
	for _, meth := range Methods {
		r.byName[meth.Name] = meth
	}
	return r
}

// Lookup returns the registered Method for name, if known.
func Lookup(name string) (Method, bool) {
	meth, ok := reg.byName[name]
	return meth, ok
}

// All returns all registered methods sorted by name.
func All() []Method {
	out := make([]Method, len(Methods))
	copy(out, Methods)
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}
