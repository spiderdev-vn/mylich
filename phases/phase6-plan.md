# Phase 6 — Automation & Extensibility

**Goal:** turn Lich into a programmable calendar platform without coupling the core calendar system to every future feature.

```text
                         Lich
                          │
                    ┌─────▼─────┐
                    │ Core      │
                    │ Calendar  │
                    └─────┬─────┘
                          │
                     Event Bus
                          │
          ┌───────────────┼────────────────┐
          ▼               ▼                ▼
      Automation       Webhooks         Plugins
          │
    ┌─────┼─────┐
    ▼     ▼     ▼
 Gotify  HTTP  Actions
```

Phase 5 established events/notifications. Phase 6 makes those events **programmable**.

---

# 1. Automation Engine

Introduce:

```text
lich-server/src/automation/

├── engine.ts
├── rules.ts
├── conditions.ts
├── actions.ts
├── executor.ts
└── scheduler.ts
```

A rule:

```text
WHEN <trigger>
IF <conditions>
THEN <actions>
```

Example:

```text
WHEN event.created
IF calendar == "Work"
THEN send_gotify
```

---

# 2. Rule model

Database:

```text
automation_rules
├── id
├── user_id
├── name
├── enabled
├── trigger
├── conditions
├── actions
├── created_at
└── updated_at
```

Example:

```json
{
  "name": "Notify work events",
  "trigger": "event.created",
  "conditions": [
    {
      "field": "calendar_id",
      "operator": "equals",
      "value": "work"
    }
  ],
  "actions": [
    {
      "type": "gotify",
      "priority": 5
    }
  ]
}
```

Keep rules as data rather than hard-coded TypeScript.

---

# 3. Triggers

Initial triggers:

```text
event.created
event.updated
event.deleted

event.starting

calendar.created
calendar.updated
calendar.deleted

sync.failed
conflict.created

reminder.due
```

Later:

```text
day.started
day.ended
```

---

# 4. Conditions

Build a small condition engine.

Operators:

```text
equals
not_equals
contains
starts_with
ends_with

greater_than
less_than
before
after

in
not_in
exists
```

Example:

```text
calendar == "work"
```

```text
title contains "meeting"
```

```text
start_at before 18:00
```

```text
location exists
```

---

# 5. Compound conditions

Support:

```text
AND
OR
NOT
```

Example:

```text
calendar == "work"
AND
title contains "meeting"
```

Representation:

```json
{
  "and": [
    {
      "field": "calendar_id",
      "operator": "equals",
      "value": "work"
    },
    {
      "field": "title",
      "operator": "contains",
      "value": "meeting"
    }
  ]
}
```

Don't build a complicated expression language yet.

A JSON-based AST is easier to validate and version.

---

# 6. Actions

Initial actions:

```text
send_gotify
send_webhook
create_event
update_event
delete_event
```

Potential future:

```text
send_email
run_command
HTTP request
```

Be very careful with arbitrary command execution.

**Do not implement arbitrary shell execution in the first version.**

---

# 7. Action execution

```text
Event
 ↓
Matching rules
 ↓
Condition evaluation
 ↓
Action generation
 ↓
Action queue
 ↓
Worker
 ↓
Execution
```

Do not execute actions synchronously inside the event handler.

---

# 8. Automation queue

Add:

```text
automation_jobs
├── id
├── rule_id
├── event_id
├── action_type
├── payload
├── status
├── attempts
├── next_attempt_at
├── last_error
└── created_at
```

This follows the same durable-worker architecture established in Phase 5.

---

# 9. Idempotency

Automation can accidentally run twice.

Example:

```text
event.created
    ↓
rule
    ↓
Gotify
```

If the worker crashes after sending but before marking the job complete, it may send twice.

Give each execution a stable ID:

```text
automation_execution_id
```

Actions should use that ID where possible.

---

# 10. Automation execution history

Expose:

```bash
lich automation history
```

Example:

```text
Automation History

✓ Notify work event
  Dinner · 10:42

✓ Webhook
  Event updated · 10:44

✗ Notify reminder
  Gotify unavailable · retrying
```

Database:

