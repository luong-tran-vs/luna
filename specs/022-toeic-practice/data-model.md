# Data Model: Luyện thi TOEIC (F27)

Tên trong code bằng tiếng Anh. Package backend mới: `internal/toeic`. Repository cài ở `internal/storage/{mongo,mysql}`
(MySQL: migration `047_toeic`, JSON lưu ở cột `TEXT NOT NULL`, id `CHAR(24)` từ `newID()`, `tableOptions`). Mongo: id là
ObjectID, `userId` là ObjectID.

## Hằng số

```go
type Part int // 1..7

var PartQuestions = [8]int{0, 6, 25, 39, 30, 30, 16, 54} // đề đầy đủ
type Section string // "listening" (Part 1–4) | "reading" (Part 5–7)

type Status string // "draft" | "published" | "hidden"
type TestKind string // "full" | "mini50" | "mini100"
type Mode string // "practice" | "test"
type SourceKind string // "test" | "random" | "part" | "tag" | "mistakes" | "retry"
type Accent string // "us" | "gb" | "au" | "ca"
```

Tag (dạng câu): danh sách cố định `Tags` trong `tags.go` (`Code`, `LabelVi`, `Parts []Part`, `GrammarPoints []string`), xem
research R6.

## Group — cụm câu (`toeic_groups`, admin)

| Trường | Kiểu | Ghi chú |
|---|---|---|
| ID | string | nội bộ |
| Key | string | khoá nghiệp vụ, duy nhất, `^[a-z0-9-]{3,48}$` |
| Part | Part | 1–7 |
| Level | int | mốc độ khó: 450, 600, 750, 900 |
| Status | Status | nhập mới = `draft` |
| Source | string | đường dẫn file trong `content/toeic/` (ghi chú) |
| Version | int | tăng mỗi lần nhập thay thế |
| Scene | string | Part 1: mô tả cảnh tiếng Anh cho ảnh |
| Script | []Turn | Part 2–4: lời nói; Part 1: không dùng (câu mô tả là `Options`) |
| Graphic | *Graphic | Part 3/4 khi có câu "Look at the graphic" |
| Docs | []Doc | Part 6/7: 1–3 văn bản |
| Questions | []Question | |
| CreatedAt, UpdatedAt | time | |

`Turn { Speaker string ("m","w","m2","w2"…); Accent Accent; Text string; TranslateVi string }`

`Graphic { Kind string ("table"|"schedule"|"price-list"|"floor-plan"|"bar-chart"); Title string; Columns []string;
Rows [][]string; Cells []GraphicCell /*floor-plan: {Label, Row, Col, Span}*/; Bars []Bar /*{Label, Value}*/ }`

`Doc { Kind string ("email"|"letter"|"memo"|"notice"|"ad"|"article"|"web"|"invoice"|"form"|"schedule"|"chat"|"review");
Title string; Meta []KV /*From, To, Subject, Date, URL…*/; Body []Block; ImageSlot string /*"doc-1" nếu có ảnh*/;
TranslateVi string }`

`Block { Kind string ("p"|"h"|"list"|"table"|"msg"); Text string /*cho phép **đậm** và chỗ trống [[n]] (Part 6, n = số
câu trong cụm)*/; Items []string; Table *Table; Who, Time string /*msg của chat*/ }`

`Question { No int /*1..n trong cụm*/; Prompt string /*Part 1: rỗng; Part 2: câu hỏi nói; Part 5: có "___"; Part 6: rỗng
hoặc "Which sentence…"*/; Options []string /*Part 2: 3, còn lại 4*/; Answer int /*0-based*/; Evidence []Ref; ExplainVi
string; TranslateVi string; Tags []string; Vocab []Vocab }`

`Ref { Doc int /*-1 = script*/; Block int; Turn int; Text string /*đoạn trích đúng nguyên văn để tô*/ }`
`Vocab { Word, IPA, MeaningVi string }`

Mã câu: `QuestionID = Key + "#" + strconv.Itoa(No)`.

### Kiểm tra khi nhập (`Validate`, trả `ValidationError{Fields}` với khoá dạng `groups[3].questions[1].answer`)

- `Key` hợp lệ, không trùng trong file; trùng trong DB → "trùng" (bỏ qua) hoặc thay thế khi `replace`.
- Theo Part:
  - P1: 1 câu, 4 phương án, `Scene` không rỗng, không `Docs`.
  - P2: 1 câu, `Prompt` không rỗng, 3 phương án.
  - P3: `Script` 2–3 người nói, có `Accent`, đúng 3 câu.
  - P4: `Script` 1 người nói, đúng 3 câu.
  - P5: 1 câu, `Prompt` có đúng một `___`, 4 phương án.
  - P6: 1 `Doc`, đúng 4 câu, đủ các chỗ trống `[[1]]`…`[[4]]` trong `Body`.
  - P7: 1–3 `Docs`, 2–5 câu.
