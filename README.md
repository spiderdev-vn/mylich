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

### `lich-cli`

CLI + TUI viết bằng Go, dùng [Bubble Tea & Lip Gloss](https://charm.sh/).

```bash
lich add "Họp nhóm" --at 10:00
lich today
lich week
lich sync -w
lich google status
```

### `lich-server`

Backend tự host, viết bằng Node.js + Fastify + TypeScript + SQLite.

Chịu trách nhiệm:
- Lưu trữ calendars và events
- Authentication (JWT, Multi-tenant)
- Incremental sync engine & changelog cursor
- Tích hợp 2 chiều Google Calendar
- Webhooks & Notifications

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
 ├── Lưu local cache (SQLite) ✓ (Phản hồi tức thì)
 ├── Hiển thị giao diện      ✓
 └── Sync ngầm với Server    ↻ (Tự động retry khi có mạng)
```

Mất mạng vẫn dùng được. Mạng quay lại thì tự sync.

---

# Bắt đầu nhanh

## 1. Khởi động `lich-server`

```bash
cd lich-server
yarn install
yarn dev
```

Server mặc định chạy tại `http://127.0.0.1:3000`.

Kiểm tra trạng thái:
```bash
curl http://127.0.0.1:3000/health
```

---

## 2. Cài đặt và chạy `lich-cli`

Build và ký Authenticode (Windows):
```powershell
.\build-and-sign.cmd
```

Chạy binary đã build:
```powershell
.\bin\lich.exe
```

Hoặc chạy trực tiếp qua Go:
```bash
cd lich-cli
go run ./cmd/lich
```

---

# Hướng dẫn sử dụng CLI

## Đăng nhập & Tài khoản

Đăng ký hoặc đăng nhập tương tác:
```bash
lich login
```

Đăng nhập qua cờ CLI:
```bash
lich login --username alice --password password123
```

Kiểm tra trạng thái máy chủ, local database & sync queue:
```bash
lich status
lich status --simple
```

---

## Quản lý cấu hình (`config`)

Xem và chỉnh sửa cấu hình client:
```bash
lich config
lich config icons nerd       # unicode | nerd | ascii | emoji
lich config agenda gantt     # gantt | list
```

---

## Thêm sự kiện (`add`)

Mở form tương tác TUI (nếu không truyền tham số):
```bash
lich add
```

Thêm nhanh bằng tham số dòng lệnh:
```bash
# Giờ bắt đầu mặc định thời lượng 1 tiếng
lich add "Họp nhóm" --at 10:00

# Chỉ định cả giờ bắt đầu và kết thúc
lich add "Làm việc tại quán cafe" --at 10:00 --to 12:30

# Sự kiện qua đêm (tự động tính sang ngày hôm sau)
lich add "Trực ca đêm" --at 22:00 --to 03:00

# Chỉ định thời lượng
lich add "Tập gym" --at 17:30 --duration 1h30m

# Ngày tương đối (today, tomorrow) hoặc YYYY-MM-DD
lich add "Khám nha khoa" --date tomorrow --at 2:30pm --to 3:15pm

# Đầy đủ địa điểm và mô tả
lich add "Gặp đối tác" --at 14:00 --to 15:30 --location "Quận 1" --desc "Ký kết hợp đồng"
```

---

## Xem lịch trình (`today`, `week`, `month`, `search`)

```bash
lich today             # Lịch hôm nay (Gantt chart hoặc List)
lich week              # Lịch 7 ngày trong tuần
lich month             # Lịch toàn bộ tháng hiện tại
lich search "nha khoa" # Tìm kiếm sự kiện theo từ khóa

# Xuất dữ liệu cho scripts / CI
lich today --simple    # ASCII plain text
lich today --json      # JSON output
```

---

## Chỉnh sửa & Xóa sự kiện (`edit`, `delete`, `nuke`)

```bash
# Chỉnh sửa sự kiện (chọn từ danh sách hoặc truyền ID)
lich edit
lich edit <event_id> --title "Tiêu đề mới" --at 14:00 --to 16:00

# Xóa sự kiện
lich delete
lich delete <event_id> --force

# Xóa sạch toàn bộ dữ liệu SQLite cache cục bộ (yêu cầu xác nhận an toàn)
lich nuke-database
```

---

## Đồng bộ hóa (`sync`)

Hỗ trợ đồng bộ hóa 2 chiều hoặc có định hướng:

```bash
# Đồng bộ 2 chiều (Push & Pull)
lich sync

# Đồng bộ có live progress bar và chi tiết từng thao tác
lich sync -w

# Chỉ đẩy thay đổi cục bộ lên server
lich sync push -w

# Chỉ kéo thay đổi mới từ server về local cache
lich sync pull -w
```

---

# Tích hợp Google Calendar (`google`)

Google Calendar hoạt động như một integration vệ tinh, dữ liệu Lich là **Source of Truth**.

```text
                 Mỹ Lích (Source of Truth)
                            │
               ┌────────────┴────────────┐
               ▼                         ▼
         Local SQLite               lich-server
                                         │
                                         ▼
                                  Google Calendar
```

### Các lệnh Google Calendar:

1. **Kết nối tài khoản**:
   ```bash
   lich google connect
   ```
   *Mở trình duyệt để bạn đăng nhập tài khoản Google và cấp quyền OAuth2.*

2. **Kiểm tra trạng thái kết nối**:
   ```bash
   lich google status
   ```

3. **Xem danh sách lịch Google Calendar**:
   ```bash
   lich google calendars
   ```

4. **Ánh xạ lịch Lich với Google Calendar**:
   ```bash
   lich google map <lich_calendar_id> <google_calendar_id> [sync_direction]
   ```

5. **Đồng bộ hóa tức thì với Google**:
   ```bash
   lich google sync                    # Đồng bộ 2 chiều
   lich google sync -d push            # Đẩy sự kiện Lich lên Google
   lich google sync -d pull            # Kéo sự kiện Google về Lich
   ```

6. **Hủy liên kết**:
   ```bash
   lich google disconnect
   ```

---

# Giao diện TUI tương tác

Chạy lệnh không có tham số:
```bash
lich
```

```text
        Tháng 8 2026

 Mo   Tu   We   Th   Fr   Sa   Su
             1    2    3    4
  5    6    7    8    9   10   11
 12   13   14   15   16   17   18
                         ↑
                       hôm nay

 10:00 - 11:30  Họp nhóm
 19:00 - 21:00  Ăn tối với người yêu
```

### Phím tắt điều khiển:
| Phím | Chức năng |
| ---- | --------- |
| `← ↓ ↑ →` / `h j k l` | Di chuyển ngày / ô lịch |
| `p` / `n` | Tháng trước / Tháng sau |
| `t` | Trở về ngày hôm nay |
| `a` | Mở modal thêm sự kiện mới |
| `e` | Mở modal chỉnh sửa sự kiện đã chọn |
| `d` | Xóa sự kiện đã chọn |
| `Enter` | Xem chi tiết sự kiện |
| `s` | Kích hoạt đồng bộ hóa ngầm |
| `g` | Chuyển đổi chế độ xem Gantt chart / Danh sách |
| `r` | Tải lại dữ liệu từ cache |
| `?` | Bật / tắt bảng trợ giúp |
| `q` / `Ctrl+C` | Thoát ứng dụng |

---

# Kiểm thử & Development

### 1. Backend (`lich-server`)
```bash
cd lich-server
yarn test
```
*Chạy 22 test suites gồm: Auth, Calendars, Events, Multi-tenant Isolation, Sync Engine, Google Calendar Integration.*

### 2. Frontend CLI (`lich-cli`)
```bash
cd lich-cli
go test -v -count=1 ./...
```
*Kiểm thử toàn bộ unit tests của API client, Local cache, Syncer engine, CLI parsing, Time edge cases, TUI components.*

---

# API Endpoints (`lich-server`)

| Phương thức | Đường dẫn | Mô tả | Yêu cầu Auth |
| ----------- | --------- | ----- | ------------ |
| `GET` | `/health` | Kiểm tra server | Không |
| `POST` | `/auth/register` | Đăng ký tài khoản | Không |
| `POST` | `/auth/login` | Đăng nhập lấy JWT | Không |
| `GET` | `/auth/me` | Thông tin tài khoản | Có |
| `GET` | `/calendars` | Danh sách lịch | Có |
| `POST` | `/calendars` | Tạo lịch mới | Có |
| `GET` | `/events` | Lấy danh sách sự kiện theo khoảng thời gian | Có |
| `POST` | `/events` | Tạo sự kiện mới | Có |
| `PATCH` | `/events/:id` | Cập nhật sự kiện | Có |
| `DELETE` | `/events/:id` | Xóa sự kiện (soft-delete) | Có |
| `GET` | `/sync` | Lấy changelog gia tăng theo cursor | Có |
| `GET` | `/integrations/google/auth-url` | Lấy URL OAuth Google | Có |
| `GET` | `/auth/google/callback` | OAuth redirect callback | Không |
| `GET` | `/integrations/google/status` | Trạng thái Google Calendar | Có |
| `GET` | `/integrations/google/calendars` | Danh sách lịch Google | Có |
| `POST` | `/integrations/google/map` | Ánh xạ lịch Lich sang Google | Có |
| `POST` | `/integrations/google/sync` | Đồng bộ 2 chiều với Google | Có |
| `DELETE` | `/integrations/google` | Hủy liên kết Google | Có |

---

# Roadmap

- [x] **Phase 1**: Nền tảng Core API, SQLite Migrations, Multi-tenant, REST Endpoints
- [x] **Phase 2**: CLI Client (CRUD, Time Parsing, Gantt, Config, Directional Sync)
- [x] **Phase 3**: Tích hợp Google Calendar (OAuth2, Incremental syncToken, Event Mapping)
- [ ] **Phase 4**: Notification System (Gotify, Desktop push alerts)
- [ ] **Phase 5**: Webhooks & Automation Actions

---

# Bản quyền

MIT License

