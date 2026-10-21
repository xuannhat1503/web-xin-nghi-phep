# G-ASIAPACIFIC – Hệ thống quản lý xin nghỉ phép (nội bộ)

Ứng dụng web nội bộ quản lý đơn xin nghỉ phép: Go (Gin, `database/sql`, sqlc) + SvelteKit + MySQL 8.

> Email mặc định **tắt gửi** (chỉ lưu ở Nhật ký email); Admin bật trong Cấu hình nghỉ phép → Email (cần `EMAIL_LIVE_ENABLED=true` + SMTP trong `.env`).

## Trạng thái

| Phase | Nội dung | Trạng thái |
|---|---|---|
| 1 | Nền tảng, migrations, seed ngày lễ | ✅ Hoàn thành |
| 2 | Google OIDC, session, kích hoạt tài khoản | ✅ Hoàn thành |
| 3 | Admin: nhân viên, Manager, lịch nghỉ, cài đặt | ✅ Hoàn thành |
| 4 | Working Calendar, form xin nghỉ, auto-save, Review & Confirm | ✅ Hoàn thành |
| 5 | Duyệt, Request Changes, resubmit | ✅ Hoàn thành |
| 6 | Notification, reminder, hàng đợi + nhật ký email | ✅ Hoàn thành |
| 7 | Dashboard, Analytics, timeline, lọc/phân trang | ✅ Hoàn thành |
| 8 | Bảo mật, hiệu năng | ✅ Hoàn thành |

## Cấu trúc thư mục

```
backend/
  cmd/server      HTTP API (+ workers ở các phase sau), graceful shutdown
  cmd/migrate     chạy migrations (nhúng sẵn trong binary)
  cmd/seed        seed dữ liệu tham chiếu (ngày lễ)
  db/migrations   file SQL versioned (golang-migrate)
  db/queries      SQL cho sqlc
  internal/
    config        đọc + validate biến môi trường
    database      pool MySQL, strict mode, kiểm tra MySQL 8, migrations, transaction
    civil         kiểu ngày (không múi giờ)
    lunar         chuyển âm lịch → dương lịch (thuật toán Hồ Ngọc Đức, UTC+7)
    holiday       lịch nghỉ + seed ngày lễ Việt Nam
    auth          Google OIDC, session MySQL, CSRF, middleware phân quyền
    httpapi       router, middleware bảo mật, rate limit, health
    httpx         helper HTTP dùng chung (lỗi JSON)
    worker        job nền định kỳ + heartbeat
    store         code sinh bởi sqlc (không sửa tay)
frontend/
  src/hooks.server.ts   proxy cùng origin → Go, đọc session, chặn route (chỉ UI)
  src/lib/server        proxy + guard
  src/routes/(public)   /login, /pending-activation
  src/routes/(app)      trang cần đăng nhập + active (sidebar/header)
  static/images/logo.png  logo gốc (không chỉnh sửa)
docs/             tài liệu thiết kế
tools/            sqlc.exe (không commit)
```

## Yêu cầu môi trường

- Windows 11, PowerShell
- Go 1.27+ · Node.js 24+ · npm 11+
- WampServer với **MySQL 8.4 ở cổng 3306** (cổng 3307 là MariaDB – không dùng; backend sẽ từ chối nếu kết nối nhầm)
- **Smart App Control phải tắt** (Windows Security → App & browser control). Nếu bật, Windows sẽ chặn các file `.exe` do `go build` sinh ra. Exclusion của Defender không có tác dụng với Smart App Control.

## Cài đặt lần đầu

### 1. Khởi động WampServer
Bảo đảm biểu tượng Wamp màu xanh và dịch vụ MySQL 8.4 đang chạy.

### 2. Tạo database rỗng bằng phpMyAdmin
Mở <http://localhost/phpmyadmin> → **New** → tên `employee_leave_management`, collation `utf8mb4_unicode_ci`.

### 3. Cấu hình `.env`
```powershell
cd D:\AWS\WebXinNghiPhep
Copy-Item .env.example .env
notepad .env
```
Máy dev cá nhân dùng `root` không mật khẩu. File `.env` đã nằm trong `.gitignore`.

