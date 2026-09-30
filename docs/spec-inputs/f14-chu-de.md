# Đầu vào cho F14: Chủ đề và lộ trình theo trình độ

Nguồn: [giai-doan-1.md › F14](../phases/giai-doan-1.md), [mvp-features.md › R1, F14](../mvp-features.md).

Làm sau F5, trước L. Thay đổi phần quản trị đã có ở F2 (003-lesson-admin).

## Khối 1: `/speckit-specify`

```text
F14 – Chủ đề và lộ trình theo trình độ (Giai đoạn 1) cho Luna.

Mục tiêu: người học sẽ học theo chủ đề trong đúng trình độ của mình (phần người học làm ở L). Tính năng này chuẩn bị phía quản trị: danh mục chủ đề và một lộ trình riêng cho mỗi chủ đề.

Kịch bản (quản trị viên):
- Quản lý danh mục chủ đề: thêm, sửa, xoá. Mỗi chủ đề có tên, trình độ (A1–C2), mô tả ngắn. Ví dụ "A1 · Gia đình", "A1 · Mua sắm", "B1 · Công việc".
- Khi tạo hoặc sửa bài: chọn chủ đề từ danh sách (bắt buộc); trình độ của bài lấy theo chủ đề.
- Mỗi chủ đề có lộ trình riêng: thêm bài của chủ đề vào lộ trình, kéo thả đổi thứ tự, gỡ bài.
- Danh sách bài lọc theo trình độ và chủ đề (chọn từ danh sách, không gõ tự do).
- Trang lộ trình hiện cảnh báo cho từng chủ đề còn dưới 3 bài chưa học.
- Dữ liệu đang có được chuyển tự động: tạo chủ đề từ các cặp trình độ và chủ đề của bài hiện có; bài chưa có chủ đề vào chủ đề "Chung" cùng trình độ; lộ trình chung cũ chia theo chủ đề, giữ thứ tự tương đối.

Quy tắc:
- Lộ trình của một chủ đề chỉ chứa bài của chủ đề đó.
- Không xoá được chủ đề còn bài.
- Tên chủ đề không trùng trong cùng trình độ (không phân biệt hoa thường).
- Đổi chủ đề của một bài thì bài bị gỡ khỏi lộ trình cũ và thêm vào cuối lộ trình mới.
- Chỉ quản trị viên truy cập; dùng được ở 360px.

Ngoài phạm vi: người học chọn trình độ và chủ đề, bài hôm nay theo chủ đề (thuộc L); chủ đề nhiều cấp; ảnh đại diện cho chủ đề.

Tiêu chí nghiệm thu: theo mục F14 trong docs/phases/giai-doan-1.md.
```

## Khối 2: `/speckit-plan`

```text
Sửa code F2 hiện có (specs/003-lesson-admin): internal/lesson, internal/storage/mongo/{lessons,roadmap}.go, features/admin.

Backend
- Package mới internal/topic: model Topic {id, name, level, description, lessonIds (lộ trình, có thứ tự), createdAt}; service, handler, repository interface. Lộ trình lưu ngay trong document topic, bỏ collection roadmap cũ.
- Collection topics: unique index (level, nameLower).
- lessons: thay trường topic (chuỗi) bằng topicId; level vẫn lưu trên bài để lọc nhanh, luôn đồng bộ với level của topic.
- Endpoint (RequireAdmin): GET/POST /api/admin/topics, PUT/DELETE /api/admin/topics/{id}, GET/PUT /api/admin/topics/{id}/roadmap. Bỏ GET/PUT /api/admin/roadmap.
- Endpoint cho người học (RequireAuth, dùng ở L): GET /api/topics?level= trả {id, name, level, description, lessonCount}.
- Lọc bài: GET /api/admin/lessons?level=&topicId=.
- Migration một lần khi khởi động (internal/storage/mongo/migrate.go, đánh dấu đã chạy trong collection migrations): tạo topics từ distinct (level, topic), gán topicId cho bài, chuyển thứ tự lộ trình cũ vào từng topic, xoá collection roadmap. Chạy lại nhiều lần không sinh dữ liệu trùng.
- Test: service topic (trùng tên, xoá khi còn bài, đổi chủ đề bài chuyển lộ trình), migration (dữ liệu mẫu có bài không chủ đề, giữ thứ tự, chạy 2 lần).

Frontend (features/admin)
- Trang /admin/topics: danh sách chủ đề nhóm theo trình độ, form thêm/sửa, xoá có hộp xác nhận.
- Form bài: thay ô trình độ + ô chủ đề bằng một ô chọn chủ đề (nhóm theo trình độ).
- Trang lộ trình: chọn chủ đề rồi kéo thả (dùng lại component roadmap hiện có, thêm tham số topicId); cảnh báo theo từng chủ đề.
- Bộ lọc danh sách bài: hai ô chọn trình độ và chủ đề.
- Cập nhật test Vitest của lesson-form, lesson-list, roadmap.
```
