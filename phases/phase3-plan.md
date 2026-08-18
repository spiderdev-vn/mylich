# Phase 3 — Google Calendar Integration

**Goal:** make Google Calendar a first-class external integration while keeping **Lich as the source of truth** and keeping the CLI/TUI responsive.

```text
                    Lich
                      │
             ┌────────┴────────┐
             │                 │
       lich-server         lich-tui
             │                 │
             └────────┬────────┘
                      │
                 Sync Engine
                      │
                Google Adapter
                      │
                      ▼
              Google Calendar
```

The important distinction:

> **Lich owns the event. Google owns a representation of that event.**

---

# 1. Google integration architecture

Do not put Google API calls directly into generic event code.

Use an integration boundary:

```text
lich-server/
└── integrations/
    ├── integration.ts
    └── google/
        ├── google-client.ts
        ├── google-calendar.ts
        ├── google-events.ts
        └── google-mapper.ts
```

Conceptually:

```ts
interface CalendarProvider {
  listCalendars(): Promise<ExternalCalendar[]>;
  createEvent(event: Event): Promise<ExternalEvent>;
  updateEvent(event: Event): Promise<ExternalEvent>;
  deleteEvent(id: string): Promise<void>;
  listChanges(...): Promise<ExternalChange[]>;
}
```

Google implements this interface.

Later:

```text
Google
CalDAV
Outlook
...
```

can implement the same abstraction.

---

# 2. Decide where Google OAuth lives

Since the server owns authentication for Lich, keep the Google OAuth flow on the **server** too.

Preferred architecture:

```text
lich-tui
    │
    │ "Connect Google"
    ▼
lich-server
    │
    ▼
your website / OAuth flow
    │
    ▼
Google
```

The TUI should **never receive or store the Google client secret**.

The server stores the Google refresh token securely.

---

# 3. Account linking

Add an integration concept:

```text
integrations
├── id
├── user_id
├── provider
├── status
├── created_at
└── updated_at
```

Provider:

```text
google
```

Then credentials should live separately:

```text
integration_credentials
├── integration_id
├── access_token
├── refresh_token
├── expires_at
└── ...
```

Do not expose tokens through the API.

---

# 4. CLI authentication

User flow:

```bash
lich google connect
```

Expected behavior:

```text
Opening browser...

Google authentication
        ↓
User grants Calendar permission
        ↓
Google redirects to Lich
        ↓
Lich stores credentials
        ↓
CLI receives success
```

Then:

```bash
lich google status
```

```text
Google Calendar

Account: user@gmail.com
Status:  connected
Calendars: 3
Last sync: 10:42
```

Commands:

```bash
lich google connect
lich google disconnect
lich google status
lich google calendars
lich google sync
```

---

# 5. Google calendar mapping

A Lich calendar should not necessarily equal a Google calendar.

Create an explicit mapping:

```text
calendar_integrations
├── calendar_id
├── integration_id
├── external_calendar_id
├── sync_direction
└── enabled
```

Example:

```text
Lich "Personal"
      │
      └── Google "primary"

Lich "Work"
      │
      └── Google "work@example.com"
```

Sync direction:

```text
PUSH
PULL
BIDIRECTIONAL
```

Initially, support:

```text
BIDIRECTIONAL
```

---

# 6. External event mapping

Never put:

```text
google_event_id
```

directly into the core event table if you plan to support multiple integrations.

Use:

```text
event_integrations
├── event_id
├── integration_id
├── external_id
├── external_updated_at
└── metadata
```

Example:

```text
Lich Event
   │
   ├── Google → abc123
   ├── future CalDAV → xyz789
   └── ...
```

---

# 7. Event mapping

Create a dedicated mapper:

```text
Lich Event
    ↓
Google Event
```

Map:

```text
title           → summary
description     → description
start_at        → start
end_at          → end
location        → location
```

And reverse:

```text
Google Event
    ↓
Lich Event
```

Be explicit about fields that don't map perfectly.

For example:

- Google attendees
- Google reminders
- conferencing
- recurrence rules
- event colors
- transparency
- visibility

Don't force every Google-specific field into the core Lich model.

Store provider-specific metadata separately when necessary.

---

# 8. Initial synchronization

When connecting Google:

```bash
lich google sync
```

First synchronization:

```text
Google calendars
       ↓
calendar mapping
       ↓
Google events
       ↓
event matching
       ↓
Lich
```

The dangerous part is **duplicate detection**.

Don't blindly create everything.

---

# 9. Event identity

The safest identity is:

```text
integration_id + external_event_id
```

So:

```text
Google event abc123
```

always maps to the same Lich integration record.

Never use title/date as the primary identity.

Bad:

```text
"Dinner" + "19:00"
```

because users can have multiple identical events.

---

# 10. First sync strategy

I'd make the initial behavior explicit:

### Option A — Lich → Google

