# Data Model: Trò chơi (F26)

Tên trong code bằng tiếng Anh. Package backend mới: `internal/games`. Kiểu dưới đây là kiểu Go của package (repository
nằm ở `internal/storage/{mongo,mysql}`).

## Mã trò

```go
type Game string

const (
    GamePicture Game = "picture" // Ảnh nhanh tay (P1)
    GameVerbs   Game = "verbs"   // Đấu động từ (P2)
    GameLetters Game = "letters" // Chữ rơi (P3)
    GameMemory  Game = "memory"  // Lật thẻ ghi nhớ (P4)
    GameGrammar Game = "grammar" // Bắn lỗi ngữ pháp (P5), chỉ thành viên
    GameDaily   Game = "daily"   // Thử thách hằng ngày (P6)
)
```

`MaxScore` mỗi trò (server từ chối điểm lớn hơn, `400`): picture 30 000, verbs 500, letters 20 000, memory 10 000,
grammar 5 000, daily 3 000.

## Best — kỷ lục (`game_bests`)

| Trường | Kiểu | Ghi chú |
|---|---|---|
| UserID | string | |
| Game | Game | không có `daily` (kỷ lục thử thách tính từ `daily_challenges`) |
| Score | int | điểm cao nhất |
| At | time | lúc lập kỷ lục |

- Khoá duy nhất `(userId, game)`. Chỉ ghi khi lượt `completed` và `Score` lớn hơn kỷ lục cũ (thao tác "lấy lớn hơn":
  Mongo `$max` + upsert; MySQL `INSERT … ON DUPLICATE KEY UPDATE score = GREATEST(...)`, `at` chỉ đổi khi điểm tăng).
- Repository:

```go
type BestRepository interface {
    // List returns every best of the user.
    List(ctx context.Context, userID string) ([]Best, error)
    // Raise stores score when it beats the user's best of game; improved reports whether it did.
    Raise(ctx context.Context, userID string, game Game, score int, at time.Time) (improved bool, err error)
}
```

## DailyResult — kết quả thử thách (`daily_challenges`)

| Trường | Kiểu | Ghi chú |
|---|---|---|
| UserID | string | |
| Day | string | `YYYY-MM-DD` theo múi giờ người học |
| Score | int | 0…3 000 |
| Answers | []DailyAnswer (10) | thứ tự câu |
| FinishedAt | time | |

`DailyAnswer { Kind string ("picture"|"verb"|"letters"|"grammar"); Right bool }`.

- Khoá duy nhất `(userId, day)`. `Save` trả `ErrExists` khi ngày đó đã có (chỉ lượt đầu được tính, FR-028).
- Repository:

```go
type DailyRepository interface {
    // Get returns ErrNotFound when the user has no result on day.
    Get(ctx context.Context, userID, day string) (DailyResult, error)
    // Save stores the first result of a day; ErrExists when the day already has one.
    Save(ctx context.Context, r DailyResult) error
    // List returns the user's results, newest day first (for the streak, the week and the comparisons).
    List(ctx context.Context, userID string) ([]DailyResult, error)
}
```

- **Tính khi đọc (không lưu)**: `Streak{Current, Longest int}`, `Week [7]bool` (Thứ Hai → Chủ Nhật của tuần hiện tại),
  `Compare{Yesterday *int, Avg7 *float64, Best int}`. Hàm thuần `streakOf(days []string, today string) Streak`.

## SpotItem — câu Bắn lỗi soạn thêm (`grammar_spot_items`)

| Trường | Kiểu | Ghi chú |
|---|---|---|
| ID | string | |
| PointID | string | điểm trong syllabus ngữ pháp |
| Level | string | trình độ của điểm (A1…) |
| Sentence | string | câu **có lỗi**, tiếng Anh |
| Wrong | string | chỗ sai, nằm đúng một lần trong `Sentence` |
| Right | string | chữ đúng thay cho `Wrong` |
| Distractors | []string (2) | khác nhau, khác `Right` |
| RuleVi | string | quy tắc ngắn tiếng Việt |
| Status | `draft` \| `approved` | chỉ `approved` vào trò chơi |
| CreatedAt, UpdatedAt | time | |

