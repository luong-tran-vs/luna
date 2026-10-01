# Research: Hỏi AI về từ (F9)

Mọi mục NEEDS CLARIFICATION đã được giải quyết. Mỗi mục dưới đây gồm: quyết định, lý do và các phương án đã cân nhắc.

## R1. `ai.Provider.Explain`

**Quyết định**

- Thêm `Explain(ctx, ExplainRequest{Text, Sentence, Level}) (Explanation{Lemma, MeaningVi, NoteVi}, error)`.
- `ai.Disabled` trả `ErrNotConfigured`.
- Gemini: 1 request, `temperature 0.2`, `responseSchema` là OBJECT với `lemma`, `meaningVi`, `noteVi` (đều bắt buộc).
- Prompt (tiếng Anh), gửi cho học viên Việt Nam trình độ CEFR `{level}`, gồm:
  - câu `{sentence}`;
  - cụm cần giải thích `{text}` (chép đúng như trong câu).
- Prompt yêu cầu AI trả về:
  - `lemma`: dạng từ điển;
  - `meaningVi`: nghĩa tiếng Việt ngắn, hợp đúng câu này;
  - `noteVi`: một câu giải thích tiếng Việt (cách dùng, vì sao nghĩa như vậy);
  - không markdown.
- Log một dòng `op=explain` ghi số từ của cụm. Không ghi câu, không ghi khoá.

**Lý do**

- Nguyên tắc III: mọi lời gọi AI đi qua interface.
- Schema cố định giúp kiểm tra kết quả dễ.

## R2. Kết quả dùng chung `ai_lookups` và khoá

**Quyết định**

- Collection `ai_lookups` gồm `{lessonId, revision, sentenceIndex, textLower, lemma, meaningVi, noteVi, createdAt}`.
- Index unique trên `(lessonId, revision, sentenceIndex, textLower)`.
- `textLower` = `normalize(text)`: chữ thường, gộp khoảng trắng. Đây chính là `normalize` của `Lookup` (F3).
- Lưu bằng insert. Gặp lỗi trùng khoá (hai máy chủ, hoặc lần lưu song song) thì coi như thành công: đọc lại bản đã có rồi trả về.
- Không có `userId`. Kết quả là nghĩa của từ trong câu, không phải dữ liệu người học.

**Lý do**

- Theo FR-006 và FR-009.
- `revision` đổi khi nội dung bài đổi (F2), nên kết quả cũ tự hết hiệu lực mà không cần dọn.

**Phương án đã loại**

- Lưu vào `lessons.annotations`: sẽ trộn với chú thích quản trị viên sửa tay, và mỗi lần hỏi lại phải ghi cả tài liệu bài.
- Lưu theo từng người học: trái khối 1 ("kể cả tài khoản khác").

## R3. Một lần gọi AI cho nhiều yêu cầu cùng lúc

**Quyết định**

- `Reader` có một `singleflight.Group` (`golang.org/x/sync`, đã có trong `go.mod`).
- Khoá của group là `lessonId|revision|sentenceIndex|textLower`.
- Hàm chạy trong group làm 3 việc theo thứ tự:
  1. Đọc `ai_lookups` lần nữa (có thể đã có từ lần gọi trước).
  2. Gọi `Explain` với `context.WithoutCancel(ctx)` và timeout 12 giây.
  3. Lưu kết quả.
- Mọi yêu cầu đang chờ nhận cùng kết quả hoặc cùng lỗi. Lỗi không được lưu, nên lần bấm sau sẽ thử lại.
- Không giữ `context` của request: người học đóng popup (request bị huỷ) thì AI vẫn trả lời và kết quả vẫn được lưu (edge case).

**Lý do**

- Theo FR-008 và SC-003.
- Timeout 12 giây giữ lời gọi dưới `WriteTimeout` 15 giây của server và dưới `proxy_read_timeout` 15 giây của nginx, nên không cần nới thời gian chờ như F7.

**Phương án đã loại**

- Khoá theo DB, ghi bản ghi "đang hỏi": thêm trạng thái và phải dọn bản ghi treo.
- Chỉ dựa vào unique index: vẫn gọi AI nhiều lần rồi mới va chạm khi lưu.

**Giới hạn**: singleflight chỉ trong một tiến trình backend. Dự án chạy một backend nên đáp ứng được.

## R4. Route `POST /api/lessons/{id}/ask` và kiểm tra đầu vào

**Quyết định**

- Body `{text, sentenceIndex}`. Đặt sau `requireAuth` và guard L như các route đọc bài.
- `Reader.Ask` kiểm tra:
  - `text` sau chuẩn hoá: không rỗng, ≤ 100 ký tự, ≤ 6 từ (giới hạn của F3);
  - `sentenceIndex` nằm trong các câu của bài;
  - `text` xuất hiện trong câu đó (không phân biệt hoa thường).

  Sai → 400 `validation_failed`, kèm `fields.text` hoặc `fields.sentenceIndex`.
