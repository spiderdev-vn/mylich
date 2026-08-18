# Phase 7 — Multi-Device, Sharing & Collaboration

**Goal:** move Lich from a personal calendar system into a system that can safely support **multiple devices, shared calendars, and multiple users** without breaking the local-first architecture.

```text
                         Lich Server
                              │
              ┌───────────────┼───────────────┐
              │               │               │
          Desktop          Laptop           Phone
          lich-tui         lich-tui          future
              │               │               │
              └───────────────┴───────────────┘
                              │
                         Sync Protocol
                              │
                    ┌─────────┴─────────┐
                    │                   │
               My Calendar         Shared Calendar
```

---

# 1. Multi-device identity

Until now, synchronization can conceptually be:

```text
user ↔ server
```

Phase 7 makes it:

```text
user
 ├── device A
 ├── device B
 └── device C
```

Add:

```text
devices
├── id
├── user_id
├── name
├── platform
├── client_version
├── last_seen_at
├── created_at
└── revoked_at
```

Example:

```text
My Devices

✓ Desktop
  Windows · lich 0.7.0
  Last seen: 2 min ago

✓ Laptop
  Linux · lich 0.7.0
  Last seen: yesterday
```

---

# 2. Device authentication

Don't use the user's main password/token permanently inside the CLI.

Device flow:

```text
lich login
    ↓
server authentication
    ↓
device credential
    ↓
stored locally
```

Use a revocable device/session token.

```text
device_sessions
├── id
├── user_id
├── device_id
├── token_hash
├── created_at
├── last_used_at
├── expires_at
└── revoked_at
```

---

# 3. Device management

CLI:

```bash
lich device list
lich device revoke <id>
lich device rename <id>
```

Example:

```text
Devices

✓ Desktop
  Windows
  Current device

✓ ThinkPad
  Linux
  Last seen 3h ago

✗ Old PC
  revoked
```

---

# 4. Multi-device synchronization

The existing Phase 4 sync engine should now support:

```text
Device A
    │
    ▼
Server
    │
    ▼
Device B
```

Instead of:

```text
Device A
    │
    ▼
Device B
```

The server remains the coordination point.

This keeps conflict resolution manageable.

---

# 5. Shared calendars

Introduce calendar ownership:

```text
calendars
├── id
├── owner_id
├── name
├── timezone
├── visibility
└── ...
```

Previously:

```text
calendar → user
```

Now:

```text
calendar → owner
             │
             ├── member A
             ├── member B
             └── member C
```

---

# 6. Calendar membership

```text
calendar_members
├── calendar_id
├── user_id
├── role
├── status
├── invited_at
├── joined_at
└── removed_at
```

Roles:

```text
owner
editor
viewer
```

Keep the permission model small initially.

---

# 7. Permission matrix

| Action          | Owner | Editor | Viewer |
| --------------- | ----: | -----: | -----: |
| View calendar   |     ✓ |      ✓ |      ✓ |
| Create event    |     ✓ |      ✓ |        |
| Update event    |     ✓ |      ✓ |        |
| Delete event    |     ✓ |      ✓ |        |
| Manage members  |     ✓ |        |        |
| Delete calendar |     ✓ |        |        |
| Change settings |     ✓ |        |        |

Do authorization on the **server**, never trust the TUI.

---

# 8. Invitations

CLI:

```bash
lich calendar invite <calendar> user@example.com
```

Server creates:

```text
calendar_invitations
├── id
├── calendar_id
├── inviter_id
├── invitee_email
├── role
├── token_hash
├── expires_at
└── accepted_at
```

Flow:

```text
Owner
 ↓
Invite
 ↓
Invitation
 ↓
User accepts
 ↓
Calendar appears
```

---

# 9. Invitation CLI

```bash
lich invite list
lich invite accept <id>
lich invite decline <id>
```

TUI can show:

```text
Invitations

John invited you to:

Family Calendar
Role: Editor

[ Accept ] [ Decline ]
```

---

# 10. Shared event synchronization

This introduces an important distinction:

```text
Personal event
```

versus:

```text
Shared event
```

The sync protocol should not care.

Both are simply:

```text
event
```

with authorization determined by its calendar.

This keeps the sync engine clean.

---

# 11. Shared calendar conflict

Example:

```text
Device A:
Dinner → 19:00

Device B:
Dinner → 20:00

Both offline
     ↓
Server receives both
```

The Phase 4 conflict system becomes essential.

Do not introduce special "shared calendar conflict logic".

Reuse:

```text
version
revision
conflict
resolution
```

---

# 12. Event ownership