Existing Lich events are pushed to Google.

### Option B — Google → Lich

Existing Google events are imported.

### Option C — Merge

Both sides are imported and matched.

For v1, **don't silently merge everything**.

Provide:

```bash
lich google sync --direction pull
lich google sync --direction push
```

and eventually:

```bash
lich google sync --direction both
```

This makes destructive behavior much easier to reason about.

---

# 11. Ongoing synchronization

After initial setup:

```text
Lich
 │
 ├── local event created
 │        ↓
 │    sync queue
 │        ↓
 │    Google API
 │
 └── Google changes
          ↓
       sync pull
          ↓
       Lich event
```

Do not make Google API requests part of:

```bash
lich add ...
```

The existing Phase 2 sync architecture handles it.

---

# 12. Google change detection

Use Google's incremental synchronization mechanism rather than repeatedly downloading the entire calendar.

Conceptually:

```text
Google
   ↓
initial sync
   ↓
sync token
   ↓
store token
```

Later:

```text
sync token
    ↓
Google changes
    ↓
new events / updates / deletions
    ↓
new sync token
```

Store provider-specific sync state:

```text
integration_sync_state
├── integration_id
├── resource
├── cursor
└── updated_at
```

---

# 13. Sync state

Extend the existing Phase 2 system.

```text
                 Sync Engine
                     │
          ┌──────────┴──────────┐
          │                     │
      Lich Server           Google
          │                     │
          └──────────┬──────────┘
                     │
               Sync Operation
```

Operations:

```text
CREATE
UPDATE
DELETE
```

Provider:

```text
lich
google
```

Example:

```text
sync_jobs

event: 123
provider: google
operation: UPDATE
status: pending
```

---

# 14. Conflict resolution

This becomes much more important in Phase 3.

Example:

```text
Lich:
Dinner → 19:00

Google:
Dinner → 20:00
```

Both changed since the last sync.

Don't silently overwrite.

Initial policy could be:

```text
if only Lich changed:
    push Lich → Google

if only Google changed:
    pull Google → Lich

if both changed:
    CONFLICT
```

Store:

```text
conflicts
├── id
├── event_id
├── integration_id
├── local_snapshot
├── remote_snapshot
├── detected_at
└── resolved_at
```

---

# 15. Conflict CLI

Expose conflicts:

```bash
lich sync conflicts
```

Example:

```text
Conflicts

1. Dinner
   Lich:   Aug 20 · 19:00
   Google: Aug 20 · 20:00

   [l] Keep Lich
   [g] Keep Google
```

Later:

```bash
lich sync resolve <id> --keep lich
```

This can eventually become a TUI workflow.

---

# 16. Recurring events

Google Calendar makes recurrence unavoidable.

Support the concept in Lich:

```text
recurrence
```

rather than flattening every occurrence into separate events.

For example:

```text
RRULE:FREQ=WEEKLY;BYDAY=MO
```

But don't attempt to implement a full recurrence engine from scratch if an appropriate Go/Node library can safely handle the relevant standards.

Phase 3 should at least establish:

```text
recurrence_rule
```

and preserve it through Google synchronization.

---

# 17. Timezone handling

This is critical.

Examples:

```text
Lich:
2026-08-20 19:00
Asia/Ho_Chi_Minh

Google:
2026-08-20 19:00
Asia/Ho_Chi_Minh
```

Don't blindly convert everything to UTC before sending to Google.

Preserve event timezone semantics.

Test:

- Vietnam
- UTC
- DST timezone
- all-day events
- midnight events
- recurring events across DST

---

# 18. All-day events

Google has different semantics for:

```text
2026-08-20
```

versus:

```text
2026-08-20 00:00 → 2026-08-21 00:00
```

Model this explicitly.

Potential Lich representation:

```text
type: timed
```

or:

```text
type: all_day
```

Don't infer this solely from timestamps.

---

# 19. Deletions

Google deletion needs to propagate:

```text
Google deleted
      ↓
Lich marks deleted
```

and:

```text
Lich deleted
      ↓
Google deleted
```

This is why Phase 2 introduced `deleted_at`.

Never immediately destroy synchronization metadata if it is still required to propagate the deletion.

---

# 20. Rate limits and failures

Google API failures should be normal application states.

Handle:

```text
401
403
404
429
5xx
network timeout
expired token
```

Especially:

```text
429 Too Many Requests
```

Use backoff.

Never turn:

```text
Google unavailable
```

into:

```text
Lich unavailable
```

---

# 21. TUI integration

Add Google status:

```text
┌──────────────────────────────────────────────┐
│ August 2026                     Lich ●       │
│                              Google ✓       │
├──────────────────────────────────────────────┤
│                                              │
│        18     19     20     21               │
│              Dinner                          │
│                                              │
├──────────────────────────────────────────────┤
│ Google: synced · 2s ago                     │
└──────────────────────────────────────────────┘
```

