# Phase 1 — Foundation

Goal: build the **minimum complete Lich system** where `lich-server` stores calendars/events and `lich-tui` can authenticate, read, and mutate them.

Do **not** implement Google sync, Gotify, webhooks, reminders, or advanced offline sync yet.

---

## 1. Repository structure

```text
.
├── AGENTS.md
├── lich-server/
│   ├── package.json
│   ├── tsconfig.json
│   ├── .env.example
│   ├── src/
│   │   ├── app.ts
│   │   ├── server.ts
│   │   │
│   │   ├── config/
│   │   │   └── config.ts
│   │   │
│   │   ├── db/
│   │   │   ├── database.ts
│   │   │   ├── migrations/
│   │   │   └── repositories/
│   │   │
│   │   ├── auth/
│   │   ├── calendars/
│   │   └── events/
│   │
│   └── test/
│
└── lich-tui/
    ├── go.mod
    ├── go.sum
    ├── cmd/
    │   └── lich/
    │       └── main.go
    └── internal/
        ├── api/
        ├── auth/
        ├── cache/
        ├── cli/
        └── tui/
```

Keep the two applications independent.

---

# 2. Server bootstrap

Create a minimal Fastify application.

```text
lich-server
    ↓
config
    ↓
database
    ↓
Fastify
    ↓
routes
```

Environment:

```env
HOST=127.0.0.1
PORT=3000
DATABASE_PATH=./data/lich.db
```

The server should expose:

```text
GET /health
```

Response:

```json
{
  "status": "ok"
}
```

This becomes the first thing the TUI can use to test connectivity.

---

# 3. Database

Use SQLite.

For Node.js, pick **one SQLite implementation** and isolate it behind your database layer. The rest of the application should not care which SQLite library is being used.

Initial schema:

```text
users
calendars
events
```

### users

```text
id
email / username
created_at
updated_at
```

### calendars

```text
id
user_id
name
description
timezone
created_at
updated_at
```

### events

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

Use UUIDs or another stable application-generated ID.

Don't use Google IDs here.

---

# 4. Database migrations

Do not create tables directly during application startup with random `CREATE TABLE IF NOT EXISTS` statements scattered throughout the code.

Have a migration system:

```text
migrations/
├── 001_initial.sql
└── ...
```

Conceptually:

```text
server starts
    ↓
check migration version
    ↓
apply missing migrations
    ↓
start API
```

This matters because Lich is intended to be self-hosted.

A user should be able to upgrade:

```text
lich-server v0.1
        ↓
lich-server v0.2
        ↓
database migration
        ↓
existing data preserved
```

---

# 5. Repository layer

Keep database access separate from business logic.

For example:

```ts
interface EventRepository {
  create(event: CreateEvent): Promise<Event>;
  findById(id: string): Promise<Event | null>;
  findByCalendar(calendarId: string, range: DateRange): Promise<Event[]>;
  update(id: string, input: UpdateEvent): Promise<Event>;
  delete(id: string): Promise<void>;
}
```

The route handler shouldn't contain SQL.

Bad:

```text
HTTP handler
 ├── validate
 ├── SQL
 ├── business logic
 ├── notification
 └── response
```

Prefer:

```text
HTTP
 ↓
Route
 ↓
Service
 ↓
Repository
 ↓
SQLite
```

---

# 6. Calendar API

Implement the basic lifecycle.

### Create

```http
POST /calendars
```

```json
{
  "name": "Personal",
  "timezone": "Asia/Ho_Chi_Minh"
}
```

### List

```http
GET /calendars
```

### Get

```http
GET /calendars/:id
```

### Update

```http
PATCH /calendars/:id
```

### Delete

```http
DELETE /calendars/:id
```

For Phase 1, you can automatically create:

```text
Personal
```

for a newly created user.

---

# 7. Event API

Implement:

```text
POST   /events
GET    /events
GET    /events/:id
PATCH  /events/:id
DELETE /events/:id
```

