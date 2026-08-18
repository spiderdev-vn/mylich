# Lich — Development Plan

## 1. Vision

**Lich** is a self-hosted personal calendar system.

```text
lich-cli
  ├── CLI
  ├── TUI
  ├── local cache
  └── Google Calendar sync
          │
          ▼
lich-server
  ├── calendar storage
  ├── authentication
  ├── reminders
  ├── notifications
  └── integrations
```

**Core principle:**

> Lich is the source of truth. Google Calendar is an integration, not the database.

---

# Phase 1 — Foundation

### `lich-server`

Build the basic API and storage.

```text
User
Calendar
Event
```

Event:

```text
id
calendar_id
title
description
start_at
end_at
timezone
location
created_at
updated_at
deleted_at
```

API:

```text
POST   /events
GET    /events
GET    /events/:id
PATCH  /events/:id
DELETE /events/:id

GET    /calendars
POST   /calendars
PATCH  /calendars/:id
DELETE /calendars/:id
```

Start with **SQLite** for development/self-hosting, with PostgreSQL compatibility planned if needed.

---

# Phase 2 — `lich-cli`

Go CLI:

```bash
lich add "Dinner tomorrow 7pm"
lich today
lich week
lich month

lich list
lich search "dinner"

lich edit <id>
lich delete <id>
```

Architecture:

```text
CLI
 ↓
Application layer
 ↓
Lich API client
 ↓
Server
```

Important: CLI commands should **never wait unnecessarily for external services**.

---

# Phase 3 — TUI

Built with Charm:

- Bubble Tea
- Bubbles
- Lip Gloss

```bash
lich
```

Calendar views:

```text
Month
Week
Day
Agenda
```

Interactions:

```text
n → new event
e → edit
d → delete
/ → search
← → previous
→ → next
q → quit
```

The TUI should operate from local cached data and update asynchronously.

---

# Phase 4 — Local-first synchronization

This is one of the most important pieces.

```text
             Local SQLite
                  │
          ┌───────┴───────┐
          │               │
        Read            Write
          │               │
        instant       local first
                          │
                     sync queue
                          │
                          ▼
                    Lich Server
```

CLI:

```bash
lich add "Dinner tomorrow 7pm"
```

should behave like:

```text
1. Parse
2. Save locally
3. Display success
4. Queue synchronization
5. Exit
```

Not:

```text
1. Call server
2. Wait
3. Call Google
4. Wait
5. Finally show result
```

---

# Phase 5 — Authentication

```bash
lich login
```

Flow:

```text
lich CLI
   ↓
open browser
   ↓
your website
   ↓
authenticate
   ↓
device authorization
   ↓
CLI receives token
```

Server owns authentication.

CLI only manages credentials/tokens.

Commands:

```bash
lich login
lich logout
lich whoami
lich auth status
```

---

# Phase 6 — Google Calendar integration

Initially:

```bash
lich sync google
```

Support:

- Pull Google events
- Push Lich events
- Update events
- Delete events
- Calendar mapping
- Timezones
- Recurring events

Maintain external IDs:

```text
event
├── id
├── ...
└── integrations
      └── google
           └── external_event_id
```

Don't pollute the core event model with Google-specific fields.

---

# Phase 7 — Notifications

Server-side event system:

```text
Event created
Event updated
Event deleted
Reminder triggered
```

Dispatcher:

```text
Event
  ↓
Event Bus
  ↓
Notification / Integration handlers
```

First integration:

```text
Gotify
```

Then:

```text
Webhook
```

Example:

```yaml
notifications:
  gotify:
    enabled: true
    url: ...

webhooks:
  - url: https://example.com/calendar
    events:
      - event.created
      - event.updated
      - event.deleted
```

---

# Phase 8 — Sync Engine

Eventually make synchronization a proper subsystem.

```text
Sync Engine
├── Lich ↔ Local
├── Lich ↔ Google
└── future integrations
```

State:

```text
SYNCED
PENDING_CREATE
PENDING_UPDATE
PENDING_DELETE
FAILED
CONFLICT
```

Persistent queue:

```text
sync_jobs
├── id
├── provider
├── event_id
├── operation
├── attempts
├── next_retry_at
└── error
```

Use exponential backoff.

---

# Phase 9 — Self-hosting

Server should be easy to deploy:

```yaml
services:
  lich-server:
    image: ...
    environment:
      DATABASE_URL: ...
```

Support:

```text
Docker
SQLite
PostgreSQL
reverse proxy
HTTPS
```

Configuration:

```text
LICH_DATABASE_URL
LICH_BASE_URL
LICH_AUTH_*
LICH_GOTIFY_*
LICH_WEBHOOK_*
```

---

# Phase 10 — CLI polish

Make the CLI feel like a real Unix-style tool.

```bash
lich add "Meeting tomorrow 10am"
lich today
lich week
lich search meeting

lich calendar list
lich calendar use work

lich sync
lich sync google

lich doctor
lich version
```

Machine-readable output:

```bash
lich today --json
lich add "Meeting" --json
```

Useful for scripts and automation.

---

# Suggested repository structure

```text
lich/
│
├── lich-cli/
│   ├── cmd/
│   ├── internal/
│   │   ├── tui/
│   │   ├── cli/
│   │   ├── sync/
│   │   ├── cache/
│   │   └── api/
│   └── main.go
│
├── lich-server/
│   ├── cmd/
│   ├── internal/
│   │   ├── auth/
│   │   ├── calendar/
│   │   ├── events/
│   │   ├── sync/
│   │   ├── notification/
│   │   └── webhook/
│   └── main.go
│
└── lich-go/
    └── shared API/domain types
```

## MVP boundary

Don't build all of this initially.

**MVP:**

```text
lich-server
├── auth
├── calendars
└── events

lich-cli
├── add
├── list
├── today
├── week
├── delete
└── TUI

        ↓

local cache + async sync

        ↓

Google Calendar
```

Then add **Gotify → Webhooks → recurring events → richer sync**.

The fundamental architecture should be **local-first, server-backed, integration-agnostic** from day one.