### 4. Chạy migrations
```powershell
cd D:\AWS\WebXinNghiPhep\backend
go run ./cmd/migrate up
go run ./cmd/migrate version     # kỳ vọng: version 12 (latest 12), dirty=false
```
Rollback là thao tác **xóa bảng và dữ liệu**, phải xác nhận bằng tên DB:
```powershell
go run ./cmd/migrate down -steps 1 -confirm employee_leave_management
```

### 5. Seed ngày lễ
```powershell
go run ./cmd/seed holidays                   # năm hiện tại và năm sau
go run ./cmd/seed holidays -years 2026,2027
```
Chạy lại bao nhiêu lần cũng được (xem [docs/holidays.md](docs/holidays.md)). Mọi ngày seed đều là **DRAFT**; Admin phải xác nhận trước khi nhân viên gửi đơn cho năm đó.

### 6. sqlc (chỉ khi sửa `db/queries` hoặc schema)
Tải binary chính thức (đã kiểm tra SHA-256 với digest GitHub công bố):
```powershell
cd D:\AWS\WebXinNghiPhep
New-Item -ItemType Directory -Force tools | Out-Null
Invoke-WebRequest https://github.com/sqlc-dev/sqlc/releases/download/v1.31.1/sqlc_1.31.1_windows_amd64.zip -OutFile $env:TEMP\sqlc.zip
(Get-FileHash $env:TEMP\sqlc.zip -Algorithm SHA256).Hash   # 352711FA7DCB05DCDFEFCA0AD71B2C9A74FD090F8D7FC609419DE4CBC725429F
Expand-Archive $env:TEMP\sqlc.zip -DestinationPath tools -Force
cd backend
..\tools\sqlc.exe generate
```

### 7. Cấu hình Google OAuth (localhost)

1. <https://console.cloud.google.com/> → tạo project → **Google Auth Platform / OAuth consent screen**: Audience **External**, trạng thái **Testing**, thêm **Test users** (các Gmail dùng để test, gồm email trong `BOOTSTRAP_ADMIN_EMAILS`).
2. **Clients → Create client → Web application**
   - Authorized JavaScript origins: `http://localhost:5173`
   - Authorized redirect URIs: `http://localhost:5173/auth/google/callback`
3. Điền `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET` vào `.env` (không commit, không gửi qua chat).
4. `SESSION_SECRET`: chuỗi base64 ≥ 32 byte ngẫu nhiên (lệnh tạo có trong `.env.example`).

Scope chỉ gồm `openid email profile` (không cần Google xét duyệt). Ứng dụng không lưu mật khẩu hay access token Google.

### 8. Chạy backend
```powershell
cd D:\AWS\WebXinNghiPhep\backend
go run ./cmd/server
```
Kiểm tra:
```powershell
curl.exe http://127.0.0.1:8080/health/live    # {"status":"ok"}
curl.exe http://127.0.0.1:8080/health/ready   # {"checks":{"database":"ok","migrations":"ok"},"status":"ok"}
```
Dừng bằng `Ctrl+C` (graceful shutdown: HTTP → workers → DB).

### 9. Chạy frontend (cửa sổ PowerShell thứ hai)
```powershell
cd D:\AWS\WebXinNghiPhep\frontend
npm install        # lần đầu
npm run dev        # http://localhost:5173
```
Trình duyệt chỉ làm việc với `http://localhost:5173`. SvelteKit proxy `/api`, `/auth`, `/health` sang Go (cùng origin → cookie HttpOnly, không cần CORS).

### 10. Đăng nhập Admin
Mở <http://localhost:5173> → **Sign in with Google** → chọn tài khoản có email trong `BOOTSTRAP_ADMIN_EMAILS`.
Tài khoản có trong `BOOTSTRAP_ADMIN_EMAILS` được tạo thành **ADMIN active ở lần đăng nhập đầu tiên** (chỉ khi tạo mới; Admin bị hạ quyền sau này không tự được nâng lại).

### 11. Tài khoản khác
Gmail khác đăng nhập lần đầu → tạo **EMPLOYEE chưa kích hoạt** → trang *Pending activation*. Admin kích hoạt và gán Manager ở `/admin/employees`.

### Ngôn ngữ giao diện

Mặc định **tiếng Việt**; nút **VI / EN** ở góc trên (lưu bằng cookie `lang`). Email luôn bằng tiếng Anh theo mẫu `mailsample.txt`.

### Tạo đơn nghỉ (Phase 4)

