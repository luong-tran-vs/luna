# Đầu vào cho F5: Sổ từ và ôn tập

Nguồn: [giai-doan-1.md › F5](../phases/giai-doan-1.md).

## Khối 1: `/speckit-specify`

```text
F5 – Sổ từ và ôn tập (Giai đoạn 1) cho Luna.

Mục tiêu: người học ôn từ đã lưu đúng lúc sắp quên để nhớ lâu.

Kịch bản:
- Xem danh sách thẻ trong sổ từ, tìm theo từ; sửa nghĩa, xoá, tự thêm thẻ mới.
- Mỗi thẻ: từ, IPA, nghĩa, câu ví dụ từ bài học, audio.
- Ôn tập với 2 kiểu:
  - Xem từ đoán nghĩa: hiện từ, chạm để lật xem nghĩa.
  - Nghe rồi gõ: nghe audio, gõ lại từ, hệ thống báo đúng hay sai.
- Sau mỗi thẻ, người học chọn Again, Hard, Good hoặc Easy; ngày ôn tiếp theo được tính lại.
- Có thể vào ôn tự do bất cứ lúc nào (các thẻ đến hạn).
- Sổ từ nhóm thẻ theo ngày lưu ("Hôm nay", "Hôm qua", ngày cụ thể) và lọc được theo bài học.
- Ở bước Đọc có mục "Từ vựng" liệt kê các từ và cụm từ đã được chú thích của bài (nghĩa, IPA, nút nghe). Mỗi từ có nút Lưu, cả mục có nút Lưu tất cả; từ đã có trong sổ hiện dấu ✓.

Quy tắc:
- Lưu tất cả chỉ thêm từ chưa có trong sổ, không tạo thẻ trùng.
- Mục Từ vựng không gọi AI; bài chưa có chú thích thì hiện "Chưa có danh sách từ vựng".
- Nhóm theo ngày tính theo múi giờ của người học.
- Lịch ôn dùng thuật toán FSRS.
- Thẻ mới lưu hôm nay đến hạn lần đầu vào hôm sau (theo múi giờ của người học).
- Kiểu Nghe rồi gõ không phân biệt hoa thường.
- Sửa nghĩa không làm mất lịch ôn; xoá thẻ thì xoá cả lịch sử ôn của thẻ.

Ngoài phạm vi: giới hạn số thẻ mỗi ngày và bước Ôn bắt buộc đầu bài (thuộc L).

Tiêu chí nghiệm thu: theo mục F5 trong docs/phases/giai-doan-1.md.
```

## Khối 2: `/speckit-plan`

```text
Backend (internal/vocab)
- Mở rộng collection cards (từ F3) với trạng thái FSRS: due, stability, difficulty, elapsedDays, scheduledDays, reps, lapses, state, lastReview. Thẻ cũ chưa có trạng thái được coi là thẻ mới.
- Dùng github.com/open-spaced-repetition/go-fsrs/v3 với tham số mặc định. Thẻ mới: due = 0h ngày hôm sau theo users.timezone.
- Collection review_logs {userId, cardId, rating, mode, reviewedAt, stateBefore}.
- Đồng hồ được tiêm vào service (interface Clock) để test theo ngày.
- Endpoint: GET /api/vocab/cards?q=&page=, POST, PATCH /api/vocab/cards/{id}, DELETE /api/vocab/cards/{id}, GET /api/vocab/review/due?limit=, POST /api/vocab/cards/{id}/review {rating, mode}.
- Audio thẻ: dùng /api/tts/word từ F3.
- Test: lịch FSRS sau từng mức đánh giá, thẻ mới đến hạn hôm sau theo múi giờ (có trường hợp gần nửa đêm), sửa nghĩa giữ lịch, xoá thẻ xoá log.

Frontend (features/vocabulary)
- Trang danh sách thẻ (tìm kiếm, phân trang), form thêm/sửa thẻ.
- shared/components/review-session: nhận danh sách thẻ, 2 chế độ, 4 nút đánh giá, phát output finished. Component này được dùng lại ở bước Ôn của L.
- Trang /vocabulary/review cho ôn tự do.
- Danh sách thẻ nhóm theo ngày (tính theo timezone) và bộ lọc theo bài; backend thêm GET /api/vocab/cards?lessonId=&groupBy=day.
- Mục Từ vựng của bài: component lesson-vocabulary đặt trong trang Đọc (F3) dưới dạng tab hoặc phần thu gọn; backend thêm GET /api/lessons/{id}/vocabulary (annotations của bài + cờ saved) và POST /api/vocab/cards/bulk (bỏ qua lemma đã có).
- Test: bulk không tạo trùng, nhóm theo ngày gần nửa đêm, bài chưa có chú thích.
```
