# Đầu vào cho F11: Hội thoại nhập vai

Nguồn: [giai-doan-3.md › F11](../phases/giai-doan-3.md). Cần F10 (nhận dạng giọng nói). Không bắt buộc trong luồng học.

## Khối 1: `/speckit-specify`

```text
F11 – Hội thoại nhập vai (Giai đoạn 3) cho Luna. Tính năng không bắt buộc.

Mục tiêu: người học dùng từ vựng của bài trong một cuộc trò chuyện ngắn với AI, như ngoài đời.

Kịch bản:
- Ở một bài đã mở (bài hôm nay hoặc bài đã học), chọn "Luyện hội thoại".
- App giới thiệu tình huống gắn với bài (ví dụ: bạn gọi món ở quán cà phê, AI là nhân viên) và câu mở đầu của AI, kèm audio.
- Người học trả lời bằng giọng nói (ghi âm như bước Nói) hoặc gõ chữ; AI trả lời tiếp, có audio, đúng trình độ của bài và dùng lại từ vựng của bài.
- Người học bấm "Gợi ý" để thấy một câu trả lời mẫu.
- Sau tối đa 10 lượt, hoặc khi bấm Kết thúc: AI tóm tắt các lỗi thường gặp và gợi ý cách nói tốt hơn, bằng tiếng Việt.
- Xem lại các phiên hội thoại cũ của bài.

Quy tắc:
- Không bắt buộc để hoàn thành bài học; không ảnh hưởng tiến độ, streak hay mục tiêu.
- Mỗi lượt trò chuyện tốn tối đa 1 request AI; tóm tắt cuối phiên tốn 1 request.
- AI lỗi hoặc hết lượt giữa phiên: báo rõ, giữ nội dung đã trò chuyện, thử lại được.
- Câu trả lời của AI ngắn (1–3 câu), đúng trình độ CEFR của bài.
- Dùng tốt trên điện thoại 360px; nhận dạng giọng nói lỗi thì vẫn gõ chữ được.
- File xuất dữ liệu (F13) có lịch sử hội thoại.

Ngoài phạm vi: hội thoại tự do không gắn bài, gọi video, nhân vật có hình ảnh, chấm điểm hội thoại.

Tiêu chí nghiệm thu: theo mục F11 trong docs/phases/giai-doan-3.md.
```

## Khối 2: `/speckit-plan`

```text
Backend (internal/conversation mới, internal/ai, internal/stt, internal/tts)
- Model Conversation {id, userId, lessonId, scenario, turns [{role user|ai, text, createdAt}], status (open|ended), summaryVi, createdAt}; repository ở storage/mongo/conversations.go.
- ai.Provider thêm StartRoleplay(ctx, lessonInfo) (scenario, openingLine), Reply(ctx, lessonInfo, turns) (reply, hint), Summarize(ctx, lessonInfo, turns) (summaryVi). lessonInfo gồm level, tiêu đề, danh sách từ vựng của bài. Mỗi hàm 1 request Gemini, responseSchema.
- Endpoint (RequireAuth): POST /api/lessons/{id}/conversations, GET /api/lessons/{id}/conversations, GET /api/conversations/{id}, POST /api/conversations/{id}/turns {text} hoặc multipart audio (dùng internal/stt của F10), POST /api/conversations/{id}/end.
- Audio câu AI: sinh bằng Kokoro qua internal/tts, cache theo hash (như /api/tts/word).
- Giới hạn 10 lượt người học; quá → 409.
- Export: thêm conversations.
- Test: service với Provider/Transcriber giả (lượt, giới hạn 10, lỗi AI giữ lượt đã có, quyền).

Frontend (features/lesson/conversation)
- Trang chat: bong bóng AI (nút nghe) và người học, nút ghi âm dùng lại component ghi âm của F10, ô gõ chữ, nút Gợi ý, Kết thúc; màn tóm tắt.
- Nút "Luyện hội thoại" ở trang bài (bài hôm nay sau khi xong, và trang Bài học).
- Test Vitest: gửi lượt, lỗi giữ nội dung, giới hạn lượt.
```