1. Menu **Tạo đơn nghỉ** → **Tạo đơn mới** (tạo bản nháp) hoặc **Tiếp tục** một bản nháp.
2. Chọn loại nghỉ, *Cả ngày / Nửa ngày sáng / Nửa ngày chiều / Tùy chọn giờ*, ngày dạng **dd/MM/yyyy** (lịch chọn ngày đánh dấu ngày lễ / làm bù / ngày nghỉ), lý do.
3. Thời gian nghỉ dự kiến được tính ngay (bỏ ngày nghỉ, ngày lễ đã xác nhận, giờ nghỉ trưa; tính ngày làm bù). Bản nháp **tự lưu** sau ~1,5 giây.
4. **Xem lại & gửi** → kiểm tra thông tin + xem trước email → **Xác nhận & gửi**. Backend kiểm tra lại toàn bộ: chính sách loại nghỉ, lịch đã xác nhận, Manager, trùng đơn; cấp mã `LR-YYYY-NNNNNN`.

Điều kiện gửi được: lịch nghỉ các năm liên quan đã **CONFIRMED**, nhân viên **có Manager đang hoạt động**.

### Duyệt đơn, thông báo, báo cáo (Phase 5–8)

| Trang | Ai dùng | Chức năng |
|---|---|---|
| `/leave/history` | Nhân viên | Lịch sử đơn của mình, lọc theo tháng/năm/trạng thái/loại/mã/từ khóa/khoảng ngày, 10 đơn/trang |
| `/leave/[id]` | Chủ đơn | Chi tiết, dòng thời gian, các lần gửi; **Hủy đơn** (PENDING/NEEDS_CHANGES); sửa và gửi lại khi cần bổ sung |
| `/leader/dashboard`, `/leader/requests`, `/leader/requests/[id]` | Người duyệt | Chỉ đơn đang được giao; **Duyệt / Từ chối / Yêu cầu bổ sung** (2 lựa chọn sau bắt buộc lý do), có hộp xác nhận |
| `/notifications` | Nhân viên, người duyệt | Thông báo trong web; mở trang là tất cả được đánh dấu đã đọc (chuông trên header) |
| `/admin/dashboard` | Admin | KPI và 3 biểu đồ theo năm/tháng (chỉ số liệu tổng hợp) |
| `/admin/notifications`, `/admin/email-log` | Admin | Hàng đợi email (thử lại khi lỗi) và Nhật ký email (nội dung + trạng thái đã gửi) |
| `/admin/system` | Admin | DB, migration, tác vụ nền, hàng đợi, lỗi vận hành (đã lọc) |

Tác vụ nền trong `cmd/server`: gửi email (30 giây), nhắc Leader (5 phút, theo cài đặt), dọn session, idempotency key, nhật ký email cũ hơn 30 ngày.

Tài khoản **Admin vừa quản lý vừa xin nghỉ được** (như nhân viên: cần có Manager mới gửi được đơn). Sau đăng nhập Admin vào `/admin/dashboard`; mục "Đơn của tôi" vẫn có trong menu.

Email: gửi đơn → TO manager; duyệt → TO nhân viên, và một email riêng cho Finance ("Apply for Leave_… [Approved]", ghi người duyệt; địa chỉ Finance đầu tiên là TO, còn lại CC); từ chối / yêu cầu sửa → TO nhân viên; hủy → TO manager. Hai trạng thái (tab Email):
- **Tắt gửi email**: email chỉ được lưu ở Nhật ký email.
- **Bật gửi email**: gửi tới đúng người nhận. Chỉ chọn được khi `.env` có `EMAIL_LIVE_ENABLED=true`; mọi người nhận phải thuộc `ALLOWED_EMAILS` / `ALLOWED_EMAIL_DOMAIN` (nếu đặt).

`ALLOWED_EMAILS` (danh sách email) / `ALLOWED_EMAIL_DOMAIN` (ví dụ `g-asiapac.com.vn`) cũng giới hạn **đăng nhập**; email trong `BOOTSTRAP_ADMIN_EMAILS` luôn được phép. Để trống cả hai = không giới hạn (chỉ dùng khi dev).

### 12. Quản trị (Phase 3)