Important query:

```http
GET /events?from=...&to=...
```

The TUI will heavily depend on this.

Example:

```http
GET /events?calendar=personal&from=2026-08-18T00:00:00+07:00&to=2026-08-19T00:00:00+07:00
```

The server should return events ordered by:

```text
start_at ASC
```

---

# 8. Event validation

At minimum:

```text
title              required
calendar_id        required
start_at           required
end_at             required
end_at > start_at
timezone           valid timezone
```

Reject:

```text
end_at <= start_at
```

Keep validation close to the domain/service layer so it isn't dependent on HTTP.

---

# 9. Authentication

Phase 1 only needs enough authentication to establish:

```text
User
  ↓
Calendar
  ↓
Events
```

Don't build a giant auth system yet.

A reasonable initial flow:

```text
lich-tui
   ↓
POST /auth/login
   ↓
server
   ↓
access token
```

The API then receives:

```http
Authorization: Bearer <token>
```

Every calendar/event request is scoped to the authenticated user.

Critical rule:

```text
User A
  ↓
GET /events/:id
```

must never be able to access User B's event simply by knowing the ID.

---

# 10. TUI API client

Create a small Go API client:

```go
type Client struct {
    BaseURL string
    Token   string
    HTTP    *http.Client
}
```

Methods:

```go
Health()
Login()
ListCalendars()

ListEvents()
GetEvent()
CreateEvent()
UpdateEvent()
DeleteEvent()
```

The TUI shouldn't construct HTTP requests everywhere.

Instead:

```text
TUI
 ↓
API client
 ↓
HTTP
 ↓
lich-server
```

---

# 11. CLI commands

Phase 1 only:

```bash
lich login
lich today
lich week
lich add "..."
lich delete <id>
lich
```

Don't implement the entire final command set yet.

### `lich login`

Authenticate and persist the token locally.

Use platform-specific config directories.

On Windows, use:

```go
os.UserConfigDir()
```

rather than hardcoding:

```text
C:\Users\...
```

---

# 12. `lich today`

Flow:

```text
lich today
    ↓
calculate today's boundaries
    ↓
GET /events?from=...&to=...
    ↓
render
```

Example:

```text
Tuesday, August 18

10:00  Team meeting
14:30  Dentist
19:00  Dinner
```

No TUI required yet.

This gives you a working vertical slice very early.

---

# 13. `lich add`

Support the first explicit syntax:

```bash
lich add "Dinner" --at 19:00
```

and:

```bash
lich add "Meeting" \
  --date 2026-08-20 \
  --at 09:00 \
  --duration 1h
```

Do **not** build sophisticated natural-language parsing yet.

First get:

```text
CLI
 ↓
parse
 ↓
API
 ↓
SQLite
 ↓
response
```

working reliably.

Natural language:

```bash
lich add "Dinner tomorrow at 7pm"
```

can come later.

---

# 14. Basic TUI

Only after the API and CLI work.

Start with:

```bash
lich
```

and render:

```text
August 2026

Mon Tue Wed Thu Fri Sat Sun
                  1   2   3
 4   5   6   7   8   9  10
11  12  13  14  15  16  17
18  19  20  21  22  23  24
25  26  27  28  29  30  31
```

Then add event indicators.

Don't build the complete Google-like calendar UI yet.

---

# 15. First vertical slice

This should be the **Phase 1 milestone**:

```text
1. Start server

   npm run dev

2. Login

   lich login

3. Add event

   lich add "Dinner" --date 2026-08-18 --at 19:00 --duration 1h

4. Query

   lich today

5. Open TUI

   lich

6. See the same event

7. Delete

   lich delete <id>

8. Event disappears from both CLI and TUI
```

At this point you have an actual working Lich system.

---

# 16. Testing

### Server unit tests

Test:

```text
CalendarService
EventService
validation
repositories
authentication
authorization
```

### API integration tests

Test:

```text
POST /events
GET /events
PATCH /events/:id
DELETE /events/:id
```

Especially:

```text
User A cannot access User B's data
```

### TUI tests

Initially focus on:

```text
command parsing
date calculations
API client
```

Don't over-test visual rendering.

---

# 17. Definition of Done

Phase 1 is complete when:

- [ ] `lich-server` starts with Fastify
- [ ] SQLite database is automatically migrated
- [ ] Users can authenticate
- [ ] Users have calendars
- [ ] Events can be created/read/updated/deleted
- [ ] Events are timezone-aware
- [ ] API is authenticated
- [ ] TUI can communicate with the server
- [ ] `lich login` works
- [ ] `lich add` works
- [ ] `lich today` works
- [ ] `lich week` works
- [ ] `lich delete` works
- [ ] `lich` opens a basic calendar TUI
- [ ] Server tests cover the core event lifecycle
- [ ] No Google/Gotify/Webhook dependency exists yet

### Phase 1 architecture

```text
                 ┌──────────────┐
                 │  lich-tui    │
                 │              │
                 │ CLI          │
                 │ Bubble Tea   │
                 └──────┬───────┘
                        │
                       HTTP
                        │
                 ┌──────▼───────┐
                 │ lich-server  │
                 │              │
                 │ Fastify      │
                 │ Auth         │
                 │ Calendar     │
                 │ Events       │
                 └──────┬───────┘
                        │
                   ┌────▼────┐
                   │ SQLite  │
                   └─────────┘
```

**Phase 1 should intentionally be boring.** The goal is to establish a clean calendar domain and API that Phase 2 can safely build synchronization and integrations on top of.

---

Deliverables

For Phase 1, I’d make **tests and deliverables explicit acceptance criteria**, so you can implement it incrementally and know exactly when the phase is actually finished.

## Phase 1 — Deliverables

### D1 — Server foundation

- [ ] Node.js + TypeScript project
- [ ] Fastify server
- [ ] Environment/config handling
- [ ] SQLite database
- [ ] Database migration system
- [ ] `GET /health`
- [ ] Graceful shutdown
- [ ] Structured error responses

**Acceptance:**

```bash
npm run dev
curl http://localhost:3000/health
```

returns:

```json
{
  "status": "ok"
}
```

---

### D2 — Authentication

- [ ] User model
- [ ] Login endpoint
- [ ] Token generation
- [ ] Authentication middleware
- [ ] Protected API routes
- [ ] User isolation

**Tests:**

- [ ] Valid login succeeds
- [ ] Invalid credentials fail
- [ ] Missing token is rejected
- [ ] Invalid token is rejected
- [ ] User A cannot access User B's calendar
- [ ] User A cannot access User B's events

---

### D3 — Calendar management

API:

```text
POST   /calendars
GET    /calendars
GET    /calendars/:id
PATCH  /calendars/:id
DELETE /calendars/:id
```

- [ ] Calendar CRUD
- [ ] Calendar belongs to user
- [ ] Calendar timezone
- [ ] Validation
- [ ] Authorization

**Tests:**

- [ ] Create calendar
- [ ] List calendars
- [ ] Update calendar
- [ ] Delete calendar
- [ ] Cannot access another user's calendar
- [ ] Invalid timezone rejected

---

### D4 — Event management

API:

```text
POST   /events
GET    /events
GET    /events/:id
PATCH  /events/:id
DELETE /events/:id
```

Support:

```text
title
description
start_at
end_at
timezone
location
calendar_id
```

**Tests:**

- [ ] Create event
- [ ] Read event
- [ ] Update event
- [ ] Delete event
- [ ] List events
- [ ] Filter events by date range
- [ ] Events ordered by start time
- [ ] Invalid date rejected
- [ ] `end_at <= start_at` rejected
- [ ] Invalid calendar rejected
- [ ] Cross-user access rejected

---

### D5 — Go API client

