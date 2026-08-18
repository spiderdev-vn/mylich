## Project Overview

This repository contains two independent parts of the Lich project:

```text
.
├── lich-server/
└── lich-tui/
```

### `lich-server/`

A self-hosted calendar backend.

- Runtime: Node.js
- Framework: Fastify
- Database: SQLite
- Language: TypeScript
- Responsibility:
  - Store calendars and events
  - Authenticate clients
  - Expose the Lich API
  - Handle server-side integrations
  - Dispatch notifications and webhooks
  - Provide synchronization infrastructure

The server is the **source of truth for Lich data**.

Google Calendar is an integration, not the primary database.

### `lich-tui/`

A terminal client for Lich.

- Language: Go
- TUI framework: Charm / Bubble Tea ecosystem
- Responsibility:
  - Interactive calendar UI
  - CLI commands
  - Communicate with `lich-server`
  - Maintain a local cache
  - Provide Google Calendar synchronization where appropriate

The TUI should remain responsive and should never make the user wait unnecessarily for network operations.

---

## Architecture

```text
                    ┌─────────────────────┐
                    │      lich-tui       │
                    │                     │
                    │ Go + Bubble Tea     │
                    │ CLI + TUI           │
                    │ Local cache         │
                    └──────────┬──────────┘
                               │
                              API
                               │
                               ▼
                    ┌─────────────────────┐
                    │     lich-server     │
                    │                     │
                    │ Node.js + Fastify   │
                    │ SQLite              │
                    │ Auth                │
                    │ Events              │
                    │ Integrations        │
                    └──────────┬──────────┘
                               │
              ┌────────────────┼────────────────┐
              ▼                ▼                ▼
          Google Calendar    Gotify          Webhooks
```

### Core principles

1. **Lich owns the calendar data.**
2. External services such as Google Calendar are integrations.
3. Network operations must not block the interactive TUI unnecessarily.
4. Local state should be used whenever possible for responsive reads.
5. Synchronization must be resilient to slow or unavailable networks.
6. Integrations should be isolated from the core calendar domain.
7. The server and TUI must remain independently deployable and usable.

---

## `lich-server` Guidelines

### Technology

Use:

- Node.js
- TypeScript
- Fastify
- SQLite

Prefer small, focused modules over large service files.

Keep the core domain independent from Fastify-specific request/response objects where practical.

Recommended separation:

```text
lich-server/
├── src/
│   ├── app/
│   ├── auth/
│   ├── calendar/
│   ├── events/
│   ├── integrations/
│   ├── notifications/
│   ├── webhooks/
│   ├── db/
│   └── server.ts
```

The exact structure may evolve with the implementation.

### Database

SQLite is the default database.

Database access must be centralized. Do not scatter raw SQL throughout HTTP handlers.

Important entities will generally include:

- users
- calendars
- events
- integrations
- sync state
- notification configuration
- webhook configuration

Use migrations for schema changes.

Never modify an existing schema without considering existing installations and migration paths.

### API

The API should be versionable and predictable.

Prefer resource-oriented endpoints:

```text
GET    /events
POST   /events
GET    /events/:id
PATCH  /events/:id
DELETE /events/:id

GET    /calendars
POST   /calendars
GET    /calendars/:id
PATCH  /calendars/:id
DELETE /calendars/:id
```

Keep integration-specific behavior out of generic event endpoints when possible.

### Authentication

Authentication belongs to `lich-server`.

The TUI should not contain application-specific authentication logic.

Authentication flows should allow the TUI to authenticate through the user's browser when necessary.

Never hard-code credentials, tokens, secrets, or private keys.

---

## `lich-tui` Guidelines

### Technology

Use:

- Go
- Bubble Tea
- Bubbles
- Lip Gloss

Prefer idiomatic Go and keep platform-specific behavior isolated.

### CLI design

The main command is:

```text
lich <action> [input] [options]
```

Examples:

```bash
lich add "Dinner tomorrow 7pm"
lich today
lich week
lich month
lich search "dinner"

lich edit <event-id>
lich delete <event-id>

lich login
lich logout
lich sync
```

Running:

```bash
lich
```

opens the interactive TUI.

### Responsiveness

The TUI must not block on slow network requests.

Avoid:

```text
user action
    ↓
HTTP request
    ↓
wait
    ↓
render
```

Prefer:

```text
user action
    ↓
update local state
    ↓
render immediately
    ↓
background request
    ↓
update state when complete
```

For mutations, use a local-first approach where practical:

```text
User action
    ↓
Local state/cache
    ↓
Immediate UI feedback
    ↓
Sync queue
    ↓
Server
```

