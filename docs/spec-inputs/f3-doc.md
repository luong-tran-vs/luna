# Đầu vào cho F3: Đọc

Nguồn: [giai-doan-1.md › F3](../phases/giai-doan-1.md).

## Khối 1: `/speckit-specify`

```text
F3 – Đọc (Giai đoạn 1) cho Luna.

Mục tiêu: người học đọc bài, tra nghĩa ngay trong bài và lưu từ mới cùng câu ngữ cảnh.

Kịch bản:
- Người học mở bước Đọc của một bài: thấy toàn bộ bài, từ đã có trong sổ từ được tô màu.
- Chạm vào một từ → popup: từ, phiên âm IPA, nghĩa, nút nghe phát âm, nút Lưu vào sổ từ, nhãn nguồn nghĩa.
- Bôi đen (hoặc chạm giữ rồi kéo) nhiều từ liền nhau → tra cả cụm.
- Thứ tự lấy nghĩa: chú thích AI của bài ("AI · theo ngữ cảnh") → từ điển Anh–Việt offline ("Từ điển") → không có thì người học tự nhập nghĩa.
- Tra dạng biến đổi (went, studies) vẫn ra nghĩa của dạng gốc.
- Lưu từ: lưu kèm câu chứa từ đó; lưu lại từ đã có thì không tạo bản trùng.
- Cuộn tới cuối bài → nút Đã đọc xong bật; bấm để hoàn thành bước Đọc.

Quy tắc:
- Tra từ không gọi AI; kết quả hiện trong 300ms.
- Dùng tốt trên điện thoại 360px: popup không che mất từ đang tra, đóng được bằng chạm ra ngoài.

Ngoài phạm vi: "Hỏi AI" cho từ chưa chú thích (giai đoạn 2), ôn tập thẻ (F5), nối vào luồng một ngày học (L).

Tiêu chí nghiệm thu: theo mục F3 trong docs/phases/giai-doan-1.md.
```

## Khối 2: `/speckit-plan`

```text
Backend (internal/dictionary, internal/vocab, internal/lesson)
- Từ điển: file dictionary.db của minhqnd/dictionary (tải từ GitHub Releases vào volume /data/dictionary, có script tải trong deploy/). Đọc bằng modernc.org/sqlite (Go thuần, không CGO), mở chế độ chỉ đọc.
- Tra cứu: GET /api/lessons/{id}/lookup?q=...&sentence=N. Service thử theo thứ tự: annotations của bài (khớp text hoặc lemma, không phân biệt hoa thường) → từ điển (thử nguyên dạng, rồi dạng gốc theo quy tắc đơn giản -s/-es/-ed/-ing + bảng động từ bất quy tắc nhỏ) → 404. Trả {source: "ai"|"dictionary", text, lemma, ipa, meanings}.
- Phát âm từ: GET /api/tts/word?text=... sinh bằng Kokoro, cache file theo hash của text.
- Sổ từ tối thiểu (F5 sẽ mở rộng): collection cards {userId, text, lemma, ipa, meaningVi, contextSentence, lessonId, createdAt}, unique index (userId, lemma). POST /api/vocab/cards, GET /api/vocab/words (danh sách lemma để tô màu).
- Endpoint học (RequireAuth): GET /api/lessons/{id} trả câu, audio URL, không trả annotations thô.
- Test: thứ tự tra cứu, đưa về dạng gốc, lưu trùng.

Frontend (features/lesson/reading, shared/)
- Tách bài thành token từ ở client (shared/utils/tokenize.ts, có unit test); mỗi từ là phần tử bấm được.
- Chọn cụm: sự kiện selectionchange / pointer, gom các token liền nhau.
- shared/components/word-popup (dùng Angular CDK Overlay để tự định vị, không che từ).
- Nút Đã đọc xong: IntersectionObserver ở cuối bài. Component phát output completed; việc nối vào luồng ngày học làm ở L.
- Tạm thời mở được bước Đọc từ trang xem trước bài của admin để kiểm tra.
- Phiên âm IPA hiển thị bằng font Noto Sans (token --font-ipa).
```