- Có trong `ai_lookups` → 200 `{result, cached: true}`, không gọi AI.
- Chưa có → gọi AI qua singleflight → 200 `{result, cached: false}`.
- `result` có dạng giống `lookup` (F3), thêm `note`: `{source: "ai", text, lemma, ipa, meanings: [{pos: "", text: meaningVi}], note}`.
- `ipa` lấy từ từ điển theo `lemma` nếu có, như cách `Lookup` làm với chú thích.
- Lỗi:

| Lỗi | HTTP | Mã lỗi | Thông báo |
| --- | --- | --- | --- |
| `ErrNotConfigured` / `ErrInvalidKey` | 503 | `ai_not_configured` | "AI chưa được cấu hình" / "Khoá AI không hợp lệ" |
| `ErrQuota` | 429 | `ai_quota` | "Đã hết lượt AI, vui lòng thử lại sau" |
| Lỗi khác, quá thời gian, hoặc kết quả thiếu nghĩa (`ErrUnusableExplanation`) | 502 | `ai_failed` | "Không hỏi được AI, vui lòng thử lại" |

**Lý do**: theo khối 2, FR-010 và FR-011. Kiểm tra cụm phải có trong câu để người dùng không biến route này thành hỏi đáp tự do (nằm ngoài phạm vi).

## R5. `GET /lookup` (F3) dùng kết quả đã lưu

**Quyết định**

- Thứ tự tra trong `Reader.Lookup`:
  1. Chú thích của bài (như cũ).
  2. `ai_lookups` theo `(lesson, revision, sentence, textLower)`.
  3. Từ điển.
- Trường hợp 2 trả `source: "ai"` kèm `note`.
- `sentence < 0` (không rõ câu) thì bỏ qua bước 2.

**Lý do**: theo FR-007 ("hiện ngay kết quả đã lưu, không có nút Hỏi AI"). Không cần thêm request nào từ frontend.

## R6. Frontend

**Quyết định**

- `LookupResult` có thêm `note?: string`. `ReadingApiService.ask(lessonId, text, sentence): Observable<AskResult>`.
- `WordPopup` có thêm:
  - input `asking: boolean`, input `askError: string | null`, output `ask`;
  - nút **Hỏi AI** hiện khi `state.kind === 'not-found'` hoặc `result.source !== 'ai'`;
  - khi đang hỏi: nút khoá, chữ "Đang hỏi AI…", có `role="status"`;
  - `note` hiện dưới nghĩa, kèm nhãn "AI · theo ngữ cảnh" (nhãn có sẵn cho nguồn `ai`);
  - lỗi hiện dưới nút trong `role="alert"`;
  - nút ≥ 44px.
- `Reading` xử lý `ask`:
  - đặt `asking`, gọi API;
  - thành công thì `popupState = {kind: 'result', result}`. Nút Lưu lúc này dùng lemma và nghĩa AI, vì `save()` đã lấy từ `popupState`;
  - lỗi thì `askError` = `message` của server, hoặc "Không hỏi được AI, vui lòng thử lại" khi mất mạng;
  - bỏ qua kết quả nếu người học đã chọn từ khác (so sánh khoá của lần chọn);
  - đổi từ hoặc đóng popup thì xoá `askError`.

**Lý do**: thay đổi nhỏ nhất trên popup sẵn có. Lưu thẻ không phải sửa gì.

## R7. Kiểm thử

**Go**

- `Reader.Ask`:
  - có sẵn trong `ai_lookups` thì không gọi Provider;
  - chưa có thì gọi đúng 1 lần, lưu lại, lần sau `cached`;
  - 10 goroutine cùng khoá, Provider giả chặn tới khi được thả, chỉ 1 lần gọi;
  - revision mới không dùng kết quả cũ;
  - đầu vào sai (không có trong câu, câu ngoài phạm vi, quá dài);
  - lỗi AI không lưu gì;
  - kết quả thiếu nghĩa → lỗi;
  - lemma rỗng → dùng chính cụm đang tra.
- `Lookup` trả kết quả AI đã lưu (source ai, có note) trước từ điển.
- Handler: 200 / 400 / 404 / 503 / 429 / 502, và 401.
- Gemini httptest: schema, prompt, 429.
- Mongo: chuyển đổi doc; insert trùng khoá trả bản đã có (kiểm bằng tay trên Docker).

**Vitest**

- Popup:
  - nút hiện hoặc ẩn theo nguồn, và hiện ở not-found;
  - đang hỏi: khoá nút, có trạng thái;
  - hiện `note`;
  - lỗi hiện dưới nút.
- Reading:
  - ask → hiện kết quả AI → Lưu gửi lemma và nghĩa AI kèm `source: 'ai'`;
  - lỗi giữ nghĩa từ điển;
  - kết quả của từ cũ không đè lên từ mới.
