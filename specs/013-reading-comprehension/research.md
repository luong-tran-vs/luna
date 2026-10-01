# Research: Câu hỏi hiểu bài và ghi chú ngữ pháp (F15)

Không còn mục NEEDS CLARIFICATION. Mỗi mục ghi: quyết định, lý do, các phương án đã cân nhắc.

## R1. Một request AI cho chú thích từ, câu hỏi, ngữ pháp và đề viết

**Quyết định**

- `ai.Provider.Annotate(ctx, sentences, level)` đổi kiểu trả về thành `ai.LessonExtras`, gồm:
  - `Annotations []Annotation`
  - `Questions []Question`, với `Question {Prompt string; Options []string; AnswerIndex int; ExplanationVi string}`
  - `GrammarNote *GrammarNote`, với `GrammarNote {Title, BodyVi string; Examples []string}`
  - `WritingPrompt string`
- Gemini vẫn gửi **1** request. `responseSchema` đổi từ mảng sang một OBJECT có các key `annotations` (mảng như cũ),
  `questions`, `grammarNote` và `writingPrompt`.
  - Bắt buộc chỉ có `annotations`.
  - Các key còn lại không bắt buộc, để AI thiếu phần nào thì phần đó trống chứ không hỏng cả request.
- Prompt (tiếng Anh) yêu cầu thêm:
  - 3–5 câu hỏi hiểu bài bằng tiếng Anh, đúng trình độ. Mỗi câu có 4 lựa chọn khác nhau, đúng 1 đáp án (`answerIndex`
    tính từ 0) và lời giải thích ngắn bằng tiếng Việt.
  - Một điểm ngữ pháp nổi bật: tiêu đề tiếng Việt, giải thích tiếng Việt ≤ 120 từ, 1–3 ví dụ chép nguyên văn từ bài.
  - Một đề viết tiếng Anh ngắn liên quan tới bài, hợp trình độ.
  - Prompt đánh số câu như cũ để chú thích từ vẫn có `sentenceIndex`.
- `ai.Disabled.Annotate` vẫn trả `ErrNotConfigured`.
- Tăng `LimitReader` từ 1 MB lên 2 MB vì phản hồi dài hơn.

**Lý do**

- Đáp ứng FR-001 và SC-001 (1 request mỗi bài), đúng khối 2.
- Schema dạng OBJECT được Gemini hỗ trợ.

**Phương án đã loại**

- Gọi một request riêng cho phần mới: tốn gấp đôi hạn mức, trái tiêu chí nghiệm thu.
- Thêm method mới trên Provider: lúc nào cũng gọi cùng Annotate nên không cần tách.

## R2. Làm sạch phần mới (`CleanExtras`)

**Quyết định.** Thêm `lesson.CleanExtras(x ai.LessonExtras, content string) Extras`. Hàm này không bao giờ trả lỗi; phần nào
hỏng thì để trống phần đó.

- **Câu hỏi.** Trim mọi trường. Bỏ câu khi gặp một trong các trường hợp sau:
  - nội dung câu hỏi rỗng hoặc dài quá 300 ký tự;
  - số lựa chọn khác 4;
  - có lựa chọn rỗng hoặc dài quá 150 ký tự;
  - hai lựa chọn trùng nhau (so sánh không phân biệt hoa thường, đã gộp khoảng trắng);
  - `answerIndex` nằm ngoài 0–3;
  - lời giải thích rỗng hoặc dài quá 500 ký tự.

  Bỏ cả câu trùng nội dung câu hỏi. Giữ tối đa 5 câu.
- **Ngữ pháp.**
  - Trim. Chỉ giữ những ví dụ xuất hiện trong nội dung bài: so sánh không phân biệt hoa thường, đã gộp khoảng trắng, và bỏ
    dấu câu cuối của ví dụ trước khi tìm.
  - Giữ tối đa 3 ví dụ.
  - Bỏ cả ghi chú khi: tiêu đề rỗng hoặc dài quá 100 ký tự, phần giải thích rỗng hoặc dài quá 1500 ký tự, hoặc không còn ví
    dụ nào.
- **Đề viết.** Trim, rồi cắt còn 500 ký tự; rỗng thì để trống.
- `ProcessAnnotate` giữ nguyên quy tắc cũ: `CleanAnnotations` lỗi (không có chú thích từ nào hợp lệ) thì job lỗi như F2.
  Phần mới không bao giờ làm job lỗi (FR-004).

**Lý do.** Theo FR-002, FR-003, FR-004 và các giới hạn đủ cho giao diện ở 360px.

**Phương án đã loại**

- Thiếu câu hỏi thì coi chú thích là lỗi: trái FR-004.
- Gọi lại AI để bù câu thiếu: tốn thêm request.

