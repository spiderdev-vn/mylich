# Mỹ Lích — Lịch của mình, mình tính

> Một cái lịch. Nhưng vì không thích mở trình duyệt nên làm luôn CLI.

**Mỹ Lích** là hệ thống lịch cá nhân **local-first**, cho phép bạn quản lý lịch bằng terminal thông qua CLI/TUI, tự lưu trữ dữ liệu bằng `lich-server`, và đồng bộ với các dịch vụ bên ngoài như Google Calendar.

Tên là **Mỹ Lích**.

Lệnh là **`lich`**.

Không có lý do gì cả. Đừng hỏi.

---

## Có gì trong đây?

```text
                         Mỹ Lích
                            │
              ┌─────────────┴─────────────┐
              │                           │
         lich-cli                   lich-server
         Go + Charm                 Node.js + Fastify
              │                           │
       ┌──────┴──────┐                    │
       │             │                    │
      CLI            TUI                SQLite
       │             │                    │
       └──────┬──────┘                    │
              │                           │
         Local-first                      │
         Local cache                      │
              │                           │
              └──────── REST API ─────────┘
                            │
                            ▼
                    Google Calendar
                    Notifications
                    Webhooks
                    ...
```

### `lich`

CLI + TUI viết bằng Go, dùng [Charm](https://charm.sh/).

```bash
lich add "Họp nhóm" --at 10:00
lich today
lich week
lich sync
```

### `lich-server`

Backend tự host, viết bằng Node.js + Fastify + TypeScript.

Nó chịu trách nhiệm:

- Lưu trữ lịch và sự kiện
- Authentication
- Đồng bộ dữ liệu
- Notifications
- Integrations
- Webhooks

---

# Triết lý

## Local-first

Internet chậm không phải lỗi của cái lịch.

Khi bạn chạy:

```bash
lich add "Ăn tối với người yêu" --at 19:00
```

`lich` không ngồi đó:

```text
"Đợi tí để tôi gọi server..."
"Đợi tí để tôi gọi Google..."
"Đợi tí mạng hơi lag..."
```

Thay vào đó:

```text
Bạn
 │
 ▼
lich
 │
 ├── Lưu local       ✓
 ├── Hiển thị        ✓
 └── Sync background ↻
```

Mất mạng vẫn dùng được.

Mạng quay lại thì tự sync.

---

# Bắt đầu nhanh

## 1. Chạy `lich-server`

```bash
cd lich-server
npm install
npm run dev
```

Server mặc định chạy tại:

```text
http://127.0.0.1:3000
```

Kiểm tra:

```bash
curl http://127.0.0.1:3000/health
```

Nếu thấy server trả lời thì mọi thứ đang ổn.

Nếu không thì... server đang không ổn.

---

## 2. Chạy `lich`

Trong lúc development:

```bash
cd lich-cli
go run ./cmd/lich
```

Hoặc chạy một command:

```bash
go run ./cmd/lich today
```

---

# Sử dụng

## Đăng ký / đăng nhập

Đăng ký:

```bash
lich login --register \
  --username alice \
  --password password123
```

Đăng nhập:

```bash
lich login \
  --username alice \
  --password password123
```

Kiểm tra tài khoản:

```bash
lich auth status
```

---

## Thêm sự kiện

Cách đơn giản:

```bash
lich add "Họp nhóm" --at 10:00
```

Chỉ định giờ bắt đầu và kết thúc (`--at` và `--to`):

```bash
lich add "Đi chơi với bạn" --at 10:00 --to 22:33
```

Sự kiện qua đêm (tự động cộng sang ngày hôm sau):

```bash
lich add "Đi quẩy đêm" --at 11:30pm --to 3:00am
```

Dùng thời lượng (`--duration`):

```bash
lich add "Họp nhóm" \
  --at 10:00 \
  --duration 1h
```

Chỉ định ngày tương đối (`today`, `tomorrow`) hoặc `YYYY-MM-DD`:

```bash
lich add "Khám nha khoa" \
  --date tomorrow \
  --at 2:30pm \
  --to 3:15pm
```

Có địa điểm và ghi chú:

```bash
lich add "Họp nhóm" \
  --at 10am \
  --to 11:30am \
  --location "Phòng 101" \
  --desc "Thảo luận kế hoạch sprint mới"
```

---

# Xem lịch

Hôm nay:

```bash
lich today
```

Ngày mai:

```bash
lich tomorrow
```

Tuần này:

```bash
lich week
```

Tháng này:

```bash
lich month
```

Tìm kiếm:

```bash
lich search "nha khoa"
```

JSON cho script:

```bash
lich today --json
```

---

# TUI

Chạy:

```bash
lich
```

Mở giao diện lịch ngay trong terminal.

```text
        August 2026

 Mo   Tu   We   Th   Fr   Sa   Su
             1    2    3    4
  5    6    7    8    9   10   11
 12   13   14   15   16   17   18
                         ↑
                       hôm nay

 19:00  Ăn tối
 21:00  Về nhà
```

Điều hướng:

```text
← ↓ ↑ → / h j k l    Di chuyển
p / n                 Tháng trước / sau
t                     Về hôm nay
r                     Reload
q                     Thoát
Ctrl+C                Thoát
```

TUI được xây dựng bằng:

- Go
- Bubble Tea
- Bubbles
- Lip Gloss

---

# Đồng bộ

Lich được thiết kế để **không phụ thuộc vào mạng trong lúc sử dụng**.

```bash
lich sync
```

Muốn chờ sync hoàn thành:

```bash
lich sync --wait
```

Kiểm tra:

```bash
lich sync status
```

Ví dụ:

```text
Lich

Server       ✓ synced
Google       ✓ synced

Pending      2
Conflicts    0

Last sync    10:42:13
```

---

# Google Calendar

Google Calendar chỉ là **integration**.

Không phải database chính của Mỹ Lích.

```text
                 Mỹ Lích
                    │
              Source of Truth
                    │
                    ▼
             Sync Engine
                    │
                    ▼
             Google Calendar
```

Mục tiêu:

```bash
lich sync google
```

để:

- Pull event từ Google
- Push event từ Mỹ Lích
- Update
- Delete
- Calendar mapping
- Timezone
- Recurring events

---

# Notifications

Mỹ Lích có thể thông báo khi có chuyện xảy ra.

Ví dụ:

```text
19:00
   ↓
"Ăn tối với người yêu"
   ↓
18:45
   ↓
Gotify:
"Ăn tối với người yêu bắt đầu sau 15 phút."
```

Các integration dự kiến:

```text
Gotify
Webhooks
...
```

Cấu trúc:

```text
Calendar Event
      │
      ▼
 Notification System
      │
 ┌────┴────┐
 ▼         ▼
Gotify   Webhook
```

Calendar không cần biết notification được gửi bằng cách nào.

---

# Self-hosting

`lich-server` được thiết kế để tự host.

Hiện tại:

```text
Node.js
Fastify
TypeScript
SQLite
```

Database mặc định là SQLite vì:

> Đây là app lịch, không phải ngân hàng.

PostgreSQL có thể được hỗ trợ khi hệ thống cần scale hơn.

---

# Kiến trúc

```text
Mỹ Lích
│
├── lich-cli
│   │
│   ├── CLI
│   ├── TUI
│   ├── Local SQLite
│   ├── Sync Engine
│   └── Google Calendar integration
│
└── lich-server
    │
    ├── Authentication
    ├── Calendar
    ├── Events
    ├── Sync
    ├── Notifications
    ├── Webhooks
    └── SQLite
```

### Local-first flow

```text
lich add
   │
   ▼
Local SQLite
   │
   ├── show result immediately
   │
   └── queue sync
           │
           ▼
      lich-server
           │
           ▼
       integrations
```

---

# Development

## Server

```bash
cd lich-server

npm install
npm run dev
```

Tests:

```bash
npm test
```

---

## CLI

```bash
cd lich-cli

go run ./cmd/lich
```

Tests:

```bash
go test ./...
```

Build:

```bash
go build -o bin/lich ./cmd/lich
```

Windows:

```powershell
go build -o bin/lich.exe ./cmd/lich
```

---

# Windows

Nếu Windows Smart App Control không thích binary tự build:

```text
Smart App Control:
"Không biết thằng này là ai."

Mỹ Lích:
"Thật ra tôi cũng không biết."
```

Trong development, sử dụng:

```powershell
go run ./cmd/lich
```

hoặc cài vào Go bin:

```powershell
go install ./cmd/lich
```

Binary có thể được build và ký bằng script:

```powershell
.\build-and-sign.cmd
```

---

# API

| Method   | Endpoint         | Mô tả              | Auth  |
| -------- | ---------------- | ------------------ | ----- |
| `GET`    | `/health`        | Health check       | Không |
| `POST`   | `/auth/register` | Đăng ký            | Không |
| `POST`   | `/auth/login`    | Đăng nhập          | Không |
| `GET`    | `/auth/me`       | Tài khoản hiện tại | Có    |
| `GET`    | `/calendars`     | Danh sách lịch     | Có    |
| `POST`   | `/calendars`     | Tạo lịch           | Có    |
| `GET`    | `/calendars/:id` | Chi tiết lịch      | Có    |
| `PATCH`  | `/calendars/:id` | Cập nhật lịch      | Có    |
| `DELETE` | `/calendars/:id` | Xóa lịch           | Có    |
| `GET`    | `/events`        | Danh sách sự kiện  | Có    |
| `POST`   | `/events`        | Tạo sự kiện        | Có    |
| `GET`    | `/events/:id`    | Chi tiết sự kiện   | Có    |
| `PATCH`  | `/events/:id`    | Cập nhật sự kiện   | Có    |
| `DELETE` | `/events/:id`    | Xóa sự kiện        | Có    |

---

# Roadmap

```text
Phase 1   Foundation
    ↓
Phase 2   CLI
    ↓
Phase 3   TUI
    ↓
Phase 4   Local-first Sync
    ↓
Phase 5   Authentication
    ↓
Phase 6   Google Calendar
    ↓
Phase 7   Notifications
    ↓
Phase 8   CLI Polish
    ↓
Phase 9   Self-hosting / Production
```

Mục tiêu cuối cùng:

```text
┌──────────────────────────────────────────┐
│                 MỸ LÍCH                  │
│                                          │
│  Lịch của bạn.                           │
│  Chạy trong terminal.                    │
│  Dữ liệu của bạn.                        │
│  Mạng có thì sync.                       │
│  Mạng không có thì... kệ mạng.           │
│                                          │
└──────────────────────────────────────────┘
```

---

# Bản quyền

MIT License
