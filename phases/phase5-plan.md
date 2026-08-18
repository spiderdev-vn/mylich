# Phase 5 — Notifications, Webhooks & Automation

**Goal:** make `lich-server` an event-driven calendar backend that can notify external services when calendar data changes.

At this point:

```text
                 ┌──────────────┐
                 │   lich-tui   │
                 └──────┬───────┘
                        │
                 Local-first sync
                        │
                        ▼
                 ┌──────────────┐
                 │ lich-server  │
                 └──────┬───────┘
                        │
                  Event System
                        │
          ┌─────────────┼─────────────┐
          ▼             ▼             ▼
       Gotify        Webhook       Future...
```

The key idea:

> **Notifications are consumers of calendar events, not part of the calendar core.**

---

# 1. Event-driven architecture

Introduce a server-side event system.

```text
lich-server/
└── src/
    ├── events/
    │   ├── event-bus.ts
    │   ├── events.ts
    │   └── handlers/
    │
    ├── notifications/
    ├── webhooks/
    └── integrations/
```

Calendar operation:

```text
POST /events
    ↓
EventService
    ↓
Database
    ↓
Domain Event
    ↓
Event Bus
    ├── Webhook
    ├── Gotify
    └── Future integration
```

The HTTP request should **not wait for external notifications**.

---

# 2. Domain events

Define stable internal events:

```text
calendar.created
calendar.updated
calendar.deleted

event.created
event.updated
event.deleted

sync.completed
sync.failed
conflict.created
```

Example:

```json
{
  "id": "evt_123",
  "type": "event.created",
  "user_id": "usr_123",
  "timestamp": "2026-08-18T10:00:00Z",
  "data": {
    "event_id": "..."
  }
}
```

Keep the event envelope consistent.

---

# 3. Outbox pattern

Don't publish directly after a DB write:

```text
DB write
   ↓
publish
```

because this can fail:

```text
DB write ✓
publish ✗
```

Instead:

```text
Database transaction
    │
    ├── calendar/event change
    └── outbox event
             ↓
        Event Worker
             ↓
       external systems
```

Add:

```text
outbox_events
├── id
├── type
├── aggregate_type
├── aggregate_id
├── payload
├── created_at
├── processed_at
└── attempts
```

This makes notifications durable.

---

# 4. Outbox worker

Worker:

```text
outbox_events
      ↓
pending event
      ↓
process
      ↓
┌─────┴─────┐
│           │
success    failure
│           │
▼           ▼
processed   retry
```

The worker should survive:

- server restart
- network failure
- Gotify unavailable
- webhook timeout

---

# 5. Webhooks

Allow users to configure:

```bash
lich webhook add ...
```

or through the server API.

Webhook model:

```text
webhooks
├── id
├── user_id
├── url
├── secret
├── enabled
├── created_at
└── updated_at
```

Subscriptions:

```text
webhook_events
├── webhook_id
└── event_type
```

Example:

```text
event.created
event.updated
event.deleted
```

---

# 6. Webhook payload

Example:

```json
{
  "id": "evt_123",
  "type": "event.created",
  "timestamp": "2026-08-18T10:30:00Z",
  "data": {
    "event_id": "event_456"
  }
}
```

Avoid exposing unnecessary user information.

The webhook should receive the minimum required data.

---

# 7. Webhook authentication

Use HMAC signatures.

Example headers:

```http
X-Lich-Event: event.created
X-Lich-Delivery: delivery_123
X-Lich-Signature: sha256=...
```

Signature:

```text
HMAC-SHA256(
    webhook_secret,
    raw_request_body
)
```

Webhook consumers can verify authenticity.

---

# 8. Webhook delivery

Never assume:

```text
HTTP 200
```

is always returned quickly.

Set:

```text
timeout = 5-10 seconds
```

Then retry transient failures.

Example:

```text
Attempt 1
   ↓
timeout
   ↓
Attempt 2
   ↓
500
   ↓
Attempt 3
   ↓
success
```

Retry only appropriate failures:

```text
408
429
5xx
network error
timeout
```

Don't endlessly retry:

```text
400
401
403
404
```

---

# 9. Webhook delivery records

Add:

```text
webhook_deliveries
├── id
├── webhook_id
├── event_id
├── status
├── attempts
├── response_status
├── last_error
├── next_attempt_at
├── created_at
└── delivered_at
```

This makes debugging possible.

CLI:

```bash
lich webhook deliveries
```

could eventually show:

```text
✓ event.created    200
✓ event.updated    200
✗ event.deleted    500 · retrying
```

---

# 10. Gotify integration

Gotify should use the same notification abstraction.

```text
NotificationProvider
        │
        ├── Gotify
        ├── Webhook
        └── Future...
```

Example:

```text
Gotify
  URL
  Token
  Priority
```

Configuration:

```text
notifications
├── id
├── user_id
├── provider
├── enabled
└── config
```

Provider-specific configuration can be JSON initially.

---

# 11. Gotify events

Useful notifications:

```text
event.created
event.updated
event.deleted

sync.failed
conflict.created
```

For example:

```text
Calendar Sync

Google synchronization failed.
Reason: authorization expired.
```

Or:

```text
Calendar

"Dinner" starts in 15 minutes.
```

The second type introduces scheduled notifications, which should be implemented separately.

---

# 12. Scheduled notifications

Introduce a scheduler:

```text
lich-server
    │
    ├── Event System
    │
    └── Scheduler
          ↓
       reminders
```

Event:

```text
Dinner
19:00
```

Reminder:

```text
18:45
```

Model:

```text
event_reminders
├── id
├── event_id
├── minutes_before
├── provider
└── enabled
```

Example:

```text
15 minutes before
30 minutes before
1 hour before
```

---

# 13. Reminder processing

