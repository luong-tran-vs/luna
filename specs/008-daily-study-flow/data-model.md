# Data Model: Luồng một ngày học (L)

## 1. goals (MongoDB, mới)

| Trường | Kiểu | Ràng buộc |
|---|---|---|
| `_id` | ObjectID | |
| `userId` | ObjectID | |
| `topicId` | ObjectID | chủ đề F14 |
| `level` | string | trình độ của chủ đề lúc chọn (hiển thị; bài lấy từ lộ trình của chủ đề) |
| `status` | string | `active` \| `paused` \| `completed`; tối đa một `active` mỗi user |
| `effectiveFrom` | string | dayKey bắt đầu có hiệu lực (hôm nay, hoặc ngày mai nếu bài hôm nay đã bắt đầu) |
| `startedAt` | Date | lần đầu chọn |
| `completedAt` | Date \| vắng | khi xong mọi bài của lộ trình |

Index: unique `(userId, topicId)`; `(userId, status)`.

Chuyển trạng thái: chọn chủ đề → goal đó `active` (tạo nếu chưa có), goal `active` cũ → `paused`; bài cuối xong → `completed`
(vẫn là mục tiêu đang hiện cho tới khi chọn chủ đề khác); lộ trình có thêm bài → coi như `active` lại khi đọc.

## 2. lesson_progress (MongoDB, mới)

| Trường | Kiểu | Ràng buộc |
|---|---|---|
| `_id` | ObjectID | |
| `userId`, `lessonId` | ObjectID | unique `(userId, lessonId)` |
| `topicId` | ObjectID | chủ đề lúc học (để hiện ở "Đã học") |
| `dayKey` | string | ngày học (`YYYY-MM-DD` theo múi giờ) |
| `steps` | object | `{review, read, listen}`: `pending` \| `done` |
| `currentStep` | string | `review` \| `read` \| `listen` \| `done` |
| `sentenceIndex` | int | vị trí câu của bước hiện tại (≥ 0) |
| `startedAt` | Date | |
| `completedAt` | Date \| vắng | khi bước cuối xong |

## 3. study_days (MongoDB, mới)

| Trường | Kiểu | Ràng buộc |
|---|---|---|
| `userId` | ObjectID | unique `(userId, dayKey)` |
| `dayKey` | string | |
| `lessonId` | ObjectID | bài của ngày (cố định từ bước đầu tiên xong) |
| `reviewedCount` | int | số thẻ đã ôn ở bước Ôn (bản sao cho F6) |
| `completed` | bool | xong bài của ngày → dùng cho streak |

## 4. review_logs (F5, sửa)

Thêm `context: "daily" | "free"` (vắng = `free`). Index mới `(userId, context, reviewedAt)`.

## 5. Go (`internal/progress`, thêm vào gói có sẵn)

```go
type Step string // "review" | "read" | "listen"; StepDone = "done"
var Steps = []Step{StepReview, StepRead, StepListen}

type Goal struct { ID, UserID, TopicID, Level string; Status GoalStatus; EffectiveFrom string; StartedAt, CompletedAt time.Time }
type LessonProgress struct { UserID, LessonID, TopicID, DayKey string; Steps map[Step]bool; CurrentStep Step
    SentenceIndex int; StartedAt, CompletedAt time.Time }
type StudyDay struct { DayKey, LessonID string; ReviewedCount int; Completed bool }

type TodayKind string // "noGoal" | "studying" | "doneToday" | "noNewLesson"
type TodayState struct { Kind TodayKind; LessonID string; Started bool }

// rules.go (thuần)
func DayKey(now time.Time, loc *time.Location) string
func NextDayKey(key string) string
func EffectiveGoal(goals []Goal) (Goal, bool)
func TodayLesson(goal Goal, roadmap []string, completed map[string]bool, today *StudyDay) TodayState
func CanStartNewLesson(today *StudyDay) bool
func Streak(completedDays []string, today string) int
func ReviewQuota(limit, reviewedToday int) int
func NextStep(steps map[Step]bool) Step

// Ports (main.go)
type Roadmaps interface {
    Roadmap(ctx, topicID string) (TopicInfo, error)         // ErrTopicNotFound
    Topics(ctx, level string) ([]TopicInfo, error)
}
type TopicInfo struct { ID, Name, Level string; LessonIDs []string }
type LessonTitles interface { Titles(ctx, ids []string) (map[string]string, error) }
type Reviews interface {
    DueCount(ctx, userID string) (int, error)
    ReviewedToday(ctx, userID string, since time.Time) (int, error) // context "daily"
}
type Timezones interface { Location(ctx, userID string) (*time.Location, error) }

type GoalRepository interface { List(ctx, userID) ([]Goal, error); Activate(ctx, userID, topicID, level, effectiveFrom string, now time.Time) (Goal, error); SetCompleted(ctx, userID, topicID string, at time.Time) error }
type ProgressRepository interface {
    Get(ctx, userID, lessonID) (LessonProgress, bool, error)
    Completed(ctx, userID) ([]LessonProgress, error)        // có completedAt, mới nhất trước
    Upsert(ctx, p LessonProgress) error
    SetPosition(ctx, userID, lessonID string, step Step, sentence int) error
}
type DayRepository interface {
    Get(ctx, userID, dayKey) (*StudyDay, error)
    Start(ctx, userID, dayKey, lessonID string) error       // upsert, không đổi lessonId đã có
    Update(ctx, userID, dayKey string, reviewed int, completed bool) error
    CompletedKeys(ctx, userID string, from string) ([]string, error)
}
```

## 6. Frontend

```ts
// core/models/study.ts
export type Step = 'review' | 'read' | 'listen';
export type StepState = 'done' | 'current' | 'locked';
export interface GoalView { topicId: string; topicName: string; level: Level; completedLessons: number; totalLessons: number;
  status: 'active' | 'paused' | 'completed'; effectiveFrom: string }
export interface Today { kind: 'noGoal' | 'studying' | 'doneToday' | 'noNewLesson'; goal: GoalView | null;
  lesson: { id: string; title: string } | null; steps: Record<Step, StepState>; currentStep: Step | 'done';
  sentenceIndex: number; reviewCount: number; streak: number; goalCompleted: boolean }
export interface MyLessons { today: { id: string; title: string } | null; completed: { id: string; title: string;
  topicName: string; completedAt: string }[]; upcoming: { id: string; title: string }[] }
```
