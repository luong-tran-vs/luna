# Đầu vào cho F12: Cài đặt

Nguồn: [giai-doan-1.md › F12](../phases/giai-doan-1.md), [design-system.md › mục 2](../design-system.md).

## Khối 1: `/speckit-specify`

```text
F12 – Cài đặt (Giai đoạn 1) cho Luna.

Mục tiêu: người học chỉnh giao diện và nhịp học theo ý mình; cài đặt đi theo tài khoản.

Người học chỉnh được:
- Chế độ giao diện: Sáng hoặc Tối (khi chưa chọn thì theo thiết bị).
- Số thẻ ôn tối đa mỗi ngày: từ 5 đến 200, mặc định 30.
- Múi giờ dùng để tính ngày học: mặc định là múi giờ lúc đăng ký, chọn từ danh sách.

Quy tắc:
- Đổi chế độ giao diện có hiệu lực ngay, không tải lại trang.
- Cài đặt lưu theo tài khoản; đăng nhập trên thiết bị khác vẫn giữ nguyên.
- Trước khi đăng nhập, app dùng lựa chọn giao diện đã lưu trên trình duyệt (như F0).
- Đổi múi giờ hoặc giới hạn thẻ có hiệu lực từ lần mở "Bài hôm nay" tiếp theo; không làm mất tiến độ hôm nay.

Ngoài phạm vi: đổi email, mật khẩu (chưa có), xuất dữ liệu (F13).

Tiêu chí nghiệm thu: theo mục F12 trong docs/phases/giai-doan-1.md.
```

## Khối 2: `/speckit-plan`

```text
Backend (internal/settings)
- Lưu trong users: settings {theme: light|dark|system (system = chưa chọn), dailyReviewLimit, timezone}. Chuyển trường timezone đang có của F1 vào settings.
- GET /api/settings, PUT /api/settings (kiểm tra 5 ≤ limit ≤ 200, timezone hợp lệ bằng time.LoadLocation).
- internal/progress đọc dailyReviewLimit và timezone từ settings thay cho giá trị mặc định.
- Test: kiểm tra đầu vào, đổi múi giờ giữa ngày không làm mất tiến độ hôm nay.

Frontend (features/settings)
- Trang /settings: nhóm Giao diện (2 lựa chọn Sáng, Tối), Học tập (giới hạn thẻ), Múi giờ (danh sách từ Intl.supportedValuesOf('timeZone')).
- ThemeService: sau khi đăng nhập lấy theme từ server, ghi vào localStorage; đổi theme thì lưu cả hai nơi.
```