During sync:

```text
Google: syncing...
```

Failure:

```text
Google: sync failed
```

But the calendar remains fully usable.

---

# 22. CLI experience

Useful commands:

```bash
lich google connect
lich google status
lich google calendars
lich google sync
lich google disconnect
```

General sync:

```bash
lich sync
lich sync status
lich sync conflicts
```

Inspect an event:

```bash
lich event <id>
```

could show:

```text
Dinner

Lich
  Aug 20 · 19:00

Google
  Personal · abc123

Sync
  ✓ synced
```

---

# 23. Server API additions

Potential endpoints:

```text
GET    /integrations
POST   /integrations/google
DELETE /integrations/google/:id

GET    /integrations/google/calendars
POST   /integrations/google/sync
GET    /integrations/google/status
```

However, keep OAuth callbacks server-side:

```text
GET /auth/google/callback
```

The TUI should not directly communicate with Google's OAuth callback.

---

# 24. Security

Google credentials are sensitive.

Rules:

- [ ] Never return refresh tokens through API
- [ ] Never log access/refresh tokens
- [ ] Never store Google client secrets in `lich-tui`
- [ ] Encrypt credentials at rest where practical
- [ ] Scope Google permissions to required Calendar access
- [ ] Revoke credentials on disconnect
- [ ] Never commit OAuth credentials
- [ ] Never expose tokens in error messages

---

# 25. Tests

### Google mapper

Test:

```text
Lich → Google
Google → Lich
```

including:

- [ ] title
- [ ] description
- [ ] location
- [ ] timed events
- [ ] all-day events
- [ ] timezone
- [ ] recurrence

### OAuth

- [ ] Connect
- [ ] Callback
- [ ] Token storage
- [ ] Token refresh
- [ ] Disconnect
- [ ] Invalid credentials

### Synchronization

- [ ] Create → Google
- [ ] Update → Google
- [ ] Delete → Google
- [ ] Google create → Lich
- [ ] Google update → Lich
- [ ] Google delete → Lich
- [ ] Incremental sync
- [ ] Duplicate prevention
- [ ] Retry after failure
- [ ] Rate limiting
- [ ] Conflict detection

### Conflict tests

```text
local only changed
remote only changed
both changed
delete locally / update remotely
update locally / delete remotely
```

---

# 26. Mock Google API

Don't make normal tests depend on real Google.

Create a fake provider:

```ts
interface CalendarProvider {
  ...
}
```

Then:

```text
FakeGoogleProvider
```

can simulate:

```text
create
update
delete
rate limit
network error
conflict
```

Use real Google API tests separately as a small integration suite.

---

# 27. Deliverables

### Server

- [ ] Google OAuth integration
- [ ] Secure credential storage
- [ ] Integration model
- [ ] Calendar mapping
- [ ] Event mapping
- [ ] External event IDs
- [ ] Google provider adapter
- [ ] Incremental sync
- [ ] Sync tokens/cursors
- [ ] Conflict detection
- [ ] Retry/backoff
- [ ] Integration status API

### TUI

- [ ] `lich google connect`
- [ ] `lich google disconnect`
- [ ] `lich google status`
- [ ] `lich google calendars`
- [ ] `lich google sync`
- [ ] Google sync status in TUI
- [ ] Conflict display
- [ ] Non-blocking synchronization

### Documentation

- [ ] Google Cloud setup
- [ ] OAuth configuration
- [ ] Required Google scopes
- [ ] Calendar mapping
- [ ] Sync behavior
- [ ] Conflict resolution
- [ ] Troubleshooting

---

# 28. Definition of Done

A complete Phase 3 flow:

```text
1. lich google connect
          ↓
2. Google OAuth
          ↓
3. Select Google calendar
          ↓
4. Map → Lich calendar
          ↓
5. Initial sync
          ↓
6. Google events appear in Lich
          ↓
7. lich add "Dinner tomorrow 7pm"
          ↓
8. Event immediately appears locally
          ↓
9. Background sync
          ↓
10. Event appears in Google
```

And the reverse:

```text
Google Calendar
      ↓
event created
      ↓
lich sync
      ↓
Lich server
      ↓
lich-tui local cache
      ↓
event appears in TUI
```

Most importantly:

```text
Google DOWN
    ↓
Lich still works

Lich server DOWN
    ↓
Local TUI still works

Internet DOWN
    ↓
Local mutations continue
    ↓
sync later
```

### Phase 3 boundary

**Build Google as a provider, not as a special case.**

The end state should be:

```text
                    Lich
                      │
                 Sync Engine
                      │
          ┌───────────┼───────────┐
          │           │           │
       Lich API     Google     Future...
```

That architecture leaves Phase 4 free to add **Gotify, webhooks, reminders, and automation** without having to redesign the calendar core.
