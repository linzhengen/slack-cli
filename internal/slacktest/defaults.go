package slacktest

import "encoding/json"

// defaultResponse implements built-in realistic behavior for the Slack Web
// API methods slack-cli's e2e tests lean on most. Anything not listed here
// gets a bare {"ok":true} — good enough for a smoke-test call, and always
// overridable per test via Handle/QueueResponse/QueueError.
func (s *Server) defaultResponse(method string, form map[string]string) any {
	switch method {
	case "auth.test":
		return map[string]any{
			"ok": true, "url": s.URL + "/", "team": "Test Team",
			"user": "testbot", "team_id": "T0TEST0001", "user_id": "U0TESTBOT1", "bot_id": "B0TESTBOT1",
		}

	case "chat.postMessage", "chat.postEphemeral", "chat.meMessage":
		ts := s.nextTS()
		return map[string]any{
			"ok": true, "channel": form["channel"], "ts": ts,
			"message": map[string]any{"type": "message", "text": form["text"], "ts": ts, "user": "U0TESTBOT1"},
		}

	case "chat.update":
		return map[string]any{
			"ok": true, "channel": form["channel"], "ts": form["ts"], "text": form["text"],
		}

	case "chat.delete":
		return map[string]any{"ok": true, "channel": form["channel"], "ts": form["ts"]}

	case "chat.scheduleMessage":
		return map[string]any{
			"ok": true, "channel": form["channel"], "scheduled_message_id": s.nextID("Q"), "post_at": form["post_at"],
		}

	case "conversations.create":
		id := s.nextID("C")
		return map[string]any{
			"ok": true,
			"channel": map[string]any{
				"id": id, "name": form["name"], "is_private": form["is_private"] == "true", "is_channel": true,
			},
		}

	case "conversations.info":
		id := form["channel"]
		if id == "" {
			id = s.nextID("C")
		}
		return map[string]any{
			"ok":      true,
			"channel": map[string]any{"id": id, "name": "test-channel", "is_channel": true},
		}

	case "conversations.list":
		return map[string]any{
			"ok": true,
			"channels": []map[string]any{
				{"id": "C0TESTCHAN1", "name": "general", "is_channel": true},
				{"id": "C0TESTCHAN2", "name": "random", "is_channel": true},
			},
			"response_metadata": map[string]any{"next_cursor": ""},
		}

	case "conversations.replies":
		ts := form["ts"]
		return map[string]any{
			"ok": true,
			"messages": []map[string]any{
				{"type": "message", "ts": ts, "text": "parent message", "user": "U0TESTUSER1"},
			},
			"response_metadata": map[string]any{"next_cursor": ""},
		}

	case "conversations.history":
		return map[string]any{
			"ok": true,
			"messages": []map[string]any{
				{"type": "message", "ts": s.nextTS(), "text": "hello", "user": "U0TESTUSER1"},
			},
			"response_metadata": map[string]any{"next_cursor": ""},
		}

	case "users.info":
		id := form["user"]
		if id == "" {
			id = s.nextID("U")
		}
		return map[string]any{
			"ok":   true,
			"user": fakeUser(id),
		}

	case "users.list":
		return map[string]any{
			"ok":                true,
			"members":           []map[string]any{fakeUser("U0TESTUSER1"), fakeUser("U0TESTUSER2")},
			"response_metadata": map[string]any{"next_cursor": ""},
		}

	case "users.lookupByEmail":
		return map[string]any{"ok": true, "user": fakeUser("U0LOOKEDUP1")}

	case "files.getUploadURLExternal":
		id := s.nextID("F")
		return map[string]any{
			"ok": true, "upload_url": s.URL + "/_upload/" + id, "file_id": id,
		}

	case "files.completeUploadExternal":
		var files []map[string]any
		if raw, ok := form["files"]; ok {
			var requested []struct {
				ID    string `json:"id"`
				Title string `json:"title"`
			}
			_ = json.Unmarshal([]byte(raw), &requested)
			for _, f := range requested {
				files = append(files, map[string]any{
					"id": f.ID, "title": f.Title, "name": f.Title, "permalink": s.URL + "/files/" + f.ID,
				})
			}
		}
		return map[string]any{"ok": true, "files": files}

	default:
		return map[string]any{"ok": true}
	}
}

func fakeUser(id string) map[string]any {
	return map[string]any{
		"id": id, "name": "test.user", "real_name": "Test User", "is_bot": false, "deleted": false,
		"profile": map[string]any{"email": "test.user@example.com", "display_name": "Test User"},
	}
}