- `Answer` trong khoảng; phương án không trùng nhau; `ExplainVi` không rỗng; `Tags` ≥ 1, đều có trong `Tags` và thuộc Part.
- `Evidence[].Text` (nếu có) phải xuất hiện nguyên văn trong khối/lượt nói được trỏ tới.
- Câu "Look at the graphic" (tag `graphic`) → cụm phải có `Graphic`.
- Thay thế cụm đang nằm trong đề **đã đăng**: số câu phải giữ nguyên.

Repository:

```go
type GroupRepository interface {
    List(ctx context.Context, f GroupFilter) ([]Group, error)          // admin: Part, Status
    ByKeys(ctx context.Context, keys []string) ([]Group, error)
    Published(ctx context.Context, part Part) ([]GroupMeta, error)     // không có Questions chi tiết: Key, Part, Level, Tags, số câu
    Upsert(ctx context.Context, gs []Group) (created, replaced []string, err error)
    SetStatus(ctx context.Context, key string, s Status) error
}
```

Chỉ số: `key` duy nhất; `(part, status)`.

## Image — ảnh trong đề (`toeic_images`, admin)

| Trường | Kiểu | Ghi chú |
|---|---|---|
| GroupKey | string | |
| Slot | string | `photo` (Part 1) hoặc `doc-<n>` |
| MIME | string | `image/jpeg` |
| Data | []byte | ≤ 800 px cạnh dài, chất lượng 70 |
| Origin | string | `gemini` \| `codex` \| `upload` |
| CreatedAt | time | |

Khoá duy nhất `(groupKey, slot)`. `ImageRepository { Get; Save (upsert); Delete; Has(keys []string) (map[string][]string) }`.

## Test — đề (`toeic_tests`, admin)

| Trường | Kiểu | Ghi chú |
|---|---|---|
| ID | string | |
| Key | string | duy nhất, ví dụ `luna-test-01`, `mini-05` |
| Title | string | "Luna Test 01" |
| Kind | TestKind | |
| Order | int | thứ tự trong thư viện; khách được làm Mini Test đã đăng có `Order` nhỏ nhất |
| Groups | []string | `groupKey` theo thứ tự Part rồi thứ tự trong Part |
| Status | Status | chỉ `published` khi `Check` đạt |
| CreatedAt, UpdatedAt | time | |

`Check(test, groups, images) TestCheck`: số câu đã có / cần cho mỗi Part (full: `PartQuestions`; mini: tỉ lệ R9 ±1 cụm), cụm
chưa đăng, ảnh Part 1 còn thiếu, số biểu đồ, số văn bản, thời lượng phần Nghe ước tính. `Ready` = đủ câu, mọi cụm đã đăng,
đủ ảnh.

## Attempt — bài làm (`toeic_attempts`, của người học)

| Trường | Kiểu | Ghi chú |
|---|---|---|
| ID | string | |
| UserID | string | |
| Mode | Mode | `practice` \| `test` |
| Source | Source | `{Kind, TestKey, Part, Tag, FromAttempt}` |
| Parts | []Part | các Part trong lượt |
| GroupKeys | []string | đề của lượt (cố định khi tạo); có chỉ số để chấm lại (R8) |
| QuestionIDs | []string | mọi câu theo thứ tự; dùng để biết "câu đã làm" |
| TimeLimitMs | int64 | 0 = không giới hạn |
| RemainingMs | int64 | chỉ giảm (R4) |
| Phase | Section | full test: `listening` rồi `reading` |
| ListeningSegment | int | đoạn kịch bản để phát tiếp |
| Position | string | mã câu đang xem |
| Answers | map[string]int | mã câu → phương án (−1 = bỏ chọn) |
| Flags | []string | câu đánh dấu xem lại |
| Status | string | `in_progress` \| `submitted` |
| Counted | bool | lần làm đầu của đề (hoặc đề ngẫu nhiên) có đủ 7 Part → tính vào biểu đồ (FR-004a) |
| Result | *Result | khi đã nộp |
| StartedAt, SavedAt, SubmittedAt | time | |

`Result { Correct, Wrong, Skipped int; ByQuestion map[string]string /*"right"|"wrong"|"skipped"*/; Listening, Reading
*SectionScore; Total int /*0 khi không đủ 7 Part*/; DurationMs int64; Version map[string]int /*version cụm lúc chấm*/ }`
`SectionScore { Correct, Total, Scaled int }`

