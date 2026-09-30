# Đầu vào cho L: Luồng một ngày học

Nguồn: [giai-doan-1.md › L](../phases/giai-doan-1.md), [mvp-features.md › mục 4](../mvp-features.md).

## Khối 1: `/speckit-specify`

```text
L – Luồng một ngày học (Giai đoạn 1) cho Luna.

Mục tiêu: mỗi ngày người học có đúng một bài mới, học theo thứ tự Ôn → Đọc → Nghe, và thấy mình tiến tới mục tiêu.

Kịch bản:
- Người học đặt mục tiêu: tự chọn trình độ (A1–C2), rồi chọn một chủ đề của trình độ đó (danh sách hiện số bài mỗi chủ đề). Mục tiêu là hoàn thành lộ trình của chủ đề, theo thứ tự quản trị viên xếp (F14).
- Chỉ học bài thuộc trình độ đang chọn; không xen kẽ trình độ.
- Đổi chủ đề hoặc trình độ bất cứ lúc nào: có hiệu lực ngay nếu bài hôm nay chưa bắt đầu, nếu đã bắt đầu thì từ hôm sau. Tiến độ mỗi lộ trình lưu riêng; quay lại chủ đề cũ thì học tiếp bài đang dở. Streak không bị ảnh hưởng.
- Mỗi ngày mở "Bài hôm nay": là bài tiếp theo chưa hoàn thành.
- Bước 1 Ôn: ôn các thẻ đến hạn, tối đa 30 thẻ/ngày; không có thẻ đến hạn thì bước tự hoàn thành.
- Bước 2 Đọc, bước 3 Nghe: dùng F3, F4. Bước sau chỉ mở khi xong bước trước.
- Hoàn thành bước cuối → bài hôm nay xong: mục tiêu +1 bài, chuỗi ngày học (streak) +1.
- Học xong bài hôm nay: bài tiếp theo chỉ mở vào ngày hôm sau; vẫn xem lại được bài cũ và ôn tự do.
- Thoát giữa chừng rồi vào lại: tiếp tục đúng bước và đúng câu đang làm.
- Hết lộ trình: chúc mừng và mời chọn chủ đề khác cùng trình độ hoặc lên trình độ tiếp theo.
- Trang "Bài học": bài hôm nay, các bài đã học (mở lại để đọc, nghe bất cứ lúc nào), các bài sắp tới ở trạng thái khoá (chỉ hiện tên). Mở lại bài đã học không làm thay đổi tiến độ, streak hay mục tiêu.

Quy tắc:
- Ngày mới bắt đầu lúc 0h theo múi giờ của người học.
- Nghỉ một hoặc nhiều ngày: hôm quay lại vẫn chỉ có một bài; streak về 0.
- Thẻ đến hạn vượt giới hạn ngày được chuyển sang hôm sau.
- Lộ trình chưa có bài tiếp theo: người học thấy "Chưa có bài mới", vẫn ôn được hoặc chọn chủ đề khác.

Ngoài phạm vi: bước Viết và Nói (giai đoạn 2, 3), màn hình chính đầy đủ (F6), chỉnh giới hạn thẻ và múi giờ (F12).

Tiêu chí nghiệm thu: theo mục L trong docs/phases/giai-doan-1.md.
```

## Khối 2: `/speckit-plan`

```text
Backend (internal/progress)
- Collection goals {userId, topicId, level, status (active|paused|completed), startedAt, completedAt}: mỗi user có tối đa một goal active; đổi chủ đề thì goal cũ chuyển paused, quay lại thì active lại. Danh sách bài của goal đọc từ lộ trình của topic (F14), không chụp cố định, để bài quản trị viên thêm sau vẫn vào lộ trình.
- Hiệu lực đổi chủ đề: nếu lesson_progress hôm nay đã có bước done thì goal mới có hiệu lực từ dayKey ngày mai (lưu effectiveFrom).
- Collection lesson_progress {userId, lessonId, dayKey (YYYY-MM-DD theo múi giờ), steps {review, read, listen: pending|done}, currentStep, sentenceIndex, completedAt}.
- Collection study_days {userId, dayKey, lessonId, reviewedCount, completed} để tính streak và giới hạn thẻ.
- Logic thuần trong internal/progress/rules.go (không truy cập DB): dayKey(now, tz), todayLesson(goal, roadmap, progress, now), effectiveGoal(goals, today), canStartNewLesson, streak(days, today), reviewQuota(limit, reviewedToday). Unit test phủ: qua nửa đêm, đổi múi giờ, nghỉ nhiều ngày, hết lộ trình, đổi chủ đề trước và sau khi bắt đầu bài hôm nay, quay lại chủ đề cũ.
- Giới hạn thẻ mặc định 30; đọc từ settings nếu có (F12 sẽ thêm), chưa có thì dùng mặc định.
- Endpoint: GET /api/goals (goal active + các goal paused), POST /api/goals {topicId}, GET /api/today (bài, các bước, bước hiện tại, số thẻ cần ôn), POST /api/today/steps/{step}/complete, PUT /api/today/position {step, sentenceIndex}, GET /api/today/review-cards (đã áp giới hạn).
- Server kiểm tra thứ tự bước: không cho hoàn thành bước khi bước trước chưa xong.
- Clock được tiêm để test.

Frontend (features/lesson)
- Trang /today: step-indicator (shared) + khung chứa bước hiện tại; dùng review-session (F5), reading (F3), listening (F4) và nối output completed vào POST complete.
- Lưu vị trí (bước, câu) khi đổi câu, debounce 1 giây.
- Trang chọn mục tiêu: bước 1 chọn trình độ, bước 2 chọn chủ đề (dùng GET /api/topics?level=, hiện tiến độ nếu đã học dở); màn chúc mừng khi hoàn thành lộ trình.
- Tạm thời trang chủ F0 có nút "Học hôm nay" dẫn tới /today; F6 sẽ thay trang chủ.
- Trang /lessons (danh sách): GET /api/lessons/mine trả {today, completed[], upcoming[] (chỉ id, title)}; bài đã học mở /lessons/:id/read và /listen ở chế độ xem lại (không gọi POST complete). Backend chặn mở bài upcoming (403). Thêm liên kết "Bài học" và "Sổ từ" vào header.
```
