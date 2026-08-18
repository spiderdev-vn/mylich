# Phase 8 — CLI Polish

**Goal:** make `lich` feel like a proper Unix-style CLI: fast, predictable, scriptable, and pleasant for both humans and automation.

Core principle:

> **The CLI should feel instant because local operations never depend on network latency.**

---

## 1. Command structure

Standardize the command tree:

```text
lich
├── add
├── edit
├── delete
├── list
├── search
├── today
├── tomorrow
├── week
├── month
│
├── calendar
│   ├── list
│   ├── create
│   ├── edit
│   ├── delete
│   └── use
│
├── sync
│   ├── status
│   └── google
│
├── notify
│   ├── list
│   ├── add
│   ├── remove
│   ├── test
│   └── history
│
├── auth
│   ├── login
│   ├── logout
│   └── status
│
├── doctor
└── version
```

Keep aliases:

```bash
lich rm <id>
lich ls
```

But don't create aliases for everything.

---

# 2. Fast path commands

The most common commands should be extremely short:

```bash
lich add "Dinner tomorrow 7pm"
lich today
lich week
lich search dinner
lich sync
```

The user shouldn't need:

```bash
lich event create ...
```

for normal usage.

---

# 3. Consistent command output

Human output:

```text
✓ Event created

Dinner
Tomorrow · 19:00
Personal
```

Errors:

```text
✗ Failed to create event

Server unavailable.
The event was saved locally and will sync automatically.
```

Avoid dumping raw:

```text
HTTP 503
ECONNRESET
fetch failed
```

unless `--verbose` is enabled.

---

# 4. Exit codes

Define stable exit codes.

```text
0  success
1  general error
2  invalid usage
3  authentication required
4  network/server unavailable
5  conflict
6  validation error
7  sync failure
```

This matters for scripts:

```bash
if lich sync; then
    echo "success"
fi
```

---

# 5. JSON output

Every relevant command should support:

```bash
lich today --json
lich search dinner --json
lich calendar list --json
lich sync status --json
```

Example:

```json
{
  "events": [
    {
      "id": "evt_123",
      "title": "Dinner",
      "start_at": "2026-08-19T19:00:00+07:00",
      "end_at": "2026-08-19T21:00:00+07:00"
    }
  ]
}
```

Important:

> `--json` must produce valid JSON only.

No:

```text
✓ Loading...
```

mixed into stdout.

---

# 6. stdout vs stderr

Follow Unix conventions.

### stdout

Actual command result:

```bash
lich today
```

### stderr

Warnings/errors:

```text
warning: server unavailable
```

This allows:

```bash
lich today --json > events.json
```

without corrupting the JSON.

---

# 7. Quiet mode

Add:

```bash
lich add "Dinner" --quiet
```

Useful for scripts.

Output:

```text
evt_123
```

or no output depending on the command.

---

# 8. Verbose mode

```bash
lich sync --verbose
```

Show:

```text
→ Pulling server changes
→ 3 changes received
→ Pushing 2 local changes
→ Google sync
✓ Complete
```

Useful debugging information should not appear during normal usage.

---

# 9. `lich doctor`

Make this one of the most useful commands.

```bash
lich doctor
```

Example:

```text
Lich Doctor

✓ Local database
✓ Configuration
✓ Authentication
✓ Lich server
✓ Sync engine
✓ Google authentication
✓ Notification providers

No problems found.
```

Failure:

```text
✓ Local database
✓ Authentication
✗ Lich server

  https://lich.example.com is unreachable

  Last successful sync:
  2026-08-18 09:42
```

---

# 10. `lich status`

Quick overview:

```bash
lich status
```

```text
Lich

Server       ✓ online
Google       ✓ connected
Sync         ✓ synced
Pending      2
Conflicts    0

Last sync    10:42:13
```

Unlike `doctor`, this should be fast and not perform expensive diagnostics.

---

# 11. Sync UX

Normal:

```bash
lich sync
```

Should trigger synchronization and return quickly.

For explicit waiting:

```bash
lich sync --wait
```

Output:

```text
Syncing...

✓ Server
✓ Google
✓ 3 events synchronized
```

Timeout:

```bash
lich sync --wait --timeout 10s
```