`lich-tui` gets a reusable API client:

```go
type Client interface {
    Login(...)
    ListCalendars(...)
    ListEvents(...)
    CreateEvent(...)
    UpdateEvent(...)
    DeleteEvent(...)
}
```

**Tests:**

- [ ] Request serialization
- [ ] Response deserialization
- [ ] HTTP error handling
- [ ] Authentication handling
- [ ] Network failure handling
- [ ] Malformed response handling

Use a fake HTTP server rather than the real server for most unit tests.

---

### D6 — CLI

Initial commands:

```bash
lich login
lich today
lich week
lich add ...
lich delete ...
```

**Tests:**

- [ ] Commands parse correctly
- [ ] Required arguments are validated
- [ ] Invalid arguments return useful errors
- [ ] Correct API methods are called
- [ ] CLI returns appropriate exit codes

---

### D7 — Basic TUI

```bash
lich
```

Initial functionality:

- [ ] Month view
- [ ] Event indicators
- [ ] Navigate previous/next month
- [ ] Select a day
- [ ] Display day's events
- [ ] Quit cleanly
- [ ] Loading/error states

Don't make the first TUI test suite screenshot-based.

Test the Bubble Tea model/state transitions instead.

Example:

```text
Key → Model.Update() → expected state
```

---

# Test Layers

I'd explicitly maintain four test levels.

```text
                 ┌───────────────┐
                 │  E2E Tests    │
                 └───────┬───────┘
                         │
                 ┌───────▼───────┐
                 │ API Tests     │
                 └───────┬───────┘
                         │
              ┌──────────▼──────────┐
              │ Integration Tests   │
              └──────────┬──────────┘
                         │
              ┌──────────▼──────────┐
              │    Unit Tests       │
              └─────────────────────┘
```

### Unit

Fast, isolated:

```text
EventService
CalendarService
validation
date handling
CLI parsing
TUI state
```

### Integration

Real SQLite:

```text
repository → SQLite
service → repository
```

Use a temporary database.

### API

Start Fastify against a test database and test:

```text
HTTP → route → service → DB
```

### E2E

Only a few.

```text
start server
    ↓
login
    ↓
create calendar
    ↓
create event
    ↓
query event
    ↓
update event
    ↓
delete event
```

This validates the whole system without creating a giant slow test suite.

---

# Important Edge Cases

Calendar software has nasty edge cases, so Phase 1 should explicitly test:

### Timezone

```text
Asia/Ho_Chi_Minh
UTC
America/New_York
```

### Date boundaries

```text
00:00
23:59:59
midnight crossing
```

### Range queries

```text
from == event.start
to == event.end
event overlaps range
event completely surrounds range
```

### Invalid events

```text
start == end
end < start
missing timezone
invalid calendar
```

### Authorization

```text
User A → User A event ✓
User A → User B event ✗
```

---

# Developer Deliverables

The phase should also produce these non-code artifacts:

- [ ] `README.md`
- [ ] `.env.example`
- [ ] API documentation
- [ ] Database migration documentation
- [ ] Local development instructions
- [ ] Test instructions
- [ ] Example CLI usage
- [ ] Architecture diagram
- [ ] Initial API contract

For example:

```bash
# Server
cd lich-server
npm install
npm test
npm run dev

# TUI
cd lich-tui
go test ./...
go run ./cmd/lich
```

---

# Phase 1 Release

I'd define the release as:

```text
Lich v0.1.0 — Foundation
```

### Release acceptance test

A fresh machine should be able to:

```text
1. Clone repository
2. Start lich-server
3. Create/login user
4. Run lich-tui
5. Login
6. Create calendar
7. Create event
8. View event
9. Edit event
10. Delete event
```

with **no Google account, Gotify, webhook, or external integration required**.

That gives Phase 1 a very clear boundary: **a complete, authenticated personal calendar backend + functional Go client, with enough test coverage that Phase 2 can safely introduce synchronization.**