Don't make individual events independently owned unless necessary.

Prefer:

```text
event
 └── calendar_id
        └── permissions
```

rather than:

```text
event.owner_id
```

This makes shared calendars much simpler.

---

# 13. Event attribution

Although ownership comes from the calendar, record who changed an event.

Add:

```text
created_by
updated_by
```

Example:

```text
Dinner

Created by: Alice
Last modified by: Bob
```

This becomes useful for collaboration and auditing.

---

# 14. Audit log

Introduce:

```text
audit_logs
├── id
├── user_id
├── action
├── entity_type
├── entity_id
├── metadata
└── created_at
```

Examples:

```text
Alice created Dinner
Bob changed Dinner
Alice removed Charlie
```

This is especially important for shared calendars.

---

# 15. Audit CLI

```bash
lich calendar history <calendar>
```

Example:

```text
Calendar History

10:42 Alice created "Dinner"
11:02 Bob changed "Dinner" → 20:00
11:05 Alice added Charlie
```

---

# 16. Soft deletion

For shared data:

```text
DELETE
```

should generally mean:

```text
deleted_at = now()
```

rather than immediately destroying the record.

This allows:

- synchronization
- audit
- recovery
- conflict resolution

Eventually provide:

```bash
lich event restore <id>
```

if appropriate.

---

# 17. Calendar sharing links

Optional but useful:

```bash
lich calendar share <id>
```

Could generate:

```text
https://lich.example.com/share/abc123
```

The link could provide:

```text
read-only calendar
```

without requiring a Lich account.

Keep this separate from membership.

---

# 18. Public calendar access

If implemented, use a random capability token.

Never:

```text
/share/calendar/123
```

where `123` is guessable.

Use something like:

```text
/share/<high-entropy-token>
```

Store only a hash of the token server-side.

Allow the owner to revoke it.

---

# 19. Calendar visibility

Possible values:

```text
private
shared
public
```

But avoid making "public" overly powerful.

A public calendar should initially mean:

```text
read-only
```

No modification through public links.

---

# 20. Timezones

Shared calendars make timezone handling more important.

Calendar:

```text
timezone = Asia/Ho_Chi_Minh
```

Event:

```text
19:00
```

Different device:

```text
America/New_York
```

The UI can display:

```text
Dinner
19:00 GMT+7
08:00 GMT-4
```

but the canonical event remains timezone-aware.

---

# 21. Recurrence + sharing

Recurring events should remain a single logical event:

```text
Weekly meeting
RRULE:FREQ=WEEKLY;BYDAY=MO
```

Don't synchronize thousands of generated occurrences.

Only materialize occurrences when necessary for:

- display
- reminders
- exception handling

---

# 22. Recurrence exceptions

Eventually support:

```text
Weekly meeting
 ├── Aug 24 → 10:00
 ├── Aug 31 → cancelled
 └── Sep 7  → normal
```

Model exceptions separately rather than modifying the entire recurrence rule.

```text
event_recurrence_exceptions
├── event_id
├── occurrence_date
├── type
└── replacement_data
```

---

# 23. Google + shared calendars

Phase 3 Google integration now needs to respect ownership.

Example:

```text
Lich Shared Calendar
        │
        └── Google Calendar
```

Only authorized users should be able to modify the Google mapping.

Do not allow an editor to silently change the calendar's Google integration.

---

# 24. Notification permissions

Shared calendars create notification privacy concerns.

A viewer shouldn't automatically receive every private notification configured by the owner.

Separate:

```text
calendar membership
```

from:

```text
notification subscriptions
```

Example:

```text
Alice:
  receives reminders

Bob:
  does not

Charlie:
  receives event-created webhook
```

---

# 25. Subscription model

Add:

```text
calendar_subscriptions
├── calendar_id
├── user_id
├── notification_type
├── provider
└── enabled
```

This gives each user control over notifications.

---

# 26. API changes

Add:

```http
GET    /calendars/:id/members
POST   /calendars/:id/members
DELETE /calendars/:id/members/:userId

GET    /calendars/:id/invitations
POST   /calendars/:id/invitations

GET    /calendars/:id/history
```

Device:

```http
GET    /devices
DELETE /devices/:id
PATCH  /devices/:id
```

---

# 27. Authorization layer

Create a centralized service:

```text
src/auth/
├── authorization.ts
├── permissions.ts
└── policies.ts
```

Example:

```ts
can(user, "event.update", event);
```

Do not scatter checks everywhere:

```ts
if (user.id === ...)
```

Centralizing permissions prevents security bugs as the application grows.

