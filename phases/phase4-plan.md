# Phase 4 — Local-first Synchronization

**Goal:** make synchronization a reliable, persistent subsystem rather than something tied to individual CLI commands.

By the end of Phase 4:

```text
                    ┌───────────────┐
                    │   lich-tui    │
                    │               │
                    │ TUI / CLI      │
                    │ Local SQLite   │
                    └───────┬───────┘
                            │
                     Sync Engine
                            │
                 ┌──────────┴──────────┐
                 │                     │
          lich-server             Google Calendar
```

The user should be able to work normally while **offline, during server downtime, or with slow Google connectivity**.

---

## 1. Core principles

### Local state is immediately usable

```text
User action
    ↓
Local DB
    ↓
UI updates immediately
    ↓
Sync happens later
```

Never:

```text
User action
    ↓
HTTP
    ↓
Google
    ↓
wait...
    ↓
UI
```

### Synchronization is eventually consistent

The goal is:

```text
Local state
    ↓
eventually
    ↓
Server state
    ↓
eventually
    ↓
External integrations
```

Temporary failures are expected.

---

# 2. Sync Engine

Create a dedicated subsystem:

```text id="7f6m5g"
lich-tui/internal/sync/

├── engine.go
├── queue.go
├── worker.go
├── retry.go
├── conflict.go
└── state.go
```

Responsibilities:

- Detect pending changes
- Queue operations
- Execute operations
- Retry failures
- Pull remote changes
- Detect conflicts
- Update local state
- Maintain sync metadata

The TUI should only consume sync state; it should not implement synchronization itself.

---

# 3. Local database

Expand the Phase 2 cache.

```text id="h8b3rf"
events
calendars

sync_jobs
sync_cursors
sync_conflicts
sync_metadata
```

### `sync_jobs`

```text id="9z8xwq"
id
provider
entity_type
entity_id
operation

status
attempts
next_attempt_at
last_error

created_at
updated_at
```

Possible status:

```text id="h7q9nz"
pending
processing
completed
failed
cancelled
```

---

# 4. Sync queue semantics

When the user creates an event:

```text id="75s8pv"
CREATE event
     ↓
write event locally
     ↓
enqueue CREATE
```

If they immediately edit it:

```text id="j8m6cd"
CREATE
  ↓
UPDATE
```

Don't blindly execute both.

The queue should eventually collapse them:

```text id="9wy2sh"
CREATE
UPDATE
UPDATE
UPDATE
```

into:

```text id="c2i4xk"
CREATE(final state)
```

This avoids unnecessary network operations.

---

# 5. Queue coalescing

Rules:

```text id="0p4y6q"
CREATE + UPDATE → CREATE
CREATE + DELETE → remove everything
UPDATE + UPDATE → latest UPDATE
UPDATE + DELETE → DELETE
```

Example:

```text id="t0e4d1"
lich add "Dinner"
lich edit dinner → 19:30
lich edit dinner → 20:00
```

Network should ideally see:

```text id="4b1f2e"
CREATE Dinner 20:00
```

not:

```text id="s2j5nf"
CREATE Dinner
UPDATE 19:30
UPDATE 20:00
```

---

# 6. Worker lifecycle

The worker should run independently from the TUI event loop.

```text id="9l8d4m"
Sync Worker
    │
    ├── fetch pending jobs
    │
    ├── lock/claim job
    │
    ├── execute
    │
    ├── success → complete
    │
    └── failure → retry
```

Use a bounded worker model.

For example:

```text id="1c5k6w"
worker count = 1
```

initially.

Calendar synchronization doesn't need massive concurrency, and a single worker makes ordering easier to reason about.

---

# 7. Persistent worker state

Never rely on:

```go
var queue []Job
```

for important operations.

The queue must survive:

```text id="w1d6rq"
TUI crash
computer restart
network failure
server restart
```

Example:

```text id="d6m7xq"
lich add "Dinner"
    ↓
SQLite
    ↓
process crashes
    ↓
lich starts again
    ↓
pending job still exists
    ↓
sync
```

---

# 8. Retry strategy

Transient failures should retry automatically.

Example:

```text id="c6b9pp"
attempt 1 → 1s
attempt 2 → 2s
attempt 3 → 4s
attempt 4 → 8s
...
```

With a maximum:

```text id="l7g0bz"
max delay = 5 minutes
```

Add jitter to avoid synchronized retries if multiple processes are running.

Permanent errors should not retry forever.

For example:

```text id="0k3f9v"
400 invalid event
401 authentication failure
403 permission denied
```

should transition to a failure state requiring user intervention.

---

# 9. Pull synchronization

Push alone isn't enough.

The client also needs:

```text id="1x4j8p"
remote changes
      ↓
local cache
```

Use a cursor:

```text id="f2z0ym"
sync_cursors
├── source
├── cursor
└── updated_at
```

Flow:

```text id="v2x5f8"
lich sync
    ↓
GET /sync?cursor=abc
    ↓
server changes
    ↓
apply locally
    ↓
receive new cursor
    ↓
persist cursor
```

