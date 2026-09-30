# Data Model: Chủ đề và lộ trình theo trình độ (F14)

## 1. topics (MongoDB, mới)

| Trường | Kiểu | Ràng buộc |
|---|---|---|
| `_id` | ObjectID | |
| `name` | string | 1–60 ký tự, đã cắt khoảng trắng |
| `nameKey` | string | `name` viết thường, gộp khoảng trắng |
| `level` | string | A1–C2 |
| `description` | string | 0–200 |
| `lessonIds` | ObjectID[] | lộ trình có thứ tự; không trùng; chỉ bài có `topicId` = `_id` |
| `createdAt`, `updatedAt` | Date | |

Index: **unique `(level, nameKey)`**.

## 2. lessons (MongoDB, sửa từ F2)

| Trường | Thay đổi |
|---|---|
| `topic` (string) | **bỏ** (migration `$unset`) |
| `topicId` | **mới**, ObjectID, bắt buộc |
| `level` | giữ; luôn bằng `level` của chủ đề |

Index mới: `(topicId)`; giữ `(level)`; bỏ index `(topic)`.

## 3. migrations (MongoDB, mới)

`{_id: "f14-topics", doneAt: Date}` — có document thì không chạy lại chuyển dữ liệu F14.

## 4. roadmap (MongoDB, bỏ)

Collection của F2 bị xoá sau khi chuyển dữ liệu xong.

## 5. Go

```go
// internal/topic
type Topic struct { ID, Name, Level, Description string; LessonIDs []string; CreatedAt, UpdatedAt time.Time }
type Input struct { Name, Level, Description string }
type Summary struct { Topic; LessonCount, Remaining int; Warning bool } // danh sách quản trị
type Roadmap struct { Topic Topic; Lessons []LessonRef; Remaining int; Warning bool }
type LessonRef struct { ID, Title string; AudioStatus, AnnotationStatus string } // từ port Lessons
type InUseError struct { Count int }
var ErrNotFound, ErrNameTaken

type Repository interface {
    Create(ctx, t Topic) (Topic, error)               // ErrNameTaken (khoá unique)
    Get(ctx, id) (Topic, error)                       // ErrNotFound
    List(ctx, level string) ([]Topic, error)          // level "" = tất cả; theo level rồi tên
    Update(ctx, id string, in Input) (Topic, error)   // ErrNameTaken, ErrNotFound
    Delete(ctx, id) error
    SetLessons(ctx, id string, lessonIDs []string) error
    RemoveLesson(ctx, id, lessonID string) (bool, error) // $pull; true nếu đã có
    AppendLesson(ctx, id, lessonID string) error          // $push nếu chưa có
}
type Lessons interface {  // port, hiện thực trên lesson.Repository
    CountByTopic(ctx) (map[string]int, error)
    TopicOf(ctx, ids []string) (map[string]string, error)
    Refs(ctx, ids []string) ([]LessonRef, error)
    SetLevelByTopic(ctx, topicID, level string) error
}

type LegacyLesson struct { ID, Level, TopicName, TopicID string; CreatedAt time.Time }
type Plan struct { Topics []Topic /* ID rỗng = tạo mới */; Assign map[string]string /* lessonID → khoá hoặc id chủ đề */ }
func PlanMigration(lessons []LegacyLesson, roadmap []string, existing []Topic) Plan

// internal/lesson (sửa)
type Lesson struct { ...; TopicID string /* thay Topic */ }
type Summary struct { ...; TopicID string; TopicName string }
type Input struct { Title, Content, TopicID, Source, License string } // bỏ Level, Topic
type Filter struct { Level Level; TopicID string }
type TopicRef struct { ID, Name string; Level Level }
type Topics interface { // port, hiện thực bởi topic.Service
    Get(ctx, id string) (TopicRef, error)                  // ErrTopicNotFound
    Names(ctx) (map[string]TopicRef, error)
    RoadmapLessonIDs(ctx) (map[string]bool, error)
    MoveLesson(ctx, lessonID, from, to string) error
}
```

`lesson.RoadmapRepository`, `Service.GetRoadmap/SetRoadmap`, `mongo.Roadmap` bị bỏ.

## 6. Frontend

```ts
// core/models/topic.ts
export interface Topic { id: string; name: string; level: Level; description: string;
  lessonCount: number; roadmapCount: number; remaining: number; warning: boolean; createdAt: string }
export interface TopicInput { name: string; level: Level | ''; description: string }
export interface TopicRoadmap { topic: Topic; lessons: LessonSummary[]; remaining: number; warning: boolean }
// core/models/lesson.ts: LessonSummary.topic → topicId + topicName; LessonInput bỏ level/topic, thêm topicId;
// LessonFilter { level?: Level | ''; topicId?: string }; bỏ Roadmap
```