## R3. Phiên bản bộ câu hỏi (`quizVersion`)

**Quyết định**

- Bài có thêm `quizVersion int`, tăng (`$inc`) mỗi khi bộ câu hỏi bị thay:
  - chú thích AI lưu xong;
  - quản trị viên lưu phần extras mà câu hỏi khác bộ cũ;
  - nội dung bài đổi (`ReplaceContent` xoá câu hỏi).
- Câu trả lời lưu kèm `quizVersion`. Chỉ câu trả lời có cùng `quizVersion` với bài mới được hiện lại và được tính "đã trả lời
  hết". Câu trả lời của phiên bản cũ vẫn tính vào thống kê.

**Lý do**

- Khối 2 gắn câu trả lời với `revision`. Nhưng `revision` chỉ đổi khi nội dung bài đổi, còn quản trị viên sửa câu hỏi hay
  chạy lại chú thích đều thay bộ câu hỏi mà `revision` vẫn giữ nguyên (FR-018, US4-5).
- Một bộ đếm riêng bao được mọi trường hợp đó, và vẫn đáp ứng ý của khối 2.

**Phương án đã loại**

- Dùng `revision`: sai khi chỉ câu hỏi đổi.
- Gắn id riêng cho từng câu hỏi: phức tạp hơn mà không thêm lợi ích.

## R4. Ghi câu trả lời: chỉ một lần, chống đua

**Quyết định**

- Route `POST /api/lessons/{id}/answers` với body `{version, questionIndex, choice}`, đặt sau `requireAuth` và guard L như các
  route đọc bài.
- `lesson.Reader.Answer(ctx, userID, lessonID, in)` xử lý như sau:
  - Bài không tồn tại → 404.
  - `version` khác `quizVersion` của bài → `ErrQuizChanged` (409 `quiz_changed`, thông báo "Câu hỏi vừa được cập nhật").
  - `questionIndex` hoặc `choice` ngoài phạm vi → 400.
  - Hợp lệ thì chèn vào collection `reading_answers` với `{userId, lessonId, quizVersion, questionIndex, choice, correct,
    answeredAt}`. Collection có index **unique** `(userId, lessonId, quizVersion, questionIndex)`.
  - Chèn bị trùng khoá → `ErrAlreadyAnswered` (409 `already_answered`). Body lỗi kèm `answer` là câu trả lời đã ghi, để tab
    còn lại hiện đúng kết quả.
  - Thành công → 200 `{answer: {questionIndex, choice, correct, answerIndex, explanationVi}, answered, total, correct}`.
- Port mới `lesson.AnswerRepository`, hiện thực ở `storage/mongo/reading_answers.go`, gồm các method:
  - `Insert`
  - `Get(user, lesson, version, index)`
  - `List(user, lesson, version)`
  - `Totals(user)`

**Lý do**

- Theo FR-012 và edge case "hai tab": index unique là cách duy nhất chống đua khi không có transaction.
- Đáp án và lời giải thích chỉ gửi về sau khi đã trả lời (FR-011).

**Phương án đã loại**

- Upsert (như dictation): cho phép sửa câu trả lời, trái FR-012.
- Lưu câu trả lời trong `lesson_progress`: lẫn với tiến độ ngày, khó tính thống kê.

## R5. GET bài cho người học: câu hỏi không lộ đáp án

**Quyết định**

- `Reader.View(ctx, id)` đổi thành `View(ctx, userID, id)`. `readingJSON` có thêm:
  - `quiz`: `null` khi bài không có câu hỏi. Ngược lại là `{version, questions: [{prompt, options}], answers: [answerJSON]}`,
    trong đó `answers` là câu trả lời của **người dùng hiện tại** ở `version` hiện tại. Đáp án và giải thích chỉ có trong
    `answers`.
  - `grammarNote`: `null` hoặc `{title, bodyVi, examples}`.
- Đề viết không có trong JSON gửi người học (FR-021).

**Lý do.** Một request là đủ để vẽ lại trạng thái quiz khi tải lại trang hoặc mở trên thiết bị khác (FR-014, FR-017).

**Phương án đã loại.** Một route riêng `GET /answers`: tốn thêm request mà không lợi gì.

## R6. Hoàn thành bước Đọc do server kiểm tra

**Quyết định**

- Giữ route `POST /api/today/steps/read/complete`.
- `progress.completeStep` thêm cổng kiểm tra cho `StepRead`, đặt cạnh cổng kiểm tra của `StepListen`. Port mới là
  `progress.ReadingQuiz` với method `Status(ctx, userID, lessonID) (questions, answered int, err error)`.
  - `questions > 0 && answered < questions` → `ErrReadIncomplete` (409 `read_incomplete`, "Hãy trả lời hết câu hỏi").
  - Bài không có câu hỏi (`questions == 0`) → hoàn thành như giai đoạn 1.
