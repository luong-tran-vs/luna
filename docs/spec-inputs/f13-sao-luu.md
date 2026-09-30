# Đầu vào cho F13: Sao lưu và xuất dữ liệu

Nguồn: [giai-doan-1.md › F13](../phases/giai-doan-1.md).

## Khối 1: `/speckit-specify`

```text
F13 – Sao lưu và xuất dữ liệu (Giai đoạn 1) cho Luna.

Mục tiêu: không mất dữ liệu học tập tích luỹ qua nhiều tháng.

Kịch bản:
- Hệ thống tự sao lưu toàn bộ cơ sở dữ liệu mỗi ngày lúc 3h sáng, giữ 7 bản gần nhất; bản thứ 8 tự xoá bản cũ nhất.
- Người vận hành khôi phục từ một bản sao lưu bằng một lệnh, có hướng dẫn trong README.
- Người học bấm "Xuất dữ liệu" trong Cài đặt → tải về một file JSON gồm sổ từ, lịch sử ôn, tiến độ, cài đặt và các bài học của mình.

Quy tắc:
- File xuất chỉ chứa dữ liệu của tài khoản đang đăng nhập, không có mật khẩu hay phiên đăng nhập.
- Sao lưu lỗi thì ghi log rõ ràng; không ảnh hưởng app đang chạy.

Ngoài phạm vi: nhập lại dữ liệu từ file JSON, sao lưu lên cloud.

Tiêu chí nghiệm thu: theo mục F13 trong docs/phases/giai-doan-1.md.
```

## Khối 2: `/speckit-plan`

```text
Sao lưu (deploy/backup)
- Service backup trong docker-compose dùng image mongo:8 (có sẵn mongodump, mongorestore), chạy script backup.sh: vòng lặp chờ tới 03:00 theo biến TZ, chạy mongodump --archive --gzip vào volume ./backups/luna-YYYYMMDD.archive.gz, sau đó giữ 7 file mới nhất.
- restore.sh <file>: mongorestore --archive --gzip --drop; hướng dẫn trong README.
- Audio và từ điển không sao lưu (tạo lại được).
- Kiểm tra thủ công: chạy backup ngay bằng biến BACKUP_NOW=1, khôi phục vào DB trống và so số document.

Xuất dữ liệu (internal/export)
- GET /api/export: gom cards, review_logs, lesson_progress, dictation_results, goals, study_days, settings và các lesson người học đã học; trả JSON với Content-Disposition attachment (luna-export-YYYYMMDD.json).
- Test: chỉ chứa dữ liệu của user hiện tại, không có passwordHash và sessions.

Frontend (features/settings)
- Nút "Xuất dữ liệu" trong trang Cài đặt.
```
