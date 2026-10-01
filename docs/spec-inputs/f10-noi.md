# Đầu vào cho F10: Nói (shadowing)

Nguồn: [giai-doan-3.md › F10](../phases/giai-doan-3.md). Cần F16 (HTTPS để dùng micro trên điện thoại). Sửa L (thêm bước), F6 (thanh Nói, thống kê), F13 (xuất dữ liệu).

## Khối 1: `/speckit-specify`

```text
F10 – Nói (shadowing) (Giai đoạn 3) cho Luna.

Mục tiêu: người học luyện nói bằng cách đọc theo từng câu của bài và biết ngay từ nào đọc chưa đúng.

Kịch bản:
- Bài hôm nay có thêm bước Nói (bắt buộc) sau bước Viết: Ôn → Đọc → Nghe → Viết → Nói.
- Với từng câu: nghe audio mẫu, bấm Ghi âm, đọc theo, bấm Dừng (tự dừng sau 30 giây).
- Sau vài giây: thấy chữ mà hệ thống nghe được và câu gốc được tô từng từ: đúng, sai (kèm từ nghe được), thiếu.
- Nghe lại bản ghi của mình; ghi lại câu đó bao nhiêu lần cũng được, kết quả lấy lần gần nhất.
- Ghi âm hết các câu → bước Nói hoàn thành (câu sai vẫn tính là xong), bài hôm nay xong; thấy tỷ lệ đọc đúng.

Quy tắc:
- Có kết quả trong vòng 5 giây với câu dưới 20 từ.
- So sánh không phân biệt hoa thường, bỏ qua dấu câu; số viết bằng chữ hay chữ số coi như nhau (nine = 9).
- Đúng, sai, thiếu phân biệt được không cần nhìn màu (cùng quy ước bước Nghe).
- Chưa cho quyền micro: hướng dẫn cấp quyền; không có micro: báo rõ.
- File ghi âm không lưu lâu dài, chỉ lưu kết quả so sánh.
- Dịch vụ nhận dạng giọng nói không chạy: báo lỗi rõ ràng, thử lại được; các bước khác không bị ảnh hưởng.
- Thanh kỹ năng Nói trên màn hình chính; thống kê thêm tỷ lệ đọc đúng; file xuất dữ liệu có kết quả bước Nói.
- Mở lại bài đã học: xem kết quả cũ và luyện lại được, không ảnh hưởng tiến độ.
- Dùng tốt trên điện thoại 360px.

Ngoài phạm vi: chấm phát âm tới từng âm vị, chấm ngữ điệu, lưu bản ghi để nghe lại sau.

Tiêu chí nghiệm thu: theo mục F10 trong docs/phases/giai-doan-3.md.
```

## Khối 2: `/speckit-plan`

```text
Deploy
- Service whisper trong docker-compose: whisper.cpp ở chế độ server (image chính thức của ggml-org nếu có bản server, không thì Dockerfile build từ source trong deploy/whisper/), model ggml small.en tải bằng script deploy/fetch-whisper-model.sh vào volume (giống fetch-dictionary), cổng chỉ mở cho localhost. Ghi trong research.md lựa chọn image cụ thể.
- Biến môi trường STT_URL, STT_MODEL (small.en, đổi base.en nếu chậm).

Backend (internal/stt mới, internal/speaking mới, internal/progress)
- internal/stt: interface Transcriber { Transcribe(ctx, audio io.Reader, contentType) (string, error) }; hiện thực gọi endpoint inference của whisper.cpp server; chuyển âm thanh trình duyệt (webm/opus, mp4/aac trên iOS) sang WAV 16 kHz mono bằng ffmpeg trong container backend (hoặc tuỳ chọn convert của whisper server, chọn trong research). Không gửi câu gốc làm prompt cho Whisper để không che lỗi.
- internal/speaking: POST /api/lessons/{id}/speaking/{sentenceIndex} (multipart audio, tối đa 2 MB, 30 giây) → {transcript, words: [{word, status, heard?}], correct, total}; so sánh bằng hàm Go dùng chung quy tắc với F4, thêm chuẩn hoá số 0–100 bằng chữ ↔ chữ số. Lưu speaking_results {userId, lessonId, revision, sentenceIndex, correctWords, totalWords, transcript, createdAt} (lần mới nhất mỗi câu). Xoá file tạm ngay sau khi nhận dạng.
- GET /api/lessons/{id}/speaking/summary.
- progress: thêm bước speak sau write; hoàn thành khi mọi câu có kết quả. Dashboard/stats: speak count, speakingAccuracy. Export: speakingResults.
- Lỗi: STT không chạy → 503 stt_unavailable; audio rỗng/hỏng → 400.
- Test: so sánh (số bằng chữ, dấu nháy, thiếu/thừa), handler với Transcriber giả (thành công, 503, file quá lớn, người khác 404), progress 5 bước.

Frontend (features/lesson/speaking)
- Dùng MediaRecorder (chọn mimeType hỗ trợ: audio/webm;codecs=opus hoặc audio/mp4), hiển thị đồng hồ và mức âm, tự dừng 30 giây; nghe lại bằng blob URL, giải phóng khi rời trang.
- Xử lý quyền: NotAllowedError → hướng dẫn cấp quyền theo trình duyệt; NotFoundError → "Không tìm thấy micro"; không phải HTTPS → hướng dẫn mở qua địa chỉ HTTPS (F16).
- Hiển thị kết quả dùng lại kiểu đúng/sai/thiếu của bước Nghe.
- /today thêm bước Nói; home thêm thanh kỹ năng Nói; stats thêm tỷ lệ đọc đúng.
- Test Vitest: trạng thái ghi âm (giả MediaRecorder), lỗi quyền, hiển thị kết quả.
- Kiểm tra thủ công trên điện thoại thật qua địa chỉ ts.net (Android Chrome, iOS Safari).
```