Trạng thái: `in_progress` → (`submit` hoặc hết giờ) → `submitted`. Không quay lại. Mỗi người học tối đa **một** bài
`in_progress` cho mỗi `Source.TestKey` (đề cố định); tạo lượt mới cho đề đó khi đang có bài dở → trả bài dở (`409
attempt_in_progress` kèm id). Chấm lại (R8) chỉ đổi `Result`.

Repository:

```go
type AttemptRepository interface {
    Create(ctx context.Context, a Attempt) (Attempt, error)
    Get(ctx context.Context, userID, id string) (Attempt, error)            // ErrNotFound khi không phải của người đó
    Save(ctx context.Context, a Attempt) error                                // tiến độ (chỉ khi in_progress)
    Submit(ctx context.Context, a Attempt) error                              // ghi Result + Status
    List(ctx context.Context, userID string, f AttemptFilter) ([]Attempt, error) // theo TestKey / Counted / Status, mới nhất trước
    InProgress(ctx context.Context, userID, testKey string) (Attempt, error)
    AnsweredQuestions(ctx context.Context, userID string) (map[string]bool, error)
    WithGroup(ctx context.Context, groupKey string) ([]Attempt, error)        // bài đã nộp có cụm này (chấm lại)
    UpdateResult(ctx context.Context, id string, r Result) error
}
```

Chỉ số: `(userId, status, startedAt)`, `(userId, source.testKey)`, `groupKeys`.

## Mistake — mục sổ câu sai (`toeic_mistakes`, của người học)

| Trường | Kiểu | Ghi chú |
|---|---|---|
| UserID | string | |
| QuestionID | string | `key#n` |
| GroupKey | string | |
| Part | Part | lọc |
| Tags | []string | |
| Streak | int | 0–1; đạt 2 thì xoá mục |
| LastWrongAt | time | xếp "sai lâu nhất trước" khi làm lại |
| AddedAt | time | |

Khoá duy nhất `(userId, questionId)`. Quy tắc (hàm thuần `applyMistake(entry *Mistake, right bool) (keep bool)`):
- Bài nộp (không phải lượt sổ câu sai): sai/bỏ qua → thêm mục (`Streak=0`) hoặc đặt `Streak=0`; đúng → không đổi.
- Lượt làm lại trong sổ (chấm ngay từng câu): đúng → `Streak++`, `Streak == 2` thì xoá; sai → `Streak = 0`.

Repository `MistakeRepository { List(userID, part); Get(userID, qid); Upsert; Delete(userID, qid); Count(userID) }`.

## Goal — mục tiêu (`toeic_goals`, của người học)

| Trường | Kiểu | Ghi chú |
|---|---|---|
| UserID | string | khoá |
| Target | int | 450 \| 600 \| 750 \| 900 |
| ExamDate | string | `YYYY-MM-DD` hoặc rỗng |
| UpdatedAt | time | |

## Dữ liệu tính khi đọc (không lưu)

- `TagStat { Tag, Correct, Wrong, Skipped int }` từ `Result.ByQuestion` của mọi bài đã nộp + tag của câu.
- `ScorePoint { AttemptID, TestTitle, Total, Listening, Reading int; At time }` từ bài `Counted`.
- Thư viện: trạng thái mỗi đề = bài `in_progress` (câu đang làm) hoặc bài nộp gần nhất.

## Xuất dữ liệu và xoá tài khoản

- `export`: thêm `CollToeicAttempts = "toeic_attempts"`, `CollToeicMistakes = "toeic_mistakes"`, `CollToeicGoals =
  "toeic_goals"`, các trường `toeicAttempts`, `toeicMistakes`, `toeicGoals` trong `Export`, và trong `Allowed()`.
- Thêm 3 tên trên vào `userData` của `storage/mongo/users.go` và `storage/mysql/users.go`.
- `toeic_groups`, `toeic_images`, `toeic_tests` là nội dung chung, không xuất theo người học.

## Quyền

| Dữ liệu / thao tác | Khách | Thành viên | Admin |
|---|---|---|---|
| Thư viện đề, chi tiết đề | xem (cờ `locked`) | xem | xem |
| Mini Test đầu tiên; luyện Part 1, 2, 5 | làm | làm | làm |
| Đề khác, Part 3/4/6/7, đề ngẫu nhiên, theo tag, sổ câu sai, thống kê | `403 members_only` | làm | làm |
| Bài làm, sổ câu sai, mục tiêu | chỉ của mình | chỉ của mình | chỉ của mình |
| Nhập, đăng/ẩn, ghép đề, ảnh | — | — | có |