| Trang | Chức năng |
|---|---|
| `/admin/employees` | Tìm kiếm, lọc *Pending activation* / *No manager*, phân trang 20/trang; kích hoạt/vô hiệu hóa (thu hồi phiên ngay), đổi role, gán/bỏ Manager kèm lý do; lịch sử thay đổi |
| `/admin/holidays` | Chọn năm, xem List/Calendar, lọc loại/trạng thái; *Suggest Vietnamese holidays* (DRAFT), thêm/sửa/xóa nháp, xác nhận từng ngày hoặc cả năm; cảnh báo lịch thiếu; nhật ký thay đổi |
| `/admin/settings` | **Cấu hình nghỉ phép**, chia tab: Giờ làm việc · Quy định nghỉ · Nhắc duyệt (lưu chung thành **phiên bản policy mới, bất biến**) · Loại nghỉ · Lý do nghỉ (thêm/sửa/xóa; đã dùng trong đơn thì chỉ ngừng dùng) · Email (bật/tắt gửi, danh sách Tài chính) · Lịch sử |

Quy tắc chính: không tự làm Manager của mình, không tạo chu trình Manager (khóa toàn cục chống race), Manager phải active, luôn còn ít nhất 1 Admin active, Admin không tự hạ quyền/vô hiệu hóa. Sửa lịch của năm đã CONFIRMED sẽ mở lại năm (DRAFT) và đánh dấu đơn bị ảnh hưởng — không bao giờ tính lại đơn đã duyệt.

## Bảo mật đăng nhập & phiên

- Google OIDC Authorization Code + **PKCE (S256)**, `state` (lưu server, dùng 1 lần, 10 phút, gắn cookie trình duyệt) và `nonce`.
- ID token được kiểm tra chữ ký (JWKS), `iss`, `aud`, `exp`, `nonce`, `email_verified`. Định danh = Google `sub`.
- Session server-side trong MySQL: token ngẫu nhiên 256-bit, **chỉ lưu SHA-256**; cookie `HttpOnly`, `SameSite=Lax`, `Secure` + tiền tố `__Host-` khi HTTPS.
- Idle timeout 30 phút (trượt), absolute 12 giờ; rotation khi đăng nhập; logout thu hồi; worker dọn session hết hạn mỗi 10 phút; session còn hiệu lực sau khi restart.
- CSRF: kiểm tra `Origin` + synchronizer token (HMAC từ session) cho mọi request thay đổi dữ liệu.
- Rate limit theo IP; giới hạn kích thước body; security headers + CSP dùng nonce; không open redirect.

## Kiểm tra code

```powershell
cd D:\AWS\WebXinNghiPhep\backend
go vet ./...
cd ..\frontend
npm run check                      # svelte-check (TypeScript strict)
npm run lint                       # prettier + eslint
```

## Backup / restore (restore vào DB riêng)

```powershell
$bin = 'C:\wamp64\bin\mysql\mysql8.4.7\bin'
# Backup (file dump đã nằm trong .gitignore)
& "$bin\mysqldump.exe" -h127.0.0.1 -uroot --single-transaction --routines --triggers employee_leave_management --result-file=backup.sql
# Restore vào một DB riêng đã tạo sẵn trong phpMyAdmin – KHÔNG restore đè DB chính
Get-Content backup.sql -Raw | & "$bin\mysql.exe" -h127.0.0.1 -uroot employee_leave_management_restore
```

## Ghi chú kỹ thuật

- Mọi timestamp lưu **UTC** (`DATETIME(3)`, session `time_zone='+00:00'`); lịch nghỉ tính theo `Asia/Ho_Chi_Minh`; thời lượng lưu bằng **phút**.
- Mỗi kết nối bật strict `sql_mode` vì sql_mode mặc định của Wamp đang trống.
- Các bảng audit (`*_events`, `leave_request_versions`, `policy_versions`) bị **trigger chặn UPDATE/DELETE**.
- Nâng một tài khoản lên `ADMIN` (kể cả sửa tay trong phpMyAdmin) sẽ **tự kích hoạt** tài khoản đó (trigger, migration 000009) và có hiệu lực ngay ở request kế tiếp, không cần đăng nhập lại.
- Mỗi ngày trong lịch chỉ có một quy tắc còn hiệu lực (cột `active_date` UNIQUE).
- Console `mysql.exe` có thể hiển thị tiếng Việt bị lỗi font; dữ liệu trong DB vẫn đúng UTF-8 (dùng phpMyAdmin để xem).
