# Walkthrough: Phase 3 - Google Calendar Integration & Directional Sync

Tất cả các tính năng của **Phase 3 (Tích hợp Google Calendar)** và **Directional Sync (`push` / `pull` / `both`)** đã được hoàn thành và kiểm thử toàn diện 100%.

---

## 1. Directional Sync (`lich sync [push|pull|both]`)

Hỗ trợ đồng bộ có định hướng:
- `lich sync` hoặc `lich sync both`: Đồng bộ 2 chiều (Push local changes $\rightarrow$ Server $\rightarrow$ Pull remote changes).
- `lich sync push`: Chỉ đẩy các thao tác cục bộ trong hàng đợi (`sync_jobs`) lên máy chủ.
- `lich sync pull`: Chỉ kéo các thay đổi mới từ máy chủ về SQLite cache cục bộ.
- Hỗ trợ các cờ: `-w` / `--wait` (hiển thị live progress bar), `-s` / `--simple` (ASCII plain text), `-d` / `--direction <push|pull|both>`.

---

## 2. Phase 3: Tích Hợp Google Calendar

### Kiến Trúc Server (`lich-server`)
1. **Migration 003**: Tạo cấu trúc bảng `integrations`, `integration_credentials`, `calendar_integrations`, `event_integrations`, `integration_sync_state`, `conflicts`.
2. **Provider Abstraction**:
   - `CalendarProvider` interface.
   - `GoogleProvider`: Tích hợp Google Calendar REST API (`oauth2.googleapis.com`, `googleapis.com/calendar/v3`).
   - `FakeGoogleProvider`: Mock provider hỗ trợ test và phát triển offline mà không cần credentials thật.
3. **Event Mapper (`GoogleEventMapper`)**:
   - Chuyển đổi 2 chiều chính xác giữa `Lich Event` $\leftrightarrow$ `Google Calendar Event`.
   - Bảo toàn múi giờ IANA, sự kiện tính theo giờ (`dateTime`) và sự kiện cả ngày (`date`).
4. **Integration Service & Fastify Endpoints**:
   - `GET /integrations/google/auth-url`: Sinh URL xác thực OAuth2.
   - `GET /auth/google/callback`: Nhận mã xác thực từ trình duyệt và lưu trữ tokens.
   - `GET /integrations/google/status`: Báo cáo trạng thái liên kết và danh sách lịch đã ánh xạ.
   - `GET /integrations/google/calendars`: Liệt kê các lịch trên Google Calendar.
   - `POST /integrations/google/map`: Thiết lập ánh xạ giữa lịch Lich và lịch Google.
   - `POST /integrations/google/sync`: Thực hiện đồng bộ 2 chiều hoặc 1 chiều có incremental cursor (`syncToken`).
   - `DELETE /integrations/google`: Hủy liên kết và thu hồi token.

### CLI Client (`lich-cli`)
- `lich google connect`: Mở trình duyệt web để người dùng đăng nhập tài khoản Google.
- `lich google status`: Xem trạng thái kết nối, tài khoản và danh sách lịch ánh xạ.
- `lich google calendars`: Liệt kê các lịch Google Calendar của tài khoản.
- `lich google map <cal_id> <ext_cal_id> [sync_direction]`: Ánh xạ lịch Lich sang Google Calendar.
- `lich google sync [--direction push|pull|both] [--calendar id]`: Kích hoạt đồng bộ hóa tức thì với Google.
- `lich google disconnect`: Hủy liên kết tài khoản Google.

---

## 3. Kết Quả Kiểm Thử

### Backend Unit & Integration Tests (`lich-server`)
```bash
node --test test/**/*.test.ts
```
- **22/22 suites passed (100% PASS)**:
  - Auth Endpoints
  - Calendar Endpoints
  - Event Endpoints
  - Multi-Tenant Isolation
  - Sync Endpoints
  - **Google Calendar Integration Tests** (EventMapper, OAuth Flow, Auto-mapping, Bidirectional Sync, Disconnect)

### Go Unit Tests (`lich-cli`)
```bash
go test -v -count=1 ./...
```
- **100% PASS** trên toàn bộ các package:
  - `lich-cli/internal/api`: PASS
  - `lich-cli/internal/cache`: PASS
  - `lich-cli/internal/cli`: PASS (bao gồm Google Integration CLI test, Multi-day, Overnight & Edge Cases)
  - `lich-cli/internal/config`: PASS
  - `lich-cli/internal/syncer`: PASS (bao gồm `PushWithProgress` & `PullWithProgress`)
  - `lich-cli/internal/tui`: PASS
  - `lich-cli/internal/ui`: PASS