- Adapter `readingQuiz` ở `main.go` gọi `lesson.Reader.QuizStatus`.
- Frontend `Reading` (mode `study`) tự phát `completed` ngay khi câu cuối được trả lời. Nếu vào lại khi đã trả lời hết mà
  bước chưa xong (ví dụ mất mạng lúc gọi complete), trang hiện nút **Tiếp tục** để gọi lại.

**Lý do**

- Theo khối 2 ("server kiểm tra") và FR-015.
- Dùng lại thứ tự bước, tính idempotent và phần cập nhật `study_days` sẵn có của `CompleteStep`.

**Phương án đã loại.** Route trả lời tự hoàn thành bước: `lesson` sẽ phụ thuộc `progress`, đảo chiều phụ thuộc hiện có.

## R7. Thống kê tỷ lệ hiểu bài

**Quyết định**

- `progress.StatsView` có thêm `Reading ReadingTotals{Answered, Correct int}` và `ReadingRate *float64` (nil khi chưa trả
  lời câu nào).
- Port `progress.ReadingQuiz` có thêm `Totals(ctx, userID) (answered, correct int, err error)`. Mongo tính bằng aggregate
  `$match userId` → `$group` (giống `DictationResults.Totals`).
- JSON `/api/stats` có thêm `reading: {answered, correct, rate}`. Trang thống kê có thêm mục "Hiểu bài: 75% (12 câu)",
  hoặc "—" kèm lời nhắc.
- Màn hình chính (`/api/dashboard`) không đổi: khối 2 ghi "dashboard/stats", nhưng spec chỉ yêu cầu trang thống kê, và màn
  hình chính đã có nút sang trang thống kê.

**Lý do.** Theo FR-019 và SC-007; dùng lại mẫu của dictation.

## R8. Quản trị: sửa extras, chạy lại chú thích

**Quyết định**

- Route `PUT /api/admin/lessons/{id}/extras` nhận `{questions: [...], grammarNote: {...}|null, writingPrompt}` và chạy kiểm
  tra như R2.
  - Kiểm tra này **báo lỗi** theo từng ô (`questions.1.options.2`, `grammarNote.examples.0`...), không lặng lẽ bỏ như khi
    làm sạch kết quả AI.
  - Cho phép 0–5 câu hỏi.
  - Ví dụ ngữ pháp không có trong bài báo "Ví dụ phải có trong bài".
  - Chú thích đang chạy → 409 `annotation_running`.
- Lưu xong đặt `extrasEditedByAdmin = true`. Nếu bộ câu hỏi khác bộ cũ thì tăng `quizVersion`.
- `Retry(annotate)` cho chạy cả khi chú thích đã xong (`done` hoặc `failed`). Đang chạy → 409 `annotation_running`.
  `Retry(tts)` giữ nguyên quy tắc chỉ chạy lại khi lỗi.
- Frontend:
  - Nút **Chạy lại chú thích** hiện cả khi chú thích đã xong.
  - Nếu có chú thích từ `editedByAdmin` hoặc `extrasEditedByAdmin`, nút mở `lu-confirm-dialog` liệt kê những phần đã sửa
    tay sẽ bị thay (FR-006). Không có phần nào sửa tay thì chạy ngay.
- Kết quả AI lưu xong đặt lại `extrasEditedByAdmin = false`.

**Lý do.** Theo FR-005 đến FR-009. Cảnh báo đặt ở frontend là đủ vì chỉ quản trị viên gọi được route này.

**Phương án đã loại.** Server bắt gửi `confirm=true`: thêm tham số mà không bảo vệ thêm được gì, vì cùng một quản trị viên.

## R9. Giao diện người học

**Quyết định**

- Component `lu-comprehension-quiz` (`features/lesson/reading/comprehension-quiz`).
  - Hiện **tất cả** câu theo thứ tự. Câu đã trả lời hiện kết quả. Câu đang làm là câu chưa trả lời đầu tiên. Các câu sau bị
    ẩn (hiện "Câu 3/5").
  - Mỗi lựa chọn là một nút ≥ 44px. Khi đang gửi thì cả câu bị khoá; lỗi mạng thì mở khoá và báo lỗi.
  - Kết quả hiện:
    - "✓ Đúng" hoặc "✗ Sai" bằng chữ đậm;
    - lựa chọn của người học ghi "(bạn chọn)";
    - đáp án đúng ghi "Đáp án đúng";
    - lời giải thích.

    Kết quả nằm trong vùng `aria-live="polite"`. Màu chỉ để phụ.
  - Trả lời xong câu cuối thì hiện "Đúng x/y câu".
  - Component phát `allAnswered` (mỗi lần câu cuối được trả lời trong phiên). Ở `?review=1` vẫn trả lời được những câu chưa
    trả lời, nhưng không phát hoàn thành.
