# Đầu vào cho F6: Màn hình chính và tiến độ

Nguồn: [giai-doan-1.md › F6](../phases/giai-doan-1.md), [mvp-features.md › mục 4.1](../mvp-features.md).

## Khối 1: `/speckit-specify`

```text
F6 – Màn hình chính và tiến độ (Giai đoạn 1) cho Luna.

Mục tiêu: mở app là biết ngay mình đang ở đâu so với mục tiêu và hôm nay cần làm gì.

Người học thấy trên màn hình chính:
- Thanh mục tiêu: tên lộ trình (ví dụ "A1 · Gia đình") và số bài đã xong trên tổng số bài của lộ trình; có nút đổi chủ đề.
- Thanh kỹ năng Nghe và Đọc: mỗi thanh đếm số bài đã xong bước của kỹ năng đó (Viết, Nói sẽ thêm ở giai đoạn sau).
- Bài hôm nay: tên bài, trình độ, chủ đề, thanh tiến trình các bước, nút "Tiếp tục: <bước>" đưa thẳng tới bước đang dở.
- Streak (số ngày học liên tiếp) và số thẻ đến hạn ngày mai.
- Trạng thái đặc biệt: chưa chọn chủ đề (nút chọn chủ đề), đã xong bài hôm nay (thông báo hẹn ngày mai + nút ôn tự do), chưa có bài mới.

Trang thống kê: số từ đã học, số câu đã chép chính tả, tỷ lệ đúng, số bài hoàn thành theo kỹ năng.

Quy tắc:
- Trên điện thoại 360px, thanh mục tiêu, bài hôm nay và nút Tiếp tục nằm trong màn hình đầu, không cần cuộn.
- Số liệu cập nhật ngay sau khi hoàn thành một bước.

Ngoài phạm vi: biểu đồ theo thời gian, bảng xếp hạng.

Tiêu chí nghiệm thu: theo mục F6 trong docs/phases/giai-doan-1.md.
```

## Khối 2: `/speckit-plan`

```text
Backend (internal/progress)
- GET /api/dashboard: gộp goal (tên chủ đề, trình độ, số bài đã xong / tổng số bài của lộ trình), số bài xong theo kỹ năng (từ lesson_progress.steps), bài hôm nay (từ logic L), streak, số thẻ đến hạn ngày mai (cards.due trong ngày mai theo múi giờ).
- GET /api/stats: số thẻ, số câu đã chép chính tả, tỷ lệ đúng trung bình, số bài hoàn thành theo kỹ năng.
- Dùng aggregation của MongoDB; thêm index cần thiết (userId + due, userId + completedAt).
- Test: dashboard ở các trạng thái (chưa có mục tiêu, đang học, xong hôm nay, hết bài).

Frontend (features/home)
- Thay trang chủ tạm của F0. Bố cục theo mockup mục 4.1 của mvp-features.md và token design-system (thanh mục tiêu dùng --color-primary, thanh kỹ năng dùng --color-skill-*).
- shared/components/progress-bar và step-indicator (dùng lại ở /today).
- Cập nhật số liệu: gọi lại /api/dashboard khi quay về trang chủ.
- Trạng thái kết nối (F0) chuyển xuống chân trang, chỉ hiện khi có lỗi.
- Trang /stats.
- Kiểm tra thủ công: 360px không cần cuộn để thấy nút Tiếp tục, cả hai chế độ.
```
