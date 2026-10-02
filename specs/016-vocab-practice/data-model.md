# Data Model: Trang chi tiết bài và luyện tập từ vựng (F17)

## `lessons` (mở rộng)

| Trường mới | Kiểu | Ghi chú |
| --- | --- | --- |
| practice | object \| null | phần luyện tập đã làm sạch (bên dưới); `null` khi chưa có |
| practiceStatus | string | `""` (chưa có) \| `running` \| `done` \| `failed` |
| practiceError | string | thông báo tiếng Việt khi `failed` |
| practiceVersion | int | tăng 1 mỗi lần lưu phần luyện tập; dùng cho đường dẫn audio |

`practice`:

| Trường | Kiểu | Ràng buộc (sau `CleanPractice`) |
| --- | --- | --- |
| objectiveVi | string | ≤ 200 ký tự, có thể rỗng |
| examples | [{lemma, sentence}] | lemma thuộc từ vựng của bài, không trùng; câu ≤ 200 ký tự, chứa nguyên từ |
| dialogue | {speakers: [2 string], turns: [{speaker: 0\|1, text, meaningVi}]} \| null | 4–10 lượt; `null` khi bị bỏ |
| grammarTipVi | string | ≤ 400 ký tự, có thể rỗng |
| translations | [{vi, en, distractors: [string]}] | ≤ 5 câu; `en` 2–15 ô, có ≥ 1 từ vựng; ≤ 4 từ nhiễu, không trùng ô đáp án |

Không có index mới. Không có `userId`; không thuộc file xuất dữ liệu của người học (F13).

### Chuyển trạng thái `practiceStatus`

```text
""  ──(chú thích lưu thành công)──────────────▶ running   (practice = null)
done/failed ──(chú thích chạy lại thành công)─▶ running   (practice = null)
done/failed ──(quản trị viên Tạo lại)─────────▶ running   (giữ practice cũ)
running ──(ProcessPractice lưu được)──────────▶ done      (practice mới, practiceVersion + 1)
running ──(AI lỗi hết lượt thử / nội dung hỏng / chưa cấu hình)──▶ failed (giữ practice cũ nếu có)
bất kỳ ──(sửa nội dung bài, revision + 1)─────▶ ""        (practice = null; rồi chuỗi chú thích → luyện tập)
```

Mọi lệnh ghi của job đều có điều kiện `{_id, revision}`; `SavePractice` thêm điều kiện `practiceVersion` = giá trị lúc job đọc bài.

## `jobs` (loại mới)

| type | revision | Việc |
| --- | --- | --- |
| `practice` | revision của bài | 1 request AI → `CleanPractice` → `SavePractice` → enqueue `practice_audio` |
| `practice_audio` | revision của bài | tạo file audio còn thiếu cho version hiện tại; lỗi chỉ retry, không đổi trạng thái bài |

## File audio

`{AUDIO_DIR}/{lessonId}/{revision}/practice/{version}/{kind}-{i}.mp3`, `kind ∈ {example, turn, answer}`, `i` = vị trí trong mảng đã làm
sạch (`answer` đọc `translations[i].en`). Xong một version thì xoá các version khác; revision cũ và bài bị xoá được dọn như F4.

## Kiểu Go mới

`ai`:

- `PracticeWord {Lemma, Text, MeaningVi string}`; `PracticeRequest {Level, Title string; Sentences []string; Words []PracticeWord}`.
- `Practice {ObjectiveVi string; Examples []Example; Dialogue Dialogue; GrammarTipVi string; Translations []Translation}`;
  `Example {Lemma, Sentence}`; `Dialogue {Speakers []string; Turns []Turn}`; `Turn {Speaker int; Text, MeaningVi string}`;
  `Translation {Vi, En string; Distractors []string}`.
- `Provider` thêm `Practice(ctx, PracticeRequest) (Practice, error)`.

`job`: `TypePractice = "practice"`, `TypePracticeAudio = "practice_audio"`.

`lesson`:

- `Practice` (cùng hình dạng `ai.Practice` nhưng `Dialogue *Dialogue`), `Lesson` thêm `Practice *Practice`, `PracticeStatus Status`,
  `PracticeError string`, `PracticeVersion int`. `StatusNone Status = ""`.
- Hàm thuần: `CleanPractice(ai.Practice, []ai.PracticeWord) (Practice, error)`, `containsWord(sentence string, w ai.PracticeWord) bool`,
  `BuildFill(*Dialogue, []ai.PracticeWord) *Fill`, `answerTiles(en string) []string`.
- `Fill {Turns []FillTurn; Blanks []Blank; WordBank []string}`; `FillTurn {Speaker int; MeaningVi string; TurnIndex int; Parts []Part}`;
  `Part {Text string; Blank *int}`; `Blank {Answer string}`.
- `PracticeView` (cho API người học): `Status`, `LessonNumber`, `ObjectiveVi`, `Examples` (+ `AudioURL`), `Dialogue` (+ `AudioURL` mỗi
  lượt), `Fill`, `GrammarTipVi`, `Translations` (`Vi`, `Answer []string`, `Tiles []string`, `AudioURL`).
- Lỗi mới: `ErrNoValidPractice`, `ErrPracticeRunning`, `ErrAnnotationNotDone`.
- `Repository` thêm `SavePractice(ctx, id string, revision, prevVersion int, p Practice) (bool, error)`; `SaveAnnotations` đặt thêm
  `practice = nil, practiceStatus = running, practiceError = ""`; `ReplaceContent` đặt thêm `practice = nil, practiceStatus = "",
  practiceError = ""`; `SetStatus` nhận `TypePractice`.
- Port `Topics` thêm `Position(ctx, topicID, lessonID string) (int, error)`.
- `Service` thêm `ProcessPractice`, `ProcessPracticeAudio`, `RegeneratePractice(ctx, id) (Lesson, error)`; `JobFailed` xử lý 2 loại mới.
- `NewReader` nhận thêm `audioDir`; `Reader` thêm `Practice(ctx, id) (PracticeView, error)`.

`storage/mongo`: `practiceDoc` và các doc con (`exampleDoc`, `dialogueDoc`, `turnDoc`, `translationDoc`) chuyển đổi trực tiếp với kiểu
`lesson`.

## Frontend

- `core/models/practice.ts`:
  - `PracticeStatus = 'none' | 'running' | 'done' | 'failed'`;
  - `PracticeView {status, lessonNumber, objectiveVi, examples[{lemma, sentence, audioUrl}], dialogue: {speakers, turns[{speaker,
    text, meaningVi, audioUrl}]} | null, fill: {turns[{speaker, meaningVi, turnIndex, parts[{text} | {blank}]}], blanks[{answer}],
    wordBank[]} | null, grammarTipVi, translations[{vi, answer[], tiles[], audioUrl}]}`;
  - `AdminPractice` (nội dung thô cho trang quản trị).
- `core/models/lesson.ts`: `Lesson` (quản trị) thêm `practice: AdminPractice | null`, `practiceStatus: PracticeStatus`,
  `practiceError: string`; `isRunning()` tính cả `practiceStatus === 'running'`.
- State trang (không lưu server): `step: 1..4`, `tab: 'lesson' | 'reading'`, `fillAnswers: (string | null)[]`, `fillChecked`,
  `translateIndex`, `translateTiles: number[][]` (chỉ số ô đã chọn mỗi câu), `translateResults: (boolean | null)[]`, `finished`, `seed`.
