---
name: slack-workflows
version: 1.0.0
description: "Signal completion/failure of Workflow Builder custom steps and custom functions, and update a workflow step's configuration. Use when the user is building a Slack app that acts as a step in Workflow Builder (a 'workflow app step' or 'custom function'), not for reading or triggering workflows interactively as an end user (Workflow Builder itself has no public 'run this workflow now' API)."
metadata:
  requires:
    bins: ["slack-cli"]
  cliHelp: "slack-cli workflows --help"
---

# slack-workflows: workflow steps and custom functions

Read `slack-shared` first for auth and output/error conventions.

Slack has two related but distinct extension points here, both covered by
this skill:

- **Legacy workflow steps** (`workflows.*`) — an app registers a step type
  that a workflow author can drag into Workflow Builder; when that step
  runs, your app gets an execution id and must report back.
- **Custom functions** (`functions.*`) — the modern replacement, defined
  via your app manifest and invoked the same way, reporting back success
  or error.

Both are callback-style: Slack invokes your app (via an event, outside this
CLI's scope), your app does its work, then calls one of these to report the
outcome — there's no "list workflows" or "run this workflow" method to
call proactively.

## Core commands

| Task | Command |
|---|---|
| Report a legacy workflow step succeeded | `slack-cli workflows step-completed --workflow_step_execute_id <id> --outputs '{"result":"ok"}'` |
| Report a legacy workflow step failed | `slack-cli workflows step-failed --workflow_step_execute_id <id> --error '{"message":"..."}'` |
| Update a step's configuration (in the editor) | `slack-cli workflows update-step --workflow_step_edit_id <id> --inputs '{"channel":{"value":"C0123"}}'` |
| Report a custom function succeeded | `slack-cli functions complete-success --function_execution_id <id> --outputs '{"result":"ok"}'` |
| Report a custom function failed | `slack-cli functions complete-error --function_execution_id <id> --error "what went wrong"` |

## Key concepts

- Every id here (`workflow_step_execute_id`, `workflow_step_edit_id`,
  `function_execution_id`) comes from the triggering event payload — none
  of them are something you look up or construct yourself.
- `outputs` must match the output shape your app declared for the step or
  function in its manifest; a mismatch fails the call with
  `invalid_arguments` even though the JSON itself is well-formed.
- Reporting completion/error is **one-shot per execution** — calling
  `step-completed`/`complete-success` twice, or completing and then
  failing, for the same execution id is an error.
- New app development should prefer custom functions (`functions.*`) over
  legacy workflow steps — Slack's guidance is that workflow steps are the
  older mechanism.

## Common OAuth scopes

- `workflow.steps:execute` — legacy `workflows.*` step completion
- Custom functions are authorized implicitly through the app manifest
  (`functions.callback_id`), not a separate OAuth scope.

## Examples

A custom function that fetches a value and reports it back:

```sh
slack-cli functions complete-success \
  --function_execution_id Fx0123456 \
  --outputs '{"status":"deployed","version":"1.4.2"}'
```

## Related

- `slack-shared` — auth, output/error format
- `slack-ai-assistant` — for conversational (assistant-thread) apps rather than workflow-triggered automation
- `slack-messaging` — a workflow step's implementation often ends with posting a message about what it did
