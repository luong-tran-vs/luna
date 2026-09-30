# Đầu vào cho F4: Nghe

Nguồn: [giai-doan-1.md › F4](../phases/giai-doan-1.md).

## Khối 1: `/speckit-specify`

```text
F4 – Nghe (Giai đoạn 1) cho Luna.

Mục tiêu: người học luyện nghe theo từng câu của bài và chép chính tả để biết mình nghe sai ở đâu.

Kịch bản:
- Mở bước Nghe: phát từng câu, có nút câu trước, câu sau, lặp lại câu.
- Chọn tốc độ 0.5x, 0.75x, 1x, 1.25x; ẩn hoặc hiện transcript.
- Chép chính tả: nghe một câu (bao nhiêu lần cũng được), gõ lại, bấm Kiểm tra.
- Kết quả so sánh từng từ: đúng; sai (gạch ngang, hiện từ đúng bên cạnh); thiếu (gạch chân chấm). Từ gõ thừa tính là sai.
- Kiểm tra hết các câu → bước Nghe hoàn thành (câu sai vẫn tính là xong).
- Tỷ lệ đúng của bài được lưu để thống kê.

Quy tắc:
- So sánh không phân biệt hoa thường, bỏ qua dấu câu; dấu nháy trong từ viết tắt (don't, it's) được giữ.
- Đúng, sai, thiếu phân biệt được không cần nhìn màu.
- Dùng tốt trên điện thoại 360px; bàn phím ảo không che nút Kiểm tra.

Ngoài phạm vi: nối vào luồng một ngày học (L), luyện nói.

Tiêu chí nghiệm thu: theo mục F4 trong docs/phases/giai-doan-1.md.
```

## Khối 2: `/speckit-plan`

```text
Frontend (features/lesson/listening, shared/)
- shared/components/audio-player: dùng HTMLAudioElement, playbackRate cho tốc độ, phát audio từng câu từ /api/audio/...
- shared/utils/dictation-compare.ts: chuẩn hoá (lowercase, bỏ dấu câu trừ dấu nháy trong từ), tách từ, căn chỉnh bằng khoảng cách chỉnh sửa theo từ (Levenshtein/LCS) → danh sách {word, status: ok|wrong|missing, expected?}. Unit test đầy đủ: trùng khớp, thiếu đầu/giữa/cuối, thừa, sai chính tả, dấu nháy, hoa thường.
- Kiểu hiển thị theo design-system: --color-ok, --color-bad + line-through, --color-warn + gạch chân chấm.
- Component phát output completed; nối vào luồng ở L.
- Trang /lessons/:id/listen; thêm nút "Mở bước Nghe" cạnh "Mở bước Đọc" ở trang chi tiết bài của admin để kiểm tra trước khi có L.

Backend (internal/progress)
- Collection dictation_results {userId, lessonId, sentenceIndex, correctWords, totalWords, createdAt}; lưu lần kiểm tra mới nhất mỗi câu.
- POST /api/lessons/{id}/dictation, GET /api/lessons/{id}/dictation/summary (tỷ lệ đúng của bài).
- Test handler: thành công, lỗi đầu vào, người khác không đọc được kết quả.
```