---

# 28. TUI UX

Calendar list:

```text
Calendars

Personal             owner
Work                 owner
Family               editor
John's Calendar      viewer
```

Calendar details:

```text
Family

Members
  You       owner
  Alice     editor
  Bob       viewer

Integrations
  Google ✓

Notifications
  Enabled
```

---

# 29. CLI UX

Examples:

```bash
lich calendar list

lich calendar members family

lich calendar invite family alice@example.com --role editor

lich calendar remove family alice@example.com

lich calendar history family
```

Sharing:

```bash
lich calendar share family
lich calendar unshare family
```

---

# 30. Tests

### Devices

- [ ] Register device
- [ ] Authenticate device
- [ ] List devices
- [ ] Revoke device
- [ ] Revoked device rejected
- [ ] Multiple devices synchronize

### Permissions

- [ ] Owner permissions
- [ ] Editor permissions
- [ ] Viewer permissions
- [ ] Unauthorized access rejected
- [ ] Cross-user isolation

### Invitations

- [ ] Create invitation
- [ ] Accept
- [ ] Decline
- [ ] Expiration
- [ ] Duplicate invitation
- [ ] Revoked invitation

### Shared calendars

- [ ] Create
- [ ] Add member
- [ ] Remove member
- [ ] Create event as editor
- [ ] Reject event creation as viewer
- [ ] Shared event synchronization

### Conflicts

- [ ] Two devices offline
- [ ] Both modify same event
- [ ] Server detects conflict
- [ ] User resolves conflict
- [ ] All devices converge

### Audit

- [ ] Event creation logged
- [ ] Event update logged
- [ ] Member changes logged
- [ ] Unauthorized actions not logged as successful actions

---

# 31. E2E acceptance test

```text
Alice creates "Family"
        ↓
Alice invites Bob
        ↓
Bob accepts
        ↓
Bob opens lich
        ↓
Family appears
        ↓
Bob creates "Dinner"
        ↓
Alice's lich syncs
        ↓
Dinner appears
```

Offline:

```text
Alice ───── offline ─────┐
                         │
                         ▼
                    both edit
                         │
                         ▼
                      Server
                         │
                         ▼
                     Conflict
                         │
                         ▼
                  User resolves
                         │
                         ▼
                 devices converge
```

---

# 32. Deliverables

### `lich-server`

- [ ] Multi-device identity
- [ ] Device sessions
- [ ] Device revocation
- [ ] Shared calendars
- [ ] Calendar memberships
- [ ] Roles/permissions
- [ ] Invitations
- [ ] Audit log
- [ ] Shared-calendar synchronization
- [ ] Public read-only sharing
- [ ] Central authorization layer
- [ ] Per-user notification subscriptions

### `lich-tui`

- [ ] Device management
- [ ] Calendar sharing
- [ ] Member management
- [ ] Invitations
- [ ] Shared calendar UI
- [ ] Calendar history
- [ ] Permission-aware actions
- [ ] Conflict resolution UI

### API

- [ ] Calendar membership endpoints
- [ ] Invitation endpoints
- [ ] Device endpoints
- [ ] Audit endpoints
- [ ] Public sharing endpoints

---

# 33. Definition of Done

The complete Phase 7 experience:

```text
                         Lich Server
                              │
               ┌──────────────┼──────────────┐
               │              │              │
            Alice          Bob            Charlie
               │              │              │
            Device A       Device B       Device C
               │              │              │
               └──────────────┼──────────────┘
                              │
                         Shared Calendar
                              │
                   ┌──────────┼──────────┐
                   │          │          │
                 Owner      Editor     Viewer
```

And the system guarantees:

- **Multiple devices can safely synchronize**
- **Calendars can be shared**
- **Permissions are enforced server-side**
- **Offline changes still work**
- **Conflicts are detected**
- **Changes are auditable**
- **Invitations are revocable**
- **Public sharing is read-only and revocable**
- **Notifications remain per-user**
- **Google integration continues working**

At this point the architecture becomes:

```text
                     ┌───────────────┐
                     │  lich-server  │
                     │               │
                     │ Calendar Core │
                     │ Sync          │
                     │ Events        │
                     │ Automation    │
                     │ Sharing       │
                     └───────┬───────┘
                             │
        ┌────────────────────┼────────────────────┐
        │                    │                    │
    lich-tui             Google              Webhooks
        │
   Local SQLite
```

**Phase 8** can then focus on making Lich a polished product: **TUI UX, search, recurring-event UX, natural-language commands, import/export, backups, observability, packaging, and production deployment.**
