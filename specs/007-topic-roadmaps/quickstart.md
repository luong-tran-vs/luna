# Quickstart: kiểm tra F14

## 0. Chuyển dữ liệu (US4) — làm trước tiên

1. Trước khi build bản F14, ghi lại dữ liệu F2 đang có:

   ```bash
   docker compose -f deploy/docker-compose.yml exec mongo mongosh luna --quiet --eval '
     printjson(db.lessons.find({}, {title:1, level:1, topic:1}).toArray());
     printjson(db.roadmap.findOne())'
   ```

   Nếu dữ liệu quá ít, tạo thêm ở bản cũ: một bài A1 chủ đề "Gia đình", một bài A1 chủ đề "gia đình ", một bài A2 không chủ
   đề, một bài B1 "Công việc"; lộ trình chung xen kẽ các bài.
2. `docker compose -f deploy/docker-compose.yml up --build -d`; log backend có "topics migration done".
3. Kiểm tra: `db.topics.find()` có đúng các chủ đề (gộp "Gia đình"/"gia đình ", có "A2 · Chung"); mọi bài có `topicId`, không
   còn `topic`; `lessonIds` của từng chủ đề theo thứ tự cũ; `db.roadmap` không còn; `db.migrations.findOne()` có `f14-topics`.
4. `docker compose -f deploy/docker-compose.yml restart backend` → dữ liệu không đổi (so lại bước 3).

## 1. Danh mục chủ đề (US1)

`/admin` → **Chủ đề**: thêm "Gia đình" A1, thêm "gia đình" A1 (bị chặn), "Gia đình" A2 (được); sửa mô tả; xoá chủ đề trống;
xoá chủ đề còn bài (bị chặn, nêu số bài).

## 2. Bài và bộ lọc (US2)

Thêm bài: ô chủ đề nhóm theo trình độ, không chọn → "Vui lòng chọn chủ đề". Sửa bài đang ở lộ trình sang chủ đề khác → bài ở
cuối lộ trình mới. Danh sách bài: lọc A1, rồi "Gia đình".

## 3. Lộ trình theo chủ đề (US3)

**Lộ trình** → chọn chủ đề → thêm 3 bài, kéo thả, tải lại giữ thứ tự; gỡ 1 bài → cảnh báo "còn 2 bài"; cảnh báo liệt kê các
chủ đề khác dưới 3 bài.

## 4. Người học

`curl -b "luna_session=<token người học>" "http://localhost:8000/api/topics?level=A1"` → danh sách chủ đề A1 và `lessonCount`.

## 5. 360px, sáng và tối

Trang Chủ đề, form bài, bộ lọc, Lộ trình: không cuộn ngang, dùng được bằng bàn phím.