```text
automation_executions
├── id
├── rule_id
├── trigger_event_id
├── status
├── started_at
├── completed_at
└── error
```

---

# 11. CLI

Add:

```bash
lich automation list
lich automation create
lich automation edit <id>
lich automation delete <id>
lich automation enable <id>
lich automation disable <id>

lich automation test <id>
lich automation history
```

Example:

```bash
lich automation create \
  --name "Work notifications" \
  --when event.created \
  --if 'calendar=work' \
  --then gotify
```

The exact CLI syntax can be refined after the underlying API is stable.

---

# 12. TUI automation editor

Eventually:

```text
┌─ Automation ──────────────────────────────┐
│ Name: Work Event Notification              │
│                                            │
│ WHEN                                       │
│   Event Created                             │
│                                            │
│ IF                                         │
│   Calendar = Work                           │
│                                            │
│ THEN                                       │
│   Send Gotify                               │
│                                            │
│ Enabled: Yes                                │
│                                            │
│             [ Save ] [ Cancel ]             │
└────────────────────────────────────────────┘
```

Don't make the first version overly visual.

A simple form is enough.

---

# 13. Automation API

Add:

```text
GET    /automations
POST   /automations
GET    /automations/:id
PATCH  /automations/:id
DELETE /automations/:id

POST   /automations/:id/test
GET    /automations/:id/history
```

Authorization:

```text
user → own automation only
```

---

# 14. Public API

This is where Lich becomes more interesting.

Expose a documented API:

```text
/api/v1/
```

Resources:

```text
calendars
events
reminders
integrations
automations
webhooks
```

Use stable API versions:

```text
/api/v1/events
```

rather than:

```text
/events
```

for the public API.

---

# 15. API tokens

Add personal API tokens:

```bash
lich token create
```

Example:

```text
Name: Home Assistant
Token: lich_xxxxxxxxx
```

Store only a hash:

```text
api_tokens
├── id
├── user_id
├── name
├── token_hash
├── last_used_at
├── expires_at
└── created_at
```

Never store the raw token.

Show it only once.

---

# 16. API scopes

Avoid giving every token full access.

Scopes:

```text
calendar:read
calendar:write

event:read
event:write

automation:read
automation:write

integration:read
```

Example:

```text
Home Assistant
  event:read
  event:write
```

This is much safer than a universal API key.

---

# 17. External integrations

Now external services can interact with Lich:

```text
Home Assistant
        │
        ▼
    Lich API
        │
        ▼
     Calendar
```

And:

```text
Lich
  │
  ▼
Webhook
  │
  ▼
External automation
```

This creates a two-way ecosystem.

---

# 18. Plugin architecture

Do **not** immediately build dynamic Go plugins.

For `lich-server`, Node.js makes a simpler architecture possible:

```text
src/integrations/
├── google/
├── gotify/
├── webhook/
└── ...
```

Use interfaces:

```ts
interface Integration {
  id: string;
  initialize(): Promise<void>;
}
```

and:

```ts
interface NotificationProvider {
  send(message: Notification): Promise<void>;
}
```

This gives extensibility without the complexity of loading arbitrary code.

---

# 19. Provider registry

Create:

```ts
class ProviderRegistry {
  register(provider: Provider): void;
  get(id: string): Provider;
}
```

Then:

```text
GoogleProvider
GotifyProvider
WebhookProvider
```

are registered at startup.

Future:

```text
DiscordProvider
TelegramProvider
CalDAVProvider
OutlookProvider
```

without changing the automation engine.

---

# 20. Scheduled automation

Extend the scheduler from Phase 5.

Example:

```text
Every Monday at 09:00
    ↓
create event
```

or:

```text
Every weekday
IF calendar == Work
THEN ...
```

Start with simple cron-like schedules.

Don't immediately build a complicated natural-language scheduler.

---

# 21. Event templates

Useful automation action:

```text
create_event_from_template
```

Example:

```text
Template:
Daily standup
09:00 → 09:30
Calendar: Work
```

Automation:

```text
weekday 1-5 at 08:00
        ↓
create Daily Standup
```

This makes Lich useful for recurring workflows.

---

# 22. Dry-run mode

Very useful for debugging:

