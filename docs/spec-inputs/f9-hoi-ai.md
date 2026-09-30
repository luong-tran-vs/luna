# Đầu vào cho F9: Hỏi AI về từ

Nguồn: [giai-doan-2.md › F9](../phases/giai-doan-2.md). Sửa F3 (popup tra từ).

## Khối 1: `/speckit-specify`

```text
F9 – Hỏi AI về từ (Giai đoạn 2) cho Luna.

Mục tiêu: khi từ hoặc cụm chưa có nghĩa theo ngữ cảnh, người học hỏi AI ngay trong popup tra từ.

Kịch bản:
- Tra một từ hoặc cụm không có trong chú thích của bài (nghĩa từ điển hoặc "Chưa có nghĩa"): popup có nút "Hỏi AI".
- Bấm nút: popup hiện trạng thái đang hỏi, rồi hiện nghĩa tiếng Việt theo đúng câu đang đọc, dạng gốc, và một câu giải thích ngắn; nhãn "AI · theo ngữ cảnh".
- Lưu vào sổ từ dùng nghĩa AI vừa trả về.
- Tra lại cùng từ trong cùng câu của cùng bài (kể cả tài khoản khác): hiện ngay kết quả đã lưu, không gọi AI.

Quy tắc:
- Chỉ gọi AI khi người học bấm nút; mỗi lần bấm tối đa 1 request.
- AI lỗi hoặc hết lượt: thông báo rõ ràng, popup vẫn dùng được với nghĩa từ điển hoặc tự nhập.
- Kết quả đã lưu gắn với bài và câu; sửa nội dung bài thì kết quả cũ không còn dùng.

Ngoài phạm vi: hỏi đáp tự do với AI, giải thích cả câu.

Tiêu chí nghiệm thu: theo mục F9 trong docs/phases/giai-doan-2.md.
```

## Khối 2: `/speckit-plan`

```text
Backend (internal/lesson, internal/ai)
- ai.Provider thêm Explain(ctx, ExplainRequest{text, sentence, level}) (Explanation{lemma, meaningVi, noteVi}, error): 1 request Gemini, responseSchema.
- Collection ai_lookups {lessonId, revision, sentenceIndex, textLower, lemma, meaningVi, noteVi, createdAt}, unique index (lessonId, revision, sentenceIndex, textLower). Dùng chung cho mọi người học.
- Endpoint (RequireAuth): POST /api/lessons/{id}/ask {text, sentenceIndex}. Có trong ai_lookups → trả luôn (cached: true); không → gọi AI, lưu, trả về. Chống bấm liên tiếp: singleflight theo khoá.
- GET /api/lessons/{id}/lookup (F3) trả thêm kết quả ai_lookups nếu có, với nguồn "ai".
- Lỗi: ErrNotConfigured 503, ErrQuota 429, khác 502.
- Test: cache hit không gọi Provider, 2 request đồng thời chỉ 1 lần gọi, revision mới không dùng cache cũ, lỗi AI.

Frontend (shared/components/word-popup)
- Nút "Hỏi AI" khi nguồn không phải AI; trạng thái đang hỏi (vô hiệu nút); hiển thị kết quả và nhãn; lỗi hiện dưới nút.
- Test Vitest: hiện/ẩn nút theo nguồn, lưu thẻ dùng nghĩa AI, lỗi.
```
