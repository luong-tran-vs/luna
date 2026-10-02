# Đầu vào cho F18: Từ vựng theo chủ đề

Nguồn: [giai-doan-3.md › F18](../phases/giai-doan-3.md). Dữ liệu ban đầu: [f18-topic-words.json](f18-topic-words.json), gồm 1.257 từ
tiếng Anh cho 42 chủ đề. Danh sách từ chỉ tham khảo từ trang Langmaster "3000 từ vựng tiếng Anh thông dụng theo chủ đề"; **chỉ lấy
từ tiếng Anh**, không chép phiên âm hay nghĩa. Sửa chủ đề (F14), sinh bài bằng AI (F7), chú thích (F2).

## Khối 1: `/speckit-specify`

```text
F18 – Từ vựng theo chủ đề cho Luna.

Mục tiêu: mỗi chủ đề có một danh sách từ vựng cốt lõi (ví dụ A1 · Gia đình: family, parents, father, mother…). Khi quản trị viên sinh
bài bằng AI, mỗi bài được giao một nhóm từ trong danh sách đó, ưu tiên từ chưa có ở bài nào của chủ đề. Nhờ vậy học hết lộ trình của
chủ đề là gặp đủ từ vựng của chủ đề, và trang chi tiết bài (F17) xoay quanh đúng các từ đó.

Danh sách từ của chủ đề:
- Mỗi chủ đề có danh sách từ tiếng Anh (từ đơn hoặc cụm như "sweet potato", "take a shower"); tối đa 300 từ (đổi từ 100 ngày 2026-10-02), mỗi từ tối đa 40 ký tự,
  chỉ chữ cái tiếng Anh, khoảng trắng, dấu gạch nối, dấu nháy đơn và dấu "/". Không trùng trong một chủ đề (không phân biệt hoa/thường).
- Chỉ lưu từ tiếng Anh; nghĩa và phiên âm vẫn đến từ chú thích bài (F2) và từ điển của app như hiện nay.
- Dữ liệu ban đầu: khi khởi động, chủ đề nào trùng tên với một chủ đề trong file dữ liệu và chưa từng được nạp thì nhận danh sách từ
  đó. Mỗi chủ đề chỉ được nạp một lần, không ghi đè danh sách quản trị viên đã sửa (kể cả khi quản trị viên xoá hết).

Trang Chủ đề (quản trị):
- Mỗi chủ đề hiện "Từ vựng: đã dùng X/Y". Một từ là "đã dùng" khi có trong nội dung của ít nhất một bài thuộc chủ đề (nguyên từ hoặc
  nguyên cụm, không phân biệt hoa/thường, tính cả dạng số nhiều/chia động từ đơn giản qua lemma của chú thích).
- Mở chủ đề để xem và sửa danh sách: thêm từ (gõ hoặc dán nhiều từ, mỗi dòng một từ hoặc cách nhau bằng dấu phẩy), xoá từ; từ đã dùng
  và chưa dùng được đánh dấu khác nhau (có chữ, không chỉ màu).

Sinh bài bằng AI (Lộ trình › Sinh bài bằng AI):
- Hộp thoại thêm mục "Từ mục tiêu mỗi bài": mặc định 8 (A1–A2), 10 (B1–B2), 12 (C1–C2); chọn 0 để sinh như hiện nay.
- Trước khi gửi, app chia từ cho từng bài: ưu tiên từ chưa dùng, rồi đến từ dùng ít nhất; các bài trong một lượt không trùng từ khi
  còn đủ từ. Hộp thoại hiện trước các nhóm từ này; quản trị viên bỏ hoặc thêm từ cho từng bài trước khi sinh.
- AI phải dùng các từ được giao trong bài của nó. Vẫn chỉ 1 request AI cho cả lượt.
- Bản nháp hiện "Dùng 7/8 từ mục tiêu" và liệt kê từ còn thiếu. Bản nháp không bị loại vì thiếu từ; quản trị viên tự sửa hoặc bỏ.
- Chủ đề chưa có từ vựng: sinh bài như hiện nay, không có mục từ mục tiêu.

Chú thích bài (F2):
- Khi chú thích bài thuộc một chủ đề có từ vựng, AI được gửi kèm các từ của chủ đề có trong bài và phải đưa các từ đó vào danh sách
  từ vựng của bài (cùng với từ khác như hiện nay). Vẫn 1 request AI cho mỗi lần chú thích.

Quy tắc:
- Không đổi luồng học, tiến độ, streak, thống kê. Người học không thấy danh sách từ của chủ đề (ngoài phạm vi).
- Xoá hoặc đổi từ trong danh sách không làm thay đổi bài đã có.
- AI lỗi khi sinh bài: báo lỗi như hiện nay, giữ nguyên các lựa chọn và nhóm từ để thử lại.
- Dùng được bằng bàn phím; dùng tốt ở 360px, sáng và tối.

Ngoài phạm vi: thẻ ôn tập theo chủ đề cho người học, người học xem danh sách từ của chủ đề, nhập phiên âm/nghĩa cho từ của chủ đề,
nhập/xuất danh sách từ bằng file, tự động sinh bài cho đủ từ.

Tiêu chí nghiệm thu: theo mục F18 trong docs/phases/giai-doan-3.md.
```