Don't create a permanent timer for every event.

Bad:

```text
100,000 events
100,000 timers
```

Instead query due reminders:

```text
SELECT ...
WHERE notify_at <= now()
AND processed_at IS NULL
```

Run a scheduler periodically.

For example:

```text
every 30 seconds
```

Then:

```text
due reminder
    ↓
notification job
    ↓
Gotify/Webhook
```

---

# 14. Notification queue

Use a durable queue:

```text
notification_jobs
├── id
├── provider
├── event_id
├── payload
├── status
├── attempts
├── next_attempt_at
└── delivered_at
```

This means:

```text
Google slow
```

doesn't affect:

```text
Gotify notification
```

and vice versa.

---

# 15. CLI configuration

Useful commands:

```bash
lich notify list
lich notify add gotify
lich notify remove <id>

lich webhook list
lich webhook add
lich webhook remove <id>

lich reminder list
```

Configuration should preferably be managed through the server rather than directly editing files.

---

# 16. TUI

Add a settings screen:

```text
Settings

Integrations
  Google Calendar      Connected
  Gotify               Connected
  Webhooks             2

Notifications
  Event reminders      Enabled
  Sync failures        Enabled
  Conflicts             Enabled
```

Don't make the TUI responsible for executing notifications.

It only configures and displays them.

---

# 17. Automation

Phase 5 can introduce basic rules.

Example:

```text
WHEN event.created
IF calendar == "Work"
THEN webhook "work-automation"
```

Model:

```text
automation_rules
├── id
├── user_id
├── trigger
├── conditions
├── action
└── enabled
```

Keep the first version intentionally simple.

Possible triggers:

```text
event.created
event.updated
event.deleted
event.starting
sync.failed
conflict.created
```

Actions:

```text
send_webhook
send_gotify
```

---

# 18. Security

Webhooks create a new attack surface.

Implement:

- [ ] HMAC signatures
- [ ] HTTPS recommended
- [ ] SSRF protection
- [ ] URL validation
- [ ] Request timeout
- [ ] Redirect restrictions
- [ ] Rate limiting
- [ ] Secret rotation
- [ ] Don't log secrets
- [ ] Don't expose webhook secrets through API

Especially protect against:

```text
http://localhost:3000
http://127.0.0.1
http://169.254.x.x
```

and private network targets.

---

# 19. Testing

### Event bus

- [ ] Event emitted
- [ ] Multiple subscribers
- [ ] Subscriber failure doesn't crash server
- [ ] Event payload validation

### Outbox

- [ ] Event created atomically with DB change
- [ ] Worker processes event
- [ ] Restart recovery
- [ ] Retry
- [ ] Failed event remains inspectable

### Webhooks

- [ ] Correct payload
- [ ] HMAC signature
- [ ] Successful delivery
- [ ] Timeout
- [ ] 500 retry
- [ ] 429 retry
- [ ] 400 no retry
- [ ] Duplicate delivery handling
- [ ] Secret rotation

### Gotify

- [ ] Successful notification
- [ ] Authentication failure
- [ ] Network failure
- [ ] Retry
- [ ] Disabled integration

### Reminders

- [ ] Correct notification time
- [ ] No duplicate notification
- [ ] Event deletion cancels reminder
- [ ] Event update recalculates reminder
- [ ] Server restart doesn't duplicate reminders

### Security

- [ ] SSRF protection
- [ ] Invalid webhook URL
- [ ] Private IP rejection
- [ ] Secret not returned
- [ ] Secret not logged

---

# 20. E2E test

The main scenario:

```text
Create event
    ↓
Event saved
    ↓
Outbox event created
    ↓
Worker processes event
    ↓
Webhook receives event
    ↓
HMAC verified
```

Then:

```text
Create event
    ↓
Reminder scheduled
    ↓
Time advances
    ↓
Notification job created
    ↓
Gotify receives notification
```

And failure:

```text
Webhook unavailable
    ↓
Delivery fails
    ↓
Retry
    ↓
Server restart
    ↓
Retry resumes
    ↓
Webhook eventually receives event
```

---

# 21. Deliverables

### `lich-server`

- [ ] Domain event system
- [ ] Transactional outbox
- [ ] Outbox worker
- [ ] Webhook integration
- [ ] HMAC signing
- [ ] Webhook retry system
- [ ] Gotify integration
- [ ] Notification queue
- [ ] Reminder scheduler
- [ ] Basic automation rules
- [ ] Delivery history
- [ ] SSRF protection

### `lich-tui`

- [ ] Notification configuration
- [ ] Webhook configuration
- [ ] Gotify configuration
- [ ] Reminder configuration
- [ ] Notification status
- [ ] Delivery/error inspection

### Documentation

- [ ] Event catalog
- [ ] Webhook API
- [ ] Webhook signature specification
- [ ] Retry policy
- [ ] Gotify setup
- [ ] Automation rules
- [ ] Security model

---

# 22. Definition of Done

A complete flow:

```text
                 User
                   │
                   ▼
              lich-tui
                   │
                   ▼
             lich-server
                   │
              DB transaction
                   │
          ┌────────┴────────┐
          │                 │
       Calendar          Outbox
          │                 │
          │                 ▼
          │              Worker
          │                 │
          │        ┌────────┴────────┐
          │        ▼                 ▼
          │     Webhook           Gotify
          │
          ▼
       Reminder
          │
          ▼
      Notification
```

The important guarantee is:

> **Calendar operations remain independent from notification delivery.**

If Gotify is down, the calendar still works.
If a webhook is broken, Google sync still works.
If Google is unavailable, local-first Lich still works.

By the end of Phase 5, `lich-server` becomes more than a calendar API: it is the **event-driven backend for your own calendar ecosystem**, while `lich` remains the fast local-first interface.