- Component `lu-grammar-note` dùng `<details open>` + `<summary>` gốc nên thu gọn được và dùng được bằng bàn phím. Tiêu đề
  là "Ngữ pháp: {title}". Ví dụ là danh sách, hiện nguyên văn câu lấy từ bài.
- `Reading` khi bài có quiz:
  - hiện mục Ngữ pháp (nếu có) và quiz sau nội dung bài;
  - ẩn nút "Đã đọc xong";
  - quiz phát `allAnswered` → `completed.emit()`.

  Bài không có quiz giữ nguyên như giai đoạn 1.
- Trang thống kê có thêm thẻ "Hiểu bài".

**Lý do.** Theo FR-010 đến FR-017, FR-013 và nguyên tắc IV. `<details>` gốc là phần tử tiếp cận được rẻ nhất.

## R10. Trang chi tiết bài (quản trị)

**Quyết định**

- Component `lu-lesson-extras` (`features/admin/lesson-extras`) dùng reactive form với `FormArray` câu hỏi. Mỗi câu có:
  - câu hỏi;
  - 4 ô lựa chọn;
  - radio "Đáp án đúng";
  - ô giải thích;
  - nút Xoá.

  Ngoài ra có:
  - nút "Thêm câu hỏi", khoá khi đã đủ 5 câu;
  - nhóm Ngữ pháp: tiêu đề, giải thích, 1–3 ô ví dụ (có nút thêm và xoá), ô "Không có ghi chú ngữ pháp";
  - ô Đề viết;
  - nút Lưu.
- Lỗi từ server hiện đúng dưới từng ô.
- Nhãn trạng thái "Đã sửa tay" lấy từ `extrasEditedByAdmin`.

**Lý do.** Trang chi tiết bài đã dài nên tách ra component riêng. Đây là một form duy nhất, ứng với route PUT duy nhất.

## R11. Xuất dữ liệu và index

**Quyết định**

- `export` có thêm `CollReadingAnswers = "reading_answers"`, trường `readingAnswers`; thêm vào `Allowed` và danh sách trong
  `service.go`.
- `EnsureIndexes` có thêm index unique `(userId, lessonId, quizVersion, questionIndex)` trên `reading_answers`.
- Không cần migration: các trường mới của `lessons` thiếu thì được hiểu là rỗng, `quizVersion` 0.

**Lý do.** Theo FR-020 (F13 xuất toàn bộ dữ liệu của người học) và R4.

## R12. Kiểm thử

- **Go**
  - `CleanExtras`: câu hỏi thiếu hoặc trùng lựa chọn, `answerIndex` 4, ví dụ không có trong bài, thiếu cả ghi chú, cắt còn
    5 câu.
  - `ProcessAnnotate`:
    - 1 lần gọi AI;
    - lưu extras;
    - extras hỏng thì vẫn lưu chú thích từ;
    - `quizVersion` tăng;
    - `extrasEditedByAdmin` được đặt lại.
  - `Retry(annotate)`: chạy cả khi đã `done`; báo lỗi khi đang `running`.
  - `UpdateExtras`: lỗi theo từng ô, `quizVersion` tăng khi câu hỏi đổi.
  - `Reader.View`: không lộ đáp án; chỉ trả câu trả lời của chính người học, đúng phiên bản.
  - `Reader.Answer`: đúng/sai, 409 trả lời lại (kèm câu trả lời đã ghi), 409 `quiz_changed`, 400 ngoài phạm vi, người khác
    trả lời không ảnh hưởng.
  - `progress`:
    - đọc có câu hỏi: chưa trả lời hết → 409, trả lời hết → xong;
    - đọc không có câu hỏi: như cũ;
    - stats có `reading`.
  - Gemini: schema OBJECT, prompt có các phần mới, phản hồi giải mã đúng.
  - Mongo: index unique (test trùng khoá với DB thật, nếu bộ test mongo có sẵn).
- **Vitest**
  - quiz: chọn → khoá; ✓/✗ có chữ; tiếp tục từ câu trả lời đã có; lỗi mạng mở khoá; 409 hiện câu trả lời đã ghi;
    phát `allAnswered`.
  - grammar note: thu gọn và mở ra.
  - reading: bài có quiz thì không có nút "Đã đọc xong"; bài không có quiz thì giữ nút; trả lời xong thì phát `completed`.
  - lesson-extras: thêm, xoá, đổi đáp án, lỗi ô.
  - lesson-detail: nút chạy lại khi đã xong, có cảnh báo khi có phần sửa tay.
  - stats: hiện tỷ lệ hiểu bài.
