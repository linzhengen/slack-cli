# Worked example: a full assistant thread turn

This walks through handling one `assistant_thread_started` event and one
user message, purely in terms of the `slack-cli` calls involved — the
event delivery itself (Events API / Socket Mode subscription) is outside
this CLI's scope and handled by your app's event listener.

## 1. Thread starts (`assistant_thread_started`)

The event payload gives you `channel_id` and `thread_ts`. Set a title and
offer a couple of starting points:

```sh
channel_id=D0123456
thread_ts=1722600000.000100

slack-cli assistant threads set-title \
  --channel_id "$channel_id" --thread_ts "$thread_ts" \
  --title "Ask about deploys"

slack-cli assistant threads set-suggested-prompts \
  --channel_id "$channel_id" --thread_ts "$thread_ts" \
  --title "Try asking:" \
  --prompts '[
    {"title":"Latest deploy status","message":"What'\''s the status of the latest deploy?"},
    {"title":"Recent failures","message":"Have there been any failed deploys this week?"}
  ]'
```

## 2. User sends a message (`message.im`)

Show a status while you work, then clear it and reply:

```sh
slack-cli assistant threads set-status \
  --channel_id "$channel_id" --thread_ts "$thread_ts" \
  --status "Checking deploy history..."

# ... your app does the actual lookup here ...

slack-cli assistant threads set-status \
  --channel_id "$channel_id" --thread_ts "$thread_ts" --status ""

slack-cli chat post-message \
  --channel "$channel_id" --thread_ts "$thread_ts" \
  --text "The last deploy (build #482) finished 12 minutes ago and is healthy." \
  --blocks '[
    {"type":"section","text":{"type":"mrkdwn","text":"*Build #482* finished 12 minutes ago — :white_check_mark: healthy"}},
    {"type":"context","elements":[{"type":"mrkdwn","text":"Checked just now"}]}
  ]'
```

## Notes

- Always clear the status (`--status ""`) before or as part of sending the
  final reply — a status left set makes the assistant look stuck.
- Suggested prompts are only useful right when the thread has little or no
  history yet; don't re-send them after several turns.
- If the work will take more than a couple of seconds, update the status
  message as it progresses rather than leaving one static string the whole
  time — it's meant to communicate real progress, not just "hi".
