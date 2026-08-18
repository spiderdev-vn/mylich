# Lich ⚡

Hệ thống lịch cá nhân cục bộ (local-first) với giao diện dòng lệnh (CLI), giao diện Terminal (TUI) và máy chủ tự lưu trữ (self-hosted backend).

```text
                    ┌─────────────────────────┐
                    │        lich-cli         │
                    │                         │
                    │ Go + Bubble Tea         │
                    │ CLI + Giao diện TUI     │
                    │ Cấu hình & Cache cục bộ │
                    └───────────┬─────────────┘
                                │ REST API
                                ▼
                    ┌─────────────────────────┐
                    │       lich-server       │
                    │                         │
                    │ Node.js + Fastify + TS  │
                    │ SQLite (node:sqlite)    │
                    │ Migrations + Auth       │
                    └─────────────────────────┘
```

---

## Bắt đầu nhanh (Quick Start)

### 1. Khởi động Máy chủ (`lich-server`)

```bash
cd lich-server
npm install
npm run dev
```

Máy chủ sẽ chạy tại `http://127.0.0.1:3000`. Kiểm tra trạng thái:
```bash
curl http://127.0.0.1:3000/health
```

---

### 2. Chạy ứng dụng Client (`lich-cli`)

Trong quá trình phát triển, bạn có thể chạy trực tiếp bằng lệnh `go run`:

```bash
cd lich-cli
go run ./cmd/lich <lệnh>
```

#### Đăng ký & Đăng nhập
```bash
# Đăng ký tài khoản mới (tự động tạo lịch mặc định "Personal")
go run ./cmd/lich login --register --username alice --password password123

# Đăng nhập vào tài khoản đã có
go run ./cmd/lich login --username alice --password password123
```

#### Các lệnh CLI
```bash
# Thêm sự kiện mới
go run ./cmd/lich add "Họp nhóm" --at 10:00 --duration 1h --location "Phòng 101"
go run ./cmd/lich add "Khám nha khoa" --date 2026-08-20 --at 14:30 --duration 45m

# Xem lịch hôm nay
go run ./cmd/lich today
go run ./cmd/lich today --json

# Xem lịch cả tuần
go run ./cmd/lich week

# Xóa sự kiện theo ID
go run ./cmd/lich delete <id-su-kien>
```

#### Mở Giao diện Terminal tương tác (TUI)
```bash
go run ./cmd/lich
```
- **Điều hướng ngày**: Phím mũi tên `←` `↓` `↑` `→` hoặc `h` `j` `k` `l`
- **Chuyển tháng**: `p` (tháng trước) / `n` (tháng sau)
- **Về hôm nay**: `t`
- **Tải lại dữ liệu**: `r`
- **Thoát**: `q` hoặc `Ctrl+C`

---

## Hướng dẫn xử lý Smart App Control trên Windows

Nếu bạn build file thực thi `.exe` (`go build -o bin/lich.exe ./cmd/lich`) trên Windows 11 và gặp thông báo:
> **"Smart App Control has blocked part of this app"**

### Nguyên nhân:
**Smart App Control (SAC)** là tính năng bảo mật của Windows 11 nhằm chặn các file `.exe` chưa có chữ ký số (digital signature) hợp lệ hoặc chưa có điểm danh tiếng (reputation) trên đám mây của Microsoft.

### Các cách khắc phục:

1. **Sử dụng script tự động build và ký chữ ký số (Đơn giản nhất):**
   Chạy file script `build-and-sign.cmd` ngay tại thư mục gốc hoặc trong `lich-cli/`:
   ```powershell
   .\build-and-sign.cmd
   ```
   Script sẽ tự động:
   - Biên dịch mã nguồn Go thành `bin/lich.exe`.
   - Tạo chứng chỉ số cá nhân `CN=LichDev` (nếu chưa có).
   - Ký chữ ký Authenticode trực tiếp vào file `bin/lich.exe` và mở khóa (Unblock-File).
   - Sau đó bạn có thể chạy trực tiếp: `.\lich-cli\bin\lich.exe`.

2. **Khuyên dùng trong lúc phát triển (Development):**
   Chạy trực tiếp mã nguồn thông qua công cụ Go mà không cần build ra file `.exe`:
   ```powershell
   cd lich-cli
   go run ./cmd/lich <các tham số>
   ```
   Go runtime thực thi trực tiếp trong môi trường dev và không bị Windows SAC chặn.

3. **Cài đặt vào `%GOPATH%/bin`:**
   ```powershell
   cd lich-cli
   go install ./cmd/lich
   ```
   Sau đó bạn có thể gõ lệnh `lich` từ bất kỳ đâu trên hệ thống.

---

## Kiểm thử (Testing)

### Backend Tests
```bash
cd lich-server
npm test
```

### Client Tests
```bash
cd lich-cli
go test ./...
```

---

## Tài liệu API (API Reference)

| Phương thức | Đường dẫn (Endpoint) | Mô tả | Yêu cầu xác thực |
| :--- | :--- | :--- | :--- |
| `GET` | `/health` | Kiểm tra kết nối máy chủ | Không |
| `POST` | `/auth/register` | Đăng ký người dùng mới & tạo lịch mặc định | Không |
| `POST` | `/auth/login` | Đăng nhập & lấy Bearer token | Không |
| `GET` | `/auth/me` | Lấy thông tin tài khoản hiện tại | Có |
| `GET` | `/calendars` | Danh sách lịch của người dùng | Có |
| `POST` | `/calendars` | Tạo lịch mới | Có |
| `GET` | `/calendars/:id` | Xem chi tiết lịch | Có |
| `PATCH` | `/calendars/:id` | Cập nhật tên / múi giờ của lịch | Có |
| `DELETE` | `/calendars/:id` | Xóa lịch | Có |
| `GET` | `/events` | Lấy danh sách sự kiện (`from`, `to`, `calendar_id`) | Có |
| `POST` | `/events` | Tạo sự kiện mới | Có |
| `GET` | `/events/:id` | Xem chi tiết sự kiện | Có |
| `PATCH` | `/events/:id` | Cập nhật sự kiện | Có |
| `DELETE` | `/events/:id` | Xóa sự kiện | Có |

---

## Bản quyền

MIT License
