# Seed ngày lễ Việt Nam

Lệnh: `go run ./cmd/seed holidays [-years 2026,2027]` (thư mục `backend`).

## Nguyên tắc

1. **Mọi ngày seed đều là `DRAFT`.** Seed chỉ là đề xuất; Admin sửa, xóa hoặc xác nhận trong `/admin/holidays`.
2. **Chỉ insert, không ghi đè.** Mỗi ngày có `seed_key` (ví dụ `VN:2026:TET_1`). Khi chạy lại, seed bỏ qua:
   - ngày đã seed trước đó,
   - ngày Admin đã sửa (`admin_modified`) hoặc đã xóa (xóa mềm – không tạo lại),
   - ngày đã có một quy tắc khác (ví dụ Admin đã đặt ngày làm bù),
   - **toàn bộ năm** nếu năm đó đã `CONFIRMED`.
3. Không tự sao chép lịch năm trước sang năm sau; mỗi năm được tính riêng.
4. Mỗi lần insert ghi một sự kiện `SEEDED` vào `calendar_day_events` (append-only).
5. Ngày nghỉ bù được tính theo **ngày làm việc trong tuần của policy hiện hành** (mặc định Thứ 2–6).

## Danh sách đề xuất

| seed_key | Ngày | Loại | Cách tính |
|---|---|---|---|
| `NEW_YEAR` | 01/01 | NATIONAL | Cố định |
| `TET_EVE` | 30 (hoặc 29) tháng Chạp | NATIONAL | Âm lịch → ngày trước mùng 1 |
| `TET_1`…`TET_4` | Mùng 1–4 Tết | NATIONAL | Âm lịch. Đề xuất 5 ngày = giao thừa + mùng 1–4; công ty có thể chọn cách chia khác |
| `HUNG_KINGS` | 10/3 âm lịch | NATIONAL | Âm lịch |
| `REUNIFICATION` | 30/04 | NATIONAL | Cố định |
| `LABOUR_DAY` | 01/05 | NATIONAL | Cố định |
| `NATIONAL_DAY` | 02/09 | NATIONAL | Cố định |
| `NATIONAL_DAY_ADJ` | 01/09 hoặc 03/09 | NATIONAL | **Đề xuất** – ưu tiên ngày nối với cuối tuần; Chính phủ công bố chính thức hằng năm |
| `CHRISTMAS` | 25/12 | COMPANY | Tùy chọn (phổ biến ở công ty FDI) – xóa nếu không áp dụng |
| `COMP:<key>` | Ngày làm việc kế tiếp | COMPENSATORY | BLLĐ 2019 Điều 111(3): lễ trùng ngày nghỉ hằng tuần được nghỉ bù |

Không seed: 8/3, 20/10 (thường chỉ nửa ngày, chỉ áp dụng nhân viên nữ), 24/11 (chưa xác minh được là ngày nghỉ hưởng lương chính thức). Admin có thể tự thêm, kể cả ngày nghỉ nửa ngày của công ty.

## Chuyển đổi âm lịch

`internal/lunar` cài đặt thuật toán thiên văn của Hồ Ngọc Đức (dựa trên Jean Meeus) với **múi giờ UTC+7**. Test đối chiếu Tết 1985, 2007, 2019–2030 và Giỗ Tổ 2020–2025 (gồm năm nhuận 2020, 2023). Lịch Việt Nam khác lịch Trung Quốc ở một số năm vì múi giờ, ví dụ Tết 1985 (21/01), 2007 (17/02) và **2030 (02/02)**.

## Kết quả seed 2026–2027 (Thứ 2–6)

| Năm | Ngày | Ghi chú |
|---|---|---|
| 2026 | 01/01 (T5); 16–20/02 (T2–T6, Tết); 26/04 (CN, Giỗ Tổ) → **bù 27/04**; 30/04 (T5); 01/05 (T6); 01–02/09 (T3–T4); 25/12 (T6) | 13 ngày |
| 2027 | 01/01 (T6); 05–09/02 (T6–T3, Tết; mùng 1–2 rơi T7/CN) → **bù 10–11/02**; 16/04 (T6, Giỗ Tổ); 30/04 (T6); 01/05 (T7) → **bù 03/05**; 02–03/09 (T5–T6); 25/12 (T7) | 15 ngày |
