# Data Model: Từ vựng theo chủ đề (F18)

## `topics` (mở rộng)

| Trường mới | Kiểu | Ghi chú |
| --- | --- | --- |
| words | [string] | ≤ 100 từ, mỗi từ ≤ 40 ký tự, `^[A-Za-z][A-Za-z '\-/]*$`, không trùng (chữ thường); giữ thứ tự và cách viết hoa |
| wordsSeeded | bool | `true` khi đã xét nạp dữ liệu ban đầu hoặc quản trị viên đã lưu danh sách; thiếu trường = `false` |

Không có index mới. Không đổi `lessonIds`, `nameKey`.

### Chuyển trạng thái `wordsSeeded`

```text
false ──(khởi động: tên khớp file dữ liệu)──▶ true, words = danh sách trong file
false ──(khởi động: tên không khớp)─────────▶ true, words = []
false/true ──(quản trị viên Lưu)────────────▶ true, words = danh sách đã lưu
true ──(khởi động)──────────────────────────▶ không đổi
```

Ghi khi khởi động có điều kiện `{_id, wordsSeeded: {$ne: true}}`.

## `lessons`

Không đổi. Độ phủ đọc `topicId`, `content`, `annotations.text`, `annotations.lemma`. Chú thích có thể có thêm mục do bổ sung bằng từ
điển (cùng hình dạng `Annotation`, `editedByAdmin=false`).

## Dữ liệu nhúng

`backend/internal/topic/seed/topic-words.json` (chép từ `docs/spec-inputs/f18-topic-words.json`, đã sửa "Intensive care unit (ICU)"
→ "Intensive care unit"): `{source, topics: [{name, words: [string]}]}`, 42 chủ đề, 1.257 từ.

## Kiểu Go mới

`wordmatch` (gói mới):

- `New(text string, lemmas map[string]string) *Text`; `(*Text).Find(term string) (span string, ok bool)`; `(*Text).Contains(term) bool`.

`topic`:

- `Topic` thêm `Words []string`, `WordsSeeded bool`. `Summary` thêm `WordCount`, `UsedWordCount int`.
- `WordUse {Text string; Used bool; LessonCount int}`; `LessonText {Content string; Lemmas map[string]string}`.
- `Seed {Topics []SeedTopic}`, `SeedTopic {Name string; Words []string}`; `SeedTarget {ID, Name string}`; `SeedWrite {ID string;
  Words []string; Matched bool}`.
- Hàm thuần: `CleanWords`, `SeedKey`, `LoadSeed() (Seed, error)` (đọc file nhúng), `PlanSeed`, `Coverage`, `PlanWords`.
- `Repository` thêm `SetWords(ctx, id string, words []string) (Topic, error)`.
- `Lessons` port thêm `Texts(ctx, topicIDs []string) (map[string][]LessonText, error)`.
- `Service` thêm `Words(ctx, id) ([]WordUse, error)`, `SetWords(ctx, id, words) ([]WordUse, error)`, `WordPlan(ctx, id, count,
  perLesson) ([][]string, error)`; `List` điền `WordCount`, `UsedWordCount`.

`ai`:

- `AnnotateRequest {Sentences []string; Level string; FocusWords []string}`; `Provider.Annotate(ctx, AnnotateRequest)`.
- `GenerateRequest` thêm `TargetWords [][]string`.

`lesson`:

- `TopicRef` thêm `Words []string`. `GenerateInput` thêm `TargetWords [][]string`. `Draft` thêm `TargetWords, MissingWords []string`.
- `Deps` thêm `Dict Dictionary`.
- Hàm `focusWords(l Lesson, words []string) []string`, `addMissedFocus(anns []Annotation, focus []string, sentences []string, dict)`.

`storage/mongo`: `topicDoc` thêm `words`, `wordsSeeded`; `Topics.SetWords`; `SeedTopicWords(ctx, db, seed, log)`; `Lessons.TopicTexts`.

## Frontend

- `core/models/topic.ts`: `Topic` thêm `wordCount: number`, `usedWordCount: number`; `TopicWord {text: string; used: boolean;
  lessonCount: number}`.
- `core/models/generate.ts`: `GenerateInput` thêm `targetWords: string[][]`; `GeneratedDraft` thêm `targetWords: string[]`,
  `missingWords: string[]`; `DEFAULT_TARGET_WORDS` theo trình độ (8/10/12), `MAX_TARGET_WORDS = 15`.
- `DraftState` (draft-list) thêm `targetWords`, `missingWords`.

## Sửa 2026-10-07: từ có trình độ (chủ đề dùng chung)

`topics.words` đổi từ `[string]` thành `[{text: string, level: string}]`; `level` là "A1"–"C2" hoặc "" (mọi trình độ). Đọc dữ liệu
cũ dạng chuỗi coi như `level: ""`. Tối đa 300 từ (FR-001, đổi từ 100 ngày 2026-10-02). Nạp từ file dữ liệu ban đầu: `level: ""`.

Khi gộp chủ đề cùng tên (`specs/007-topic-roadmaps/data-model.md` mục 7): mỗi từ của chủ đề cũ nhận `level` = trình độ của chủ đề
cũ đó (kể cả từ đã nạp từ file); trùng (chữ thường) thì giữ một, trình độ thấp nhất; `wordsSeeded` = OR của các chủ đề cũ.

```go
// internal/topic
type Word struct { Text, Level string }
type WordStat struct { Text, Level string; Used bool; LessonCount int }
// WordPlan(ctx, topicID, level string, count, perLesson int): chỉ xét từ có Level == "" hoặc Level ≤ level;
// chưa dùng trước (đúng level trước, rồi thấp hơn/trống), rồi LessonCount tăng dần, rồi thứ tự danh sách.
// SuggestWords(ctx, topicID, level string, count int): từ mới mang level.
```

```ts
export interface TopicWord { text: string; level: Level | ''; used: boolean; lessonCount: number }
export interface WordInput { text: string; level: Level | '' }
```