A temporary network failure should not make the TUI feel frozen.

### Local cache

The TUI may maintain a local SQLite cache.

The cache should support:

- Fast calendar rendering
- Offline reads
- Pending mutations
- Retryable synchronization
- Recovery after process termination

Important pending operations must not exist only in memory.

---

## Synchronization

Synchronization should be treated as a separate subsystem.

Do not tightly couple Google Calendar behavior to the core event model.

Conceptually:

```text
Lich Event
    │
    ├── Local state
    │
    ├── Lich Server
    │
    └── External integrations
          └── Google Calendar
```

Track external identifiers separately from the core event.

For example:

```text
event
└── integrations
    └── google
        └── externalEventId
```

Possible synchronization states:

```text
SYNCED
PENDING_CREATE
PENDING_UPDATE
PENDING_DELETE
FAILED
CONFLICT
```

Synchronization must tolerate:

- Offline operation
- Slow networks
- Server downtime
- Google API failures
- Process restarts
- Duplicate requests
- Partial synchronization

Use retries with backoff where appropriate.

---

## Notifications and Webhooks

Notifications are server-side concerns.

The server should be able to react to calendar events:

```text
event.created
event.updated
event.deleted
event.reminder
```

Integrations should consume these events without coupling them to the calendar domain.

Potential integrations include:

- Gotify
- Generic webhooks
- Email
- Future notification providers

A webhook should not make the main event mutation unnecessarily dependent on the webhook endpoint being available.

Prefer asynchronous dispatch.

---

## Time and Dates

Calendar applications require careful timezone handling.

Always preserve timezone information.

Do not treat calendar timestamps as plain strings without defined semantics.

Consider:

- User timezone
- Event timezone
- UTC conversion
- Daylight saving time
- Recurring events
- Google Calendar timezone behavior

Never silently discard timezone information.

---

## Windows Compatibility

`lich-tui` must work well on Windows.

Do not assume Unix-only behavior such as:

- `$HOME`
- XDG directories
- `fork()`
- Unix signals
- `/tmp`
- `/dev/null`

Use Go's platform-aware APIs such as:

```go
os.UserConfigDir()
os.UserCacheDir()
os.UserHomeDir()
```

Keep OS-specific functionality isolated behind platform-specific Go files where necessary.

The TUI should work correctly in modern Windows Terminal and should degrade gracefully in other supported terminals.

---

## Error Handling

Errors should be actionable and concise.

Distinguish between:

- Invalid user input
- Local database errors
- Authentication errors
- Server errors
- Network errors
- Integration errors
- Synchronization conflicts

A successful local mutation should not necessarily be considered a failed CLI command merely because an external synchronization operation is temporarily unavailable.

For example:

```text
✓ Event created
  Sync: pending
```

is preferable to reporting the entire operation as failed when the event was successfully saved locally.

---

## Testing

### Server

Test:

- API handlers
- Domain logic
- Database operations
- Authentication
- Event lifecycle
- Integration dispatch
- Webhook behavior
- Notification behavior

### TUI

Test:

- Command parsing
- Date/time parsing
- Local state transitions
- Sync queue behavior
- Error handling

Avoid making tests dependent on live Google Calendar APIs.

External integrations should be mocked or tested separately.

---

## Development Principles

- Keep `lich-server` and `lich-tui` loosely coupled through a stable API.
- Prefer simple implementations over premature abstractions.
- Do not introduce Google-specific concepts into the core calendar domain unless necessary.
- Do not block the TUI on network operations.
- Persist important synchronization state.
- Make offline behavior predictable.
- Keep authentication and secrets out of source control.
- Prefer explicit behavior over magic.
- Maintain backward compatibility for the API where practical.
- Add migrations for database schema changes.
- Keep dependencies minimal and justified.

## What Not To Do

Do not:

- Make Google Calendar the source of truth.
- Require Google Calendar for basic Lich functionality.
- Make every CLI command wait for network synchronization.
- Store important pending operations only in memory.
- Put server credentials in the TUI.
- Couple notification providers directly to event business logic.
- Add a dependency when the standard library or existing project dependency is sufficient.
- Break existing database installations without a migration.
- Assume the application only runs on Linux/macOS.

## Project Goal

Lich should feel like a **personal calendar system**, not merely a Google Calendar CLI.

The desired experience is:

```text
lich add "Dinner tomorrow 7pm"
        ↓
    instant feedback
        ↓
    local state
        ↓
  background sync
        ↓
lich-server
        ↓
 ┌──────┼─────────┐
Google  Gotify   Webhook
```

The user should be able to use Lich every day without caring which underlying service is currently available.