- Chỉ mục `(pointId, status)`. Admin sửa (`Sentence`, `Wrong`, `Right`, `Distractors`, `RuleVi`) thì kiểm tra lại như
  khi nhận từ AI (`ValidateSpot`), sửa không đổi `Status`.
- Repository:

```go
type SpotRepository interface {
    ListByPoint(ctx context.Context, pointID string) ([]SpotItem, error)
    // Approved returns the approved items of the points (all points when pointIDs is nil).
    Approved(ctx context.Context, pointIDs []string) ([]SpotItem, error)
    // Counts returns the approved and draft items per point.
    Counts(ctx context.Context) (map[string]SpotCount, error)
    Create(ctx context.Context, items []SpotItem) ([]SpotItem, error)
    Get(ctx context.Context, id string) (SpotItem, error)
    Update(ctx context.Context, it SpotItem) error
    Delete(ctx context.Context, id string) error
}
```

## Kiểu chỉ có trong một lượt (không lưu)

```go
// PictureQuestion: one picture and four options; Answer is the index of the right one.
type PictureQuestion struct { Lemma, ImageURL, MeaningVi string; Options []string; Answer int }

// LetterWord: a word to spell from falling (or shuffled) letters.
type LetterWord struct { Lemma, MeaningVi, ImageURL string } // ImageURL may be ""

// MemoryPair: an English word and its picture, or its Vietnamese meaning when it has no picture.
type MemoryPair struct { Lemma, ImageURL, MeaningVi string }

// SpotQuestion: one Bắn lỗi sentence as tokens. Wrong is the index range of the mistake (empty when
// the sentence is right); Options are the 3 fixes, Answer the right one.
type SpotQuestion struct {
    Tokens   []string
    Wrong    *Span  // nil = "Không có lỗi"
    Options  []string
    Answer   int
    RuleVi   string
    TopicVi  string // TitleVi of the grammar point
    PointID  string
    Source   string // "choice" | "fix" | "mistake" | "spot"
}
type Span struct{ From, To int } // tokens [From, To)

// DailyQuestion: one question of the day's challenge; exactly one of the pointers is set.
type DailyQuestion struct {
    Kind    string // "picture" | "verb" | "letters" | "grammar"
    Picture *PictureQuestion
    Verb    *DailyVerb // { Base string } — "" lets the page pick a common verb with Seed
    Letters *LetterWord
    Grammar *SpotQuestion
}
```

- **Kết quả một lượt** (`POST /api/games/{game}/results`, không lưu): `Result{Score int; Completed bool; WrongWords
  []string; WrongVerbs []VerbMiss{Base, Form, Typed}}` → `Raise` kỷ lục (khi `Completed`), `vocab.GameMisses(WrongWords)`,
  `verbs.Record(Game: true, Right: false)` cho `WrongVerbs`.

## Thay đổi ở package có sẵn

- `vocab.Service.GameMisses(ctx, userID, lemmas) (rescheduled int, err error)` — R1.
- `verbs.Result.Game bool` và nhánh xử lý trong `ApplyResult` — R1.
- `grammar.Service.PublishedSources(ctx) ([]PointSource, error)`: mỗi điểm đã đăng `{PointID, Level, TitleVi, Exercises
  (Practice+Mastery), Mistakes (Content.Mistakes + MistakesOf)}`; `games` đọc qua interface `GrammarSource`, nên
  `games` không import `grammar` ngược chiều.
- `ai.Provider.GrammarSpotItems` — R5.

## Quyền

| Ai | picture, verbs, letters, memory, daily | grammar (chơi) | spot items (admin) |
|---|---|---|---|
| guest | ✓ (nội dung từ bài được mở) | 403 `members_only` | 403 |
| member | ✓ | ✓ | 403 |
| admin | ✓ | ✓ | ✓ |
