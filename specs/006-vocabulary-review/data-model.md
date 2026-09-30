# Data Model: Sổ từ và ôn tập (F5)

## 1. cards (MongoDB, mở rộng từ F3)

| Trường | Kiểu | Ràng buộc |
|---|---|---|
| `_id` | ObjectID | |
| `userId` | ObjectID | từ phiên |
| `text` | string | 1–100 ký tự; không sửa được |
| `lemma` | string | 1–100, viết thường, gộp khoảng trắng; **unique theo `userId`**; không sửa được |
| `ipa` | string | 0–100 |
| `meaningVi` | string | 1–200 |
| `contextSentence` | string | 0–1000 |
| `lessonId` | ObjectID \| **vắng** | vắng = thẻ tự thêm (F5 nới từ bắt buộc) |
| `source` | string | `ai` \| `dictionary` \| `manual` |
| `createdAt` | Date | |
| `due` | Date \| vắng | **mới**; thẻ mới = `FirstDue(createdAt)`; vắng ở thẻ F3 cũ (research R2) |
| `stability` | double | mới, FSRS |
| `difficulty` | double | mới, FSRS |
| `elapsedDays` | int | mới, FSRS |
| `scheduledDays` | int | mới, FSRS |
| `reps` | int | mới; vắng = 0; dùng chống đánh giá kép (research R3) |
| `lapses` | int | mới |
| `state` | int | mới; 0 New, 1 Learning, 2 Review, 3 Relearning; vắng = New |
| `lastReview` | Date \| vắng | mới |

Index: `(userId, lemma)` unique (F3); `(userId, createdAt)` (F3); **mới** `(userId, due)`, `(userId, lessonId, createdAt)`.

## 2. review_logs (MongoDB, mới)

| Trường | Kiểu | Ghi chú |
|---|---|---|
| `_id` | ObjectID | |
| `userId` | ObjectID | |
| `cardId` | ObjectID | xoá cùng thẻ |
| `rating` | int | 1 Again, 2 Hard, 3 Good, 4 Easy |
| `mode` | string | `flip` (Xem từ đoán nghĩa) \| `listen` (Nghe rồi gõ) |
| `reviewedAt` | Date | |
| `stateBefore` | object | `{due, stability, difficulty, elapsedDays, scheduledDays, reps, lapses, state, lastReview}` trước khi ôn |

Index: `(userId, cardId)`, `(userId, reviewedAt)` (F6 dùng).

## 3. Go (`internal/vocab`)

```go
type Schedule struct { // ánh xạ 1-1 với fsrs.Card
    Due time.Time; Stability, Difficulty float64; ElapsedDays, ScheduledDays, Reps, Lapses uint64
    State int; LastReview time.Time
}
type Card struct { /* các trường F3 */ ; Schedule Schedule } // LessonID "" = tự thêm

type Rating int // 1..4
type Mode string // "flip" | "listen"
type ReviewLog struct { UserID, CardID string; Rating Rating; Mode Mode; ReviewedAt time.Time; Before Schedule }

type Intervals struct { Again, Hard, Good, Easy time.Duration }
type DueCard struct { Card; Intervals Intervals }

type ListQuery struct { Q, LessonID string /* "manual" = tự thêm */; Page int }
type Page struct { Cards []DayCard; HasMore bool; Today, Yesterday string }
type DayCard struct { Card; Day string } // "YYYY-MM-DD" theo múi giờ người học

type Repository interface {
    Create(ctx, c Card) (Card, error)                       // F3; ErrExists
    FindByLemma(ctx, userID, lemma string) (Card, error)    // F3
    Words(ctx, userID string) ([]WordRef, error)            // F3
    Get(ctx, userID, id string) (Card, error)               // ErrNotFound
    List(ctx, userID string, q ListQuery, limit, skip int) ([]Card, bool, error)
    LessonCounts(ctx, userID string) (map[string]int, error) // "" = tự thêm
    UpdateDetails(ctx, userID, id string, d Details) (Card, error) // chỉ meaningVi, ipa, contextSentence
    // UpdateSchedule ghi s nếu thẻ vẫn có reps = expectedReps; ok=false khi không khớp.
    UpdateSchedule(ctx, userID, id string, expectedReps uint64, s Schedule) (bool, error)
    Due(ctx, userID string, now, startOfToday time.Time, limit int) ([]Card, int, error) // thẻ, tổng số
    NextDue(ctx, userID string, now, startOfToday time.Time) (time.Time, bool, error)
    Delete(ctx, userID, id string) (bool, error)
}
type ReviewLogRepository interface {
    Add(ctx, l ReviewLog) error
    DeleteByCard(ctx, userID, cardID string) (int, error)
}

// Ports (hiện thực trong main.go)
type Timezones interface { Timezone(ctx, userID string) (*time.Location, error) }  // qua auth
type LessonVocabulary interface { Vocabulary(ctx, lessonID string) ([]VocabItem, error) } // qua lesson.Reader
type LessonTitles interface { Titles(ctx, ids []string) (map[string]string, error) }      // qua lesson.Repository.Summaries
```

Service (`NewService(Deps{Repo, Logs, Lessons, Timezones, Vocabulary, Titles, Now func() time.Time})`):
`Save` (F3, lessonId tuỳ chọn, gán `FirstDue`), `SaveBulk`, `List`, `Lessons`, `Update`, `Delete`, `Due`, `Review`.

## 4. Gói `lesson`

```go
type VocabItem struct { Lemma, Text, MeaningVi, IPA string; SentenceIndex int; Sentence string }
func (r *Reader) Vocabulary(ctx, lessonID string) (items []VocabItem, available bool, err error)
```

## 5. Frontend

```ts
// core/models/vocab.ts (mở rộng)
export type ReviewMode = 'flip' | 'listen';
export type Rating = 1 | 2 | 3 | 4;
export interface Intervals { again: number; hard: number; good: number; easy: number } // giây
export interface Card { id; text; lemma; ipa; meaningVi; contextSentence; lessonId: string | null; source; createdAt;
  due: string; reps: number; state: 'new' | 'learning' | 'review' | 'relearning' }
export interface DayCard extends Card { day: string }
export interface CardPage { cards: DayCard[]; hasMore: boolean; today: string; yesterday: string }
export interface DueCard extends Card { intervals: Intervals }
export interface DueList { cards: DueCard[]; total: number; nextDue: string | null }
export interface VocabItem { lemma; text; meaningVi; ipa; sentenceIndex: number; sentence }
export interface ReviewSummary { reviewed: number; counts: Record<Rating, number> }
```

## 6. Chuyển trạng thái thẻ (FSRS, research R1)

```text
New ──Again/Hard/Good──▶ Learning (đến hạn sau vài phút) ──Good──▶ Review (vài ngày)
New ──Easy──▶ Review
Review ──Again──▶ Relearning (lapses+1) ──Good──▶ Review
Review ──Hard/Good/Easy──▶ Review (khoảng cách tăng)
```