---

# 10. Server synchronization endpoint

The server should expose a durable change feed.

For example:

```http id="n9u4pg"
GET /sync?cursor=<cursor>
```

Response:

```json id="v8d1br"
{
  "cursor": "cursor-123",
  "changes": [
    {
      "type": "event.created",
      "event": {}
    },
    {
      "type": "event.updated",
      "event": {}
    },
    {
      "type": "event.deleted",
      "id": "..."
    }
  ]
}
```

The cursor must be stable and recoverable.

---

# 11. Server-side change log

To support reliable synchronization, add a change log.

```text id="x8j5zq"
changes
├── sequence
├── entity_type
├── entity_id
├── operation
├── payload
└── created_at
```

Example:

```text id="p3y7fv"
1001 EVENT CREATE abc
1002 EVENT UPDATE abc
1003 EVENT DELETE xyz
```

A client can say:

```text id="x0s4jq"
"I have processed through 1000."
```

Server returns:

```text id="5p7c8m"
1001+
```

This is significantly safer than relying only on timestamps.

---

# 12. Cursor vs timestamp

Prefer:

```text id="v9j3xk"
cursor / sequence
```

over:

```text id="d2k5qh"
updated_at > last_sync
```

because timestamps introduce problems around:

- clock skew
- identical timestamps
- precision
- timezone conversions
- concurrent writes

A monotonically increasing server-side sequence is much easier to reason about.

---

# 13. Deletions

Never immediately destroy synchronization information.

Instead:

```text id="g3r9kx"
deleted_at != NULL
```

The deletion must remain visible to the sync system long enough for clients to receive it.

Eventually implement cleanup:

```text id="b5j0c2"
DELETE FROM events
WHERE deleted_at < retention_threshold
```

Only after all relevant synchronization guarantees are satisfied.

---

# 14. Idempotency

This is critical.

A network request can succeed while the client thinks it failed:

```text id="w4t2yn"
POST /events
    ↓
server creates event
    ↓
network dies
    ↓
client receives timeout
```

Client retries:

```text id="r7s6wm"
POST /events
```

Without idempotency:

```text id="z1q5ka"
Dinner
Dinner
```

Use an idempotency key:

```http id="8j2x5m"
Idempotency-Key: <operation-id>
```

Server stores the result for the operation.

Retrying the same operation becomes safe.

---

# 15. Conflict detection

A conflict happens when:

```text id="v1y5eq"
Local version
       +
Remote version
       +
Both changed
```

Use version information.

For example:

```text id="8y4q3k"
version
updated_at
```

or a server revision.

Update:

```text id="n7s0fd"
PATCH /events/:id
If-Version: 42
```

If the server is now version 43:

```text id="0e3gk8"
409 Conflict
```

The client can then retrieve both versions.

---

# 16. Conflict model

Store:

```text id="n4x7vq"
sync_conflicts
├── id
├── entity_type
├── entity_id
├── local_version
├── remote_version
├── detected_at
├── status
└── resolution
```

Status:

```text id="h2z5nx"
pending
resolved
ignored
```

---

# 17. Conflict resolution

Start simple.

CLI:

```bash id="2y9b8f"
lich conflicts
```

Example:

```text id="7f4k2m"
1. Dinner

Local:
  Aug 20 19:00

Server:
  Aug 20 20:00

[c] choose local
[r] choose remote
```

Later:

```bash id="o3h5d7"
lich conflict resolve <id> --local
```

Do not build automatic three-way merging in Phase 4.

---

# 18. Sync status

Expose a unified state:

```text id="x1b6ry"
SYNCED
SYNCING
OFFLINE
PENDING
FAILED
CONFLICT
```

TUI:

```text id="6j8y4v"
Lich ● Synced
```

or:

```text id="w2v6px"
Lich ◌ Syncing · 3 pending
```

or:

```text id="s7g3km"
Lich ! 1 conflict
```

---

# 19. `lich status`

Make this useful:

```bash id="j5n2cz"
lich status
```

```text id="5q3m1f"
Lich

Server       ✓ online
Local DB     ✓ healthy

Sync
  Status     syncing
  Pending    3
  Failed     0
  Conflicts  1

Last sync
  2026-08-18 11:02:31
```

This becomes an important debugging tool.

---

# 20. Offline detection

Don't constantly ping the server.

Use actual request failures to determine connectivity.

```text id="k3x9zq"
request
   ↓
success → ONLINE
failure → potentially OFFLINE
```

Avoid:

```text id="h8m4qw"
ping server every second
```

because that wastes network resources.

---

# 21. TUI behavior

While synchronization runs:

```text id="r6d0wx"
┌─────────────────────────────────────────┐
│ August 2026                 ◌ Syncing 3 │
├─────────────────────────────────────────┤
│                                         │
│          18      19      20             │
│                  Dinner                 │
│                                         │
└─────────────────────────────────────────┘
```

The user can continue:

```text
n
e
d
navigate
search
```

Nothing should freeze.

---

# 22. CLI behavior

Mutations should return immediately:

