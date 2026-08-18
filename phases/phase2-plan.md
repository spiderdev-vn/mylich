# Phase 2 — Local-first CLI + TUI

**Goal:** make `lich-tui` feel like a real daily-use application: instant interaction, local cache, offline reads, and asynchronous synchronization with `lich-server`.

Phase 2 should **not** introduce Google Calendar yet. Google becomes Phase 3.

---

## 1. Architecture

Phase 1:

```text
lich-tui ──HTTP──> lich-server ──> SQLite
```

Phase 2:

```text
                    ┌────────────────────┐
                    │     lich-tui       │
                    │                    │
                    │ TUI / CLI          │
                    │ Local SQLite       │
                    │ Sync Engine        │
                    └─────────┬──────────┘
                              │
                         background
                              │
                              ▼
                    ┌────────────────────┐
                    │   lich-server      │
                    │                    │
                    │ Source of truth    │
                    └─────────┬──────────┘
                              │
                           SQLite
```

The important change:

> **The TUI stops treating the server as part of the rendering path.**

---

# 2. Local database

Add SQLite to `lich-tui`.

Suggested structure:

```text id="4e1x7v"
lich-tui/
└── internal/
    └── cache/
        ├── database.go
        ├── migrations/
        ├── events.go
        ├── calendars.go
        └── sync.go
```

Local tables:

```text id="x5w3w6"
calendars
events
sync_jobs
sync_metadata
```

You don't need to mirror every server table.

The local DB is a **client cache + offline state**, not another independent calendar backend.

---

# 3. Local event model

Store enough information to render the calendar without contacting the server.

```text id="kj2d7m"
events
├── id
├── calendar_id
├── title
├── description
├── start_at
├── end_at
├── timezone
├── location
├── updated_at
└── sync_state
```

`sync_state`:

```text id="6y2q93"
synced
pending_create
pending_update
pending_delete
failed
```

---

# 4. Sync queue

This is the most important component of Phase 2.

```text id="qg4nvl"
sync_jobs
├── id
├── entity_type
├── entity_id
├── operation
├── created_at
├── attempts
├── next_attempt_at
└── last_error
```

Operations:

```text id="m20mqa"
CREATE
UPDATE
DELETE
```

Example:

```text id="9r7p3z"
lich add "Dinner"
        │
        ▼
local event created
        │
        ▼
sync_jobs
        │
        ▼
background worker
        │
        ▼
POST /events
```

---

# 5. Local-first mutation

`lich add` becomes:

```text id="8t2dqa"
Input
  ↓
Validate
  ↓
Create local event
  ↓
Create sync job
  ↓
Render success
  ↓
Return exit 0
```

Example:

```bash id="gux8po"
lich add "Dinner tomorrow 7pm"
```

Immediately:

```text id="q3a1xv"
✓ Added "Dinner"
  Tomorrow · 19:00
  Sync: pending
```

The command should **not wait for the server**.

---

# 6. Background synchronization

Implement a Go sync worker.

Conceptually:

```text id="k2cv8v"
SyncManager
    │
    ├── read pending jobs
    │
    ├── execute
    │
    ├── success → mark synced
    │
    └── failure → retry later
```

The worker should be able to run independently from the TUI.

Potential implementation:

```text id="q7tvkz"
lich command
   ↓
local DB
   ↓
spawn / notify sync process
   ↓
exit
```

Later this can become:

```bash id="w80s5d"
lich daemon
```

Do not require a permanent daemon yet.

---

# 7. Retry strategy

Network failures are normal.

Example:

```text id="j2t9mw"
Attempt 1 → fail
     ↓
2 seconds
     ↓
Attempt 2 → fail
     ↓
5 seconds
     ↓
Attempt 3 → fail
     ↓
30 seconds
     ↓
...
```

Use exponential backoff with a maximum delay.

Persist:

```text id="l8x2qf"
attempts
next_attempt_at
last_error
```

So restarting `lich` doesn't lose the queue.

---

# 8. Read path

For:

```bash id="9l8lcy"
lich today
```

don't do:

```text id="3l1vzw"
HTTP → server → response → render
```

Instead:

```text id="z5n5m6"
SQLite → render
```

Then optionally trigger background synchronization.

This gives:

```text id="3u6g1e"
$ lich today

Today · August 18

10:00  Team meeting
14:00  Dentist
19:00  Dinner

● Syncing...
```

---

# 9. Cache synchronization

The local cache still needs server changes.

Example:

```text id="8tdc3j"
Server
  ↓
GET /events?updated_since=...
  ↓
TUI
  ↓
update local cache
```

Add a server API such as:

```http id="2h7vax"
GET /events?updated_after=<timestamp>
```

or use a cursor:

```http id="t5j3k0"
GET /sync?cursor=<cursor>
```

I would prefer a **cursor-based sync endpoint** eventually.

Example response:

```json id="x4s5w6"
{
  "cursor": "abc123",
  "changes": [
    {
      "type": "event.updated",
      "event": {}
    }
  ]
}
```

This will scale better when integrations are added later.

---

# 10. Conflict handling

Phase 2 should establish the mechanism, even if conflict resolution is simple.

Example:

```text id="xq9k4n"
Local event
Dinner 19:00

Server event
Dinner 20:00
```

Don't silently overwrite.

Add:

```text id="m9c4fk"
conflict
```

state.

Initially, you can use:

> server wins

or:

> latest `updated_at` wins

But document the policy clearly.

More sophisticated conflict resolution can come later.

---

# 11. Offline mode

The application should remain usable without internet.

```text id="g9l1uj"
Network OFFLINE

lich today     ✓
lich add ...   ✓
lich edit ...  ✓
lich delete .. ✓

lich sync      pending
```

TUI indicator:

```text id="8m3qf1"
Lich · Offline
```

When connectivity returns:

```text id="6f7g5c"
Lich · Syncing
       ↓
Lich · Synced
```

---

# 12. TUI architecture

Now make the TUI properly asynchronous.

Bubble Tea model:

```go id="qz6t1e"
type Model struct {
    calendar CalendarState
    events   []Event

    syncState SyncState
    loading   bool
    error     error
}
```

Commands:

```text id="k3c1wb"
tea.Cmd
   │
   ├── loadEvents
   ├── createEvent
   ├── updateEvent
   ├── deleteEvent
   └── sync
```

Messages:

```text id="d4y7fv"
EventsLoadedMsg
EventCreatedMsg
EventUpdatedMsg
EventDeletedMsg
SyncStartedMsg
SyncCompletedMsg
SyncFailedMsg
```

This prevents network work from blocking `Update()` or `View()`.

---

# 13. TUI calendar

Expand the basic Phase 1 TUI.

Views:

```text id="g1w5dx"
Month
Week
Day
Agenda
```

Navigation:

```text id="m5j4vn"
h / ←   previous
l / →   next
j / ↓   next item
k / ↑   previous item
t       today
```

Actions:

```text id="8f5g9c"
n       new
e       edit
d       delete
/       search
r       refresh
q       quit
```

Keyboard mappings should be configurable eventually, but don't build configuration yet.

---

# 14. Event editor

Create a proper TUI form.

```text id="0p4f7b"
┌─ New Event ────────────────────────────┐
│ Title:       Dinner                    │
│ Date:        2026-08-18                │
│ Start:       19:00                     │
│ End:         20:00                     │
│ Calendar:    Personal                  │
│ Location:                               │
│ Description:                            │
│                                         │
│              [ Create ]                 │
└─────────────────────────────────────────┘
```

Use Bubbles components where appropriate.

The form should update local state immediately after submission.

---

# 15. CLI improvements

Phase 2 expands:

```bash id="n9b0h8"
lich list
lich today
lich tomorrow
lich week
lich month

lich add ...
lich edit <id>
lich delete <id>

lich search <query>

lich sync
lich status
```

`lich status` is particularly useful:

```text id="z4v8kp"
Server     ✓ connected
Cache      ✓ healthy
Sync       ✓ synced
Pending    2
Failed     0
```

---

# 16. Server changes

The server needs to support efficient synchronization.

Add:

```text id="j4n8pq"
GET /sync
```

or equivalent.

Also add:

```text id="k1x6nq"
updated_at
deleted_at
```

to resources where necessary.

**Soft deletion is important.**

If:

```text id="7a6v4c"
Client A deletes event
```

the server cannot immediately remove the record if another client needs to discover that deletion.

Instead:

```text id="l5g3nq"
deleted_at = 2026-08-18T...
```

Then synchronization can propagate the deletion.

---

# 17. Sync lifecycle

A complete example:

```text id="2f7n5a"
lich add "Dinner"
        │
        ▼
SQLite
        │
        ├── event
        └── sync_job
                │
                ▼
          Sync Worker
                │
                ▼
        POST /events
                │
          ┌─────┴─────┐
        success      failure
          │             │
          ▼             ▼
       synced        retry
```

For server-side changes:

```text id="r2v9wm"
lich sync
   ↓
GET /sync?cursor=...
   ↓
apply changes locally
   ↓
update cursor
```

---

# 18. Testing

### Local database

- [ ] Create event locally
- [ ] Update event locally
- [ ] Delete event locally
- [ ] Persist after restart
- [ ] Pending jobs persist
- [ ] Failed jobs persist

### Sync worker

- [ ] Successful create
- [ ] Successful update
- [ ] Successful delete
- [ ] Network failure
- [ ] Server 500
- [ ] Retry behavior
- [ ] Backoff behavior
- [ ] Worker restart
- [ ] Duplicate execution safety

### Offline behavior

- [ ] Create event offline
- [ ] Edit event offline
- [ ] Delete event offline
- [ ] Reconnect
- [ ] All operations eventually synchronize

### TUI

- [ ] Loading does not block UI
- [ ] Event creation updates UI immediately
- [ ] Sync status updates correctly
- [ ] Network failure doesn't crash TUI
- [ ] Navigation works while synchronization is running

### E2E

```text id="n5h6t0"
Start server
   ↓
Login
   ↓
Start TUI
   ↓
Create event
   ↓
Verify local cache
   ↓
Verify server
   ↓
Stop network
   ↓
Create event
   ↓
Restore network
   ↓
Verify synchronization
```

---

# 19. Phase 2 Deliverables

### `lich-server`

- [ ] Sync endpoint
- [ ] Incremental change tracking
- [ ] `updated_at`
- [ ] `deleted_at`
- [ ] Cursor-based synchronization
- [ ] Tests for synchronization API

### `lich-tui`

- [ ] Local SQLite cache
- [ ] Persistent sync queue
- [ ] Background sync worker
- [ ] Retry/backoff
- [ ] Offline operation
- [ ] `lich status`
- [ ] Improved CLI
- [ ] Month/week/day/agenda TUI
- [ ] Event editor
- [ ] Async TUI operations
- [ ] Sync indicators

### Documentation

- [ ] Local cache design
- [ ] Sync protocol
- [ ] Conflict policy
- [ ] Offline behavior
- [ ] CLI reference
- [ ] TUI keybindings

---

# 20. Definition of Done

Phase 2 is complete when this works:

```text
Internet ON
    │
    ▼
lich add "Dinner"
    │
    ├── instant local update
    └── background sync
             │
             ▼
        server updated
```

Then:

```text
Internet OFF
    │
    ▼
lich add "Dinner"
lich edit ...
lich delete ...
lich today
    │
    ▼
Everything still works locally
```

Then:

```text
Internet ON
    │
    ▼
Sync automatically
    │
    ▼
Server eventually matches local state
```

The key acceptance criterion is:

> **A slow or completely unavailable network must not make `lich` feel slow.**

That foundation makes Phase 3 — **Google Calendar integration** — substantially cleaner because Google becomes another synchronization provider rather than something the entire application depends on.