---

# 12. Event creation UX

Support structured flags:

```bash
lich add "Dinner" \
  --date tomorrow \
  --time 19:00 \
  --duration 2h \
  --calendar personal
```

But natural language remains the convenient form:

```bash
lich add "Dinner tomorrow 7pm"
```

---

# 13. Event editing

Simple:

```bash
lich edit evt_123
```

opens an interactive editor.

Scriptable:

```bash
lich edit evt_123 --title "Dinner with Alice"
lich edit evt_123 --start "2026-08-20T19:00"
```

Don't require the TUI for automation.

---

# 14. Event deletion

```bash
lich delete evt_123
```

Interactive:

```text
Delete "Dinner"?

[y/N]
```

Automation:

```bash
lich delete evt_123 --yes
```

Never ask for confirmation when stdin isn't interactive.

---

# 15. Date parsing

Support useful shortcuts:

```text
today
tomorrow
yesterday
monday
next monday
2026-08-20
```

Time:

```text
7pm
19:00
7:30pm
```

Timezone-aware:

```text
tomorrow 7pm Asia/Ho_Chi_Minh
```

Internally normalize timestamps carefully.

---

# 16. Calendar selection

Current calendar:

```bash
lich calendar use personal
```

Then:

```bash
lich add "Dinner tomorrow 7pm"
```

uses:

```text
personal
```

Override:

```bash
lich add "Meeting" --calendar work
```

Show current:

```bash
lich calendar current
```

---

# 17. Shell completion

Generate completion:

```bash
lich completion bash
lich completion zsh
lich completion fish
lich completion powershell
```

Support:

- commands
- flags
- calendar names
- possibly event IDs

For example:

```bash
lich calendar <TAB>
```

---

# 18. Environment variables

Allow automation without config files:

```text
LICH_SERVER_URL
LICH_PROFILE
LICH_OUTPUT
LICH_NO_COLOR
```

Avoid environment variables containing tokens unless explicitly supported.

Credentials should remain in the OS credential store where possible.

---

# 19. Profiles

Useful for self-hosting:

```bash
lich --profile personal
lich --profile work
```

Example:

```text
~/.config/lich/
├── profiles/
│   ├── personal/
│   └── work/
└── config.toml
```

Each profile can have:

```text
server
account
local database
```

This makes multiple Lich servers possible.

---

# 20. Configuration file

Use a small config:

```toml
[default]
calendar = "personal"
output = "human"

[server]
url = "https://lich.example.com"

[ui]
color = true
```

Don't put secrets here.

---

# 21. Color and terminal compatibility

Support:

```bash
lich --no-color
```

and:

```text
LICH_NO_COLOR=1
```

Automatically disable styling when:

```text
stdout is not a TTY
```

For example:

```bash
lich today | grep Dinner
```

should not produce terminal escape sequences.

---

# 22. Interactive detection

Commands should behave differently depending on whether they are attached to a terminal.

Interactive:

```bash
lich delete evt_123
```

→ confirmation.

Pipeline:

```bash
echo evt_123 | lich delete
```

→ no interactive prompt.

But destructive operations should still require explicit confirmation flags in scripts where appropriate.

---

# 23. Pagination

For large results:

```bash
lich search meeting
```

Avoid dumping thousands of events.

Options:

```bash
--limit 50
--offset 100
```

For JSON:

```bash
lich search meeting --json --limit 100
```

---

# 24. Timezone display

Human output should use the user's configured timezone:

```text
Dinner
Aug 20 · 19:00
```

Verbose:

```text
2026-08-20T19:00:00+07:00
```

JSON should contain explicit timezone information.

---

# 25. Error presentation

Create structured internal errors:

```go
type CLIError struct {
    Code    string
    Message string
    Cause   error
}
```

Map server errors:

```text
401 → authentication required
403 → permission denied
404 → not found
409 → conflict
429 → rate limited
5xx → server unavailable
```

The CLI should never expose implementation details by default.

---

# 26. `--help`

Every command needs useful help:

```bash
lich add --help
```