## Khối 2: `/speckit-plan`

```text
Backend (internal/topic, internal/lesson, internal/ai, internal/storage/mongo, cmd/api)
- topic.Topic thêm Words []string và WordsSeeded bool (Mongo: words, wordsSeeded). Chuẩn hoá từ: trim, gộp khoảng trắng; kiểm tra
  theo Quy tắc; trùng thì báo lỗi theo từ.
- Dữ liệu ban đầu: copy docs/spec-inputs/f18-topic-words.json vào internal/topic/seed/topic-words.json, nhúng bằng go:embed.
  Khi khởi động: với mỗi topic chưa wordsSeeded và có tên trùng (so tên đã chuẩn hoá, không phân biệt hoa/thường, "khoẻ" = "khỏe")
  → ghi words + wordsSeeded=true trong một update có điều kiện wordsSeeded != true. Chủ đề không khớp: đặt wordsSeeded=true, words
  rỗng. Log số chủ đề đã nạp.
- Độ phủ: topic.Service tính used/unused cho từng từ từ nội dung các bài của chủ đề (lesson port: nội dung + lemma chú thích theo
  topicId). Khớp nguyên từ/cụm không phân biệt hoa/thường; dạng biến đổi qua lemma của chú thích hoặc hậu tố đơn giản (s, es, ed, ing).
  Đếm số bài dùng mỗi từ (để chia từ "dùng ít nhất").
- API quản trị: GET /api/admin/topics trả thêm wordCount, usedWordCount; GET /api/admin/topics/{id}/words → {words: [{text, used,
  lessonCount}]}; PUT /api/admin/topics/{id}/words {words: []} (thay toàn bộ, đặt wordsSeeded=true);
  GET /api/admin/topics/{id}/word-plan?count=&perLesson= → [[từ…] mỗi bài] để hộp thoại hiện trước.
- Sinh bài: ai.GenerateRequest thêm TargetWords [][]string (mỗi phần tử cho một bài, có thể rỗng). Prompt Gemini: bài i phải dùng mọi
  từ trong TargetWords[i]. Request body POST /api/admin/topics/{id}/generate thêm targetWords [][]string (từ hộp thoại, đã kiểm tra
  thuộc danh sách chủ đề; số phần tử = count). Kết quả mỗi bản nháp thêm targetWords, missingWords (tính ở backend như độ phủ).
- Chú thích: ai.AnnotateRequest thêm FocusWords []string = từ của chủ đề có trong bài; prompt yêu cầu có mặt trong annotations.
  Không đổi giới hạn số từ hiện có; nếu vượt thì ưu tiên FocusWords.
- Test: chuẩn hoá và kiểm tra từ, nạp dữ liệu một lần (không ghi đè, chủ đề không khớp), độ phủ (cụm từ, hoa/thường, số nhiều),
  chia từ (ưu tiên chưa dùng, không trùng giữa các bài), handler (400 theo từ, 404), gemini prompt chứa TargetWords/FocusWords.

Frontend (features/admin)
- topics: cột/nhãn "Từ vựng: đã dùng X/Y"; trang hoặc panel sửa từ của chủ đề (danh sách có nhãn "Đã dùng"/"Chưa dùng", thêm nhiều
  từ một lần, xoá, Lưu; lỗi hiện theo từ).
- generate-dialog: trường "Từ mục tiêu mỗi bài" (0–15), gọi word-plan khi mở hoặc đổi số bài/số từ, hiện nhóm từ của từng bài với
  nút bỏ từ và ô thêm từ (gợi ý từ danh sách chủ đề); gửi targetWords.
- draft-list: dòng "Dùng a/b từ mục tiêu" và "Còn thiếu: …".
- admin-api.service + models: các API mới.
- Test Vitest: sửa danh sách từ (thêm nhiều, trùng, xoá, lỗi), hộp thoại sinh bài (nhóm từ, bỏ/thêm từ, 0 từ), bản nháp thiếu từ,
  bàn phím.
```
