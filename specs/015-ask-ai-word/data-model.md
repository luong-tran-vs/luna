# Data Model: Hỏi AI về từ (F9)

## `ai_lookups` (MỚI, dùng chung cho mọi người học)

| Trường | Kiểu | Ghi chú |
| --- | --- | --- |
| _id | ObjectID | |
| lessonId | ObjectID | bài |
| revision | int | revision của nội dung bài lúc hỏi |
| sentenceIndex | int | câu chứa cụm từ |
| textLower | string | cụm từ đã chuẩn hoá: chữ thường, gộp khoảng trắng, ≤ 100 ký tự, ≤ 6 từ |
| lemma | string | dạng gốc; AI trả rỗng thì dùng `textLower` |
| meaningVi | string | nghĩa theo câu, không rỗng, ≤ 200 ký tự |
| noteVi | string | câu giải thích, ≤ 300 ký tự |
| createdAt | date | |

- Index unique trên `(lessonId, revision, sentenceIndex, textLower)`.
- Không có `userId`; không thuộc file xuất dữ liệu của người học (F13).
- Kết quả của revision cũ được giữ lại nhưng không bao giờ được đọc nữa.

## Kiểu Go mới

`ai`:

- `ExplainRequest {Text, Sentence, Level string}`.
- `Explanation {Lemma, MeaningVi, NoteVi string}`, với JSON `lemma`, `meaningVi`, `noteVi`.
- `Provider` có thêm `Explain(ctx, ExplainRequest) (Explanation, error)`.

`lesson`:

- `AskKey {LessonID string; Revision, SentenceIndex int; Text string}`.
- `AskResult {AskKey; Lemma, MeaningVi, NoteVi string; CreatedAt time.Time}`.
- `AskRepository`:
  - `Get(ctx, AskKey) (AskResult, bool, error)`;
  - `Put(ctx, AskResult) (AskResult, error)`: trùng khoá thì trả bản đã có.
- `LookupResult` có thêm `Note string`.
- Lỗi mới: `ErrUnusableExplanation`.
- `Reader` có thêm phụ thuộc `asks AskRepository`, `ai ai.Provider`, `group singleflight.Group`. Thêm method `Ask(ctx, lessonID, text, sentence) (LookupResult, bool, error)`.

## Frontend

- `core/models/reading.ts`:
  - `LookupResult.note?: string`;
  - `AskResult {result: LookupResult; cached: boolean}`.
- `WordPopup` có thêm input `asking`, `askError` và output `ask`.