```bash id="3z8n7c"
lich add "Dinner tomorrow 7pm"
```

```text id="k4s2p9"
✓ Added Dinner
  Aug 19 · 19:00
  Sync: pending
```

Not:

```text id="k9r2s4"
Creating...
Connecting...
Uploading...
Waiting...
✓
```

---

# 23. Manual synchronization

Provide:

```bash id="m5j9qw"
lich sync
```

Behavior:

```text id="9g2d7s"
$ lich sync

Local changes:  3
Remote changes: 5

✓ Applied 8 changes
✓ No conflicts
```

Optional:

```bash id="p3w8k1"
lich sync --wait
```

This is useful for scripts and debugging.

Normal commands remain asynchronous.

---

# 24. Server restart behavior

Test:

```text id="a6n1kx"
Client
  ↓
pending operations
  ↓
server crashes
  ↓
operations remain pending
  ↓
server returns
  ↓
sync resumes
```

Neither side should lose state.

---

# 25. Multiple client instances

Eventually users may run:

```text id="y5c2zr"
lich
lich
```

or:

```text id="h8s1xv"
desktop client
mobile client
TUI
```

The sync protocol must tolerate multiple clients.

Each client gets its own:

```text id="q2v4gm"
client_id
cursor
```

The server remains authoritative for ordering.

---

# 26. Tests

### Queue

- [ ] Create job
- [ ] Persist job
- [ ] Load pending jobs
- [ ] Coalesce CREATE + UPDATE
- [ ] Coalesce UPDATE + DELETE
- [ ] Cancel redundant operations
- [ ] Recover queue after restart

### Worker

- [ ] Successful operation
- [ ] Retry transient failure
- [ ] Stop retrying permanent failure
- [ ] Backoff
- [ ] Jitter
- [ ] Worker restart
- [ ] Concurrent worker safety

### Pull

- [ ] Receive changes
- [ ] Apply create
- [ ] Apply update
- [ ] Apply delete
- [ ] Persist cursor
- [ ] Resume from cursor
- [ ] Invalid cursor handling

### Idempotency

- [ ] Duplicate request
- [ ] Timeout after server success
- [ ] Retry produces one event
- [ ] Same operation returns same result

### Conflicts

- [ ] Local-only update
- [ ] Remote-only update
- [ ] Both update
- [ ] Local delete / remote update
- [ ] Local update / remote delete
- [ ] Resolve local
- [ ] Resolve remote

### Offline

- [ ] Create offline
- [ ] Update offline
- [ ] Delete offline
- [ ] Restart while offline
- [ ] Reconnect
- [ ] All operations eventually synchronize

---

# 27. E2E test

The most important test:

```text id="h9q1sx"
Start server
    ↓
Start client
    ↓
Create event
    ↓
Disconnect network
    ↓
Edit event
    ↓
Create another event
    ↓
Restart client
    ↓
Restore network
    ↓
Run sync
    ↓
Verify server
```

Expected:

```text id="f4v7b2"
Server state == final intended local state
```

with no duplicate events.

---

# 28. Deliverables

### `lich-tui`

- [ ] Persistent local sync queue
- [ ] Sync worker
- [ ] Retry/backoff
- [ ] Queue coalescing
- [ ] Incremental pull
- [ ] Cursor persistence
- [ ] Conflict detection
- [ ] Conflict resolution
- [ ] Idempotent mutations
- [ ] Offline mode
- [ ] Sync status
- [ ] `lich sync`
- [ ] `lich status`
- [ ] `lich conflicts`

### `lich-server`

- [ ] Change log
- [ ] Monotonic change sequence
- [ ] Sync endpoint
- [ ] Cursor support
- [ ] Soft deletion
- [ ] Idempotency keys
- [ ] Optimistic concurrency/versioning
- [ ] Conflict responses
- [ ] Sync-related API tests

### Documentation

- [ ] Sync protocol
- [ ] State machine
- [ ] Conflict policy
- [ ] Retry policy
- [ ] Offline behavior
- [ ] Idempotency behavior
- [ ] Data retention policy

---

# 29. Definition of Done

The phase is complete when this entire scenario works:

```text id="3k5x8z"
                INTERNET
                   │
                   X
                   │
              ┌────▼────┐
              │ lich-tui│
              │         │
              │ SQLite  │
              └────┬────┘
                   │
              local changes
                   │
             ┌─────▼─────┐
             │ Sync Queue│
             └─────┬─────┘
                   │
              network returns
                   │
             ┌─────▼─────┐
             │   Server  │
             └─────┬─────┘
                   │
             change stream
                   │
             ┌─────▼─────┐
             │  Clients  │
             └───────────┘
```

And:

- **No data loss**
- **No duplicate mutations**
- **No blocking TUI**
- **Offline mutations work**
- **Restart recovery works**
- **Remote changes are eventually received**
- **Conflicts are detected rather than silently overwritten**
- **Synchronization can be inspected and manually triggered**

Phase 4 essentially turns Lich from **"a CLI that talks to a server"** into **"a distributed calendar application with a reliable local-first sync protocol."**