```bash
lich automation test work-notification
```

Output:

```text
Trigger:
  event.created

Event:
  Work / Meeting

Conditions:
  calendar == work      ✓

Actions:
  send_gotify           ✓

Result:
  Would send notification
```

No actual action is executed.

---

# 23. Security boundaries

Automation becomes potentially dangerous.

Rules:

- [ ] Never execute arbitrary shell commands by default
- [ ] Validate all action parameters
- [ ] Restrict webhook targets
- [ ] Protect API tokens
- [ ] Enforce user ownership
- [ ] Rate-limit automation
- [ ] Prevent infinite loops
- [ ] Limit recursion
- [ ] Limit action execution frequency

---

# 24. Infinite-loop protection

Example:

```text
event.updated
   ↓
automation
   ↓
update event
   ↓
event.updated
   ↓
automation
   ↓
...
```

Prevent this using:

```text
causation_id
correlation_id
depth
```

Example:

```json
{
  "event_id": "...",
  "causation_id": "...",
  "depth": 2
}
```

Reject or stop execution after a configured depth.

---

# 25. Testing

### Rule engine

- [ ] Simple condition
- [ ] AND
- [ ] OR
- [ ] NOT
- [ ] Invalid condition
- [ ] Missing field
- [ ] Type mismatch

### Trigger matching

- [ ] Correct trigger executes
- [ ] Wrong trigger ignored
- [ ] Disabled rule ignored

### Actions

- [ ] Gotify
- [ ] Webhook
- [ ] Create event
- [ ] Update event
- [ ] Delete event

### Reliability

- [ ] Retry
- [ ] Idempotency
- [ ] Server restart
- [ ] Duplicate event
- [ ] Action failure
- [ ] Queue recovery

### Security

- [ ] User isolation
- [ ] API token scope
- [ ] Invalid token
- [ ] Automation cannot access another user's data
- [ ] SSRF protection
- [ ] Infinite-loop prevention

---

# 26. E2E acceptance test

```text
Create automation
       ↓
Create event
       ↓
Event emitted
       ↓
Automation matches
       ↓
Condition passes
       ↓
Action queued
       ↓
Worker executes
       ↓
Gotify/Webhook receives notification
       ↓
Execution marked successful
```

Then:

```text
Event
 ↓
Automation
 ↓
Update event
 ↓
Automation triggered again
 ↓
Loop detection
 ↓
Execution stopped
```

---

# 27. Deliverables

### `lich-server`

- [ ] Automation engine
- [ ] Rule model
- [ ] Condition engine
- [ ] Action engine
- [ ] Automation queue
- [ ] Execution history
- [ ] Scheduled automation
- [ ] Public API v1
- [ ] API tokens
- [ ] API scopes
- [ ] Provider registry
- [ ] Loop protection

### `lich-tui`

- [ ] Automation list
- [ ] Create/edit/delete automation
- [ ] Enable/disable
- [ ] Test automation
- [ ] Execution history
- [ ] API token management
- [ ] Integration management

### Documentation

- [ ] Automation specification
- [ ] Rule schema
- [ ] Condition operators
- [ ] Action types
- [ ] Public API documentation
- [ ] API authentication
- [ ] API scopes
- [ ] Integration developer guide

---

# 28. Definition of Done

The complete Phase 6 flow:

```text
                   Calendar Event
                         │
                         ▼
                    Event Bus
                         │
                         ▼
                  Automation Engine
                         │
                  ┌──────┴──────┐
                  │             │
              Conditions      Rules
                  │
                  ▼
                Actions
             ┌────┼────┐
             ▼    ▼    ▼
          Gotify Webhook API
```

And an external application can do:

```text
POST /api/v1/events
        ↓
      Lich
        ↓
    event.created
        ↓
    automation
        ↓
     webhook
        ↓
External application
```

while the original guarantees remain:

```text
Local-first
    +
Reliable sync
    +
External integrations
    +
Durable events
    +
Programmable automation
```

At this point, Lich is no longer just a calendar CLI/TUI. It has become a **local-first calendar platform with a programmable event system and public API**, while keeping the core calendar model relatively small and provider-agnostic.