```text
Create a calendar event.

Usage:
  lich add <title> [flags]

Examples:
  lich add "Dinner tomorrow 7pm"
  lich add "Meeting" --calendar work --time 10:00

Flags:
  --calendar string
  --date string
  --time string
  --duration duration
  --json
  --quiet
```

Keep help concise.

---

# 27. Man page / documentation generation

Eventually generate:

```text
lich(1)
lich-add(1)
lich-sync(1)
lich-calendar(1)
```

This makes the Unix CLI feel native.

---

# 28. Aliases

Useful:

```text
ls      → list
rm      → delete
cal     → calendar
s       → sync
```

Avoid overly clever aliases.

The canonical command names should always appear in documentation.

---

# 29. Shell-friendly event IDs

IDs should be easy to copy:

```text
evt_01J...
```

Avoid UUIDs if there's no reason to expose them.

ULID/UUIDv7-style IDs are useful because they are:

- unique
- sortable
- distributed-system friendly

---

# 30. `lich open`

Convenient command:

```bash
lich open evt_123
```

Could open the event in the TUI.

Or:

```bash
lich open
```

opens the main TUI.

This gives a nice bridge between CLI and TUI.

---

# 31. TUI fallback

Commands can optionally become interactive when input is incomplete.

```bash
lich add
```

→ open event editor.

```bash
lich edit
```

→ event picker.

```bash
lich calendar
```

→ calendar management TUI.

This makes the CLI useful for both keyboard-driven and command-driven workflows.

---

# 32. Testing

### Command tests

- [ ] Correct parsing
- [ ] Correct API calls
- [ ] Correct local DB operations
- [ ] Correct exit codes
- [ ] JSON output validity
- [ ] stderr/stdout separation
- [ ] `--quiet`
- [ ] `--verbose`
- [ ] `--no-color`

### Interactive tests

- [ ] Confirmation
- [ ] Cancellation
- [ ] TTY detection
- [ ] Non-TTY behavior

### Date parsing

- [ ] today
- [ ] tomorrow
- [ ] weekday
- [ ] ISO timestamp
- [ ] 12-hour time
- [ ] 24-hour time
- [ ] timezone

### Shell

- [ ] Bash completion
- [ ] Zsh completion
- [ ] Fish completion
- [ ] PowerShell completion

---

# 33. Acceptance tests

### Fast local operation

```text
Network unavailable
      ↓
lich add "Dinner"
      ↓
<100ms perceived operation
      ↓
event exists locally
```

### Scriptable

```bash
lich today --json | jq '.events[]'
```

must work without human-oriented output contaminating stdout.

### Failure

```text
Server unavailable
      ↓
lich add
      ↓
local event created
      ↓
exit success
      ↓
sync later
```

### Destructive action

```bash
lich delete evt_123
```

Interactive:

```text
Delete event? [y/N]
```

Automation:

```bash
lich delete evt_123 --yes
```

---

# 34. Deliverables

### CLI

- [ ] Stable command hierarchy
- [ ] Fast-path commands
- [ ] Consistent output
- [ ] Exit codes
- [ ] JSON output
- [ ] Quiet mode
- [ ] Verbose mode
- [ ] `doctor`
- [ ] `status`
- [ ] Sync UX
- [ ] Natural date/time parsing
- [ ] Calendar selection
- [ ] Shell completion
- [ ] Profiles
- [ ] Configuration
- [ ] Color/TTY handling
- [ ] Pagination
- [ ] Error model
- [ ] Complete help

### TUI integration

- [ ] `lich open`
- [ ] Interactive fallback
- [ ] Event picker
- [ ] Calendar picker
- [ ] Consistent keybindings

---

# 35. Definition of Done

A user should be able to do:

```bash
lich add "Dentist tomorrow 9am"
lich today
lich sync
```

without thinking about:

```text
HTTP
SQLite
Google
OAuth
network
sync queues
retries
```

And a script should be able to do:

```bash
lich today --json
```

without caring about terminal formatting.

The CLI's responsibility is simple:

```text
Human
  │
  ▼
lich
  │
  ├── local-first
  ├── fast
  ├── scriptable
  └── asynchronous
          │
          ▼
      Sync Engine
```

**Phase 8 is complete when `lich` feels like a native command-line calendar rather than a thin wrapper around a REST API.**
