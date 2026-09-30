# Data Model: Trang quản trị bài học (F2)

## 1. lessons

| Trường | Kiểu | Ràng buộc / ý nghĩa |
|---|---|---|
| `_id` | ObjectID | |
| `title` | string | 1–200 ký tự |
| `content` | string | 1–10.000 ký tự, văn bản gốc quản trị viên dán |
| `level` | string | `A1` `A2` `B1` `B2` `C1` `C2` |
| `topic` | string | 0–60 ký tự |
| `source` | string | 1–200 ký tự |
| `license` | string | 1–100 ký tự |
| `revision` | int | bắt đầu 1, +1 mỗi khi `content` đổi (research R1) |
| `sentences` | array | `[{index int, text string, audioPath string}]`, `audioPath` rỗng tới khi audio xong |
| `audioStatus` | string | `running` \| `done` \| `failed` |
| `audioError` | string | lý do ngắn khi `failed` |
| `annotationStatus` | string | `running` \| `done` \| `failed` |
| `annotationError` | string | |
| `annotations` | array | `[{text, lemma, meaningVi string, sentenceIndex int, editedByAdmin bool}]` |
| `createdAt`, `updatedAt` | Date | UTC |

Index: `{createdAt: -1}` (danh sách), `{level: 1}`, `{topic: 1}`.

**Chuyển trạng thái (audio và chú thích độc lập)**

```text
tạo bài / sửa nội dung ──► running ──(job xong)──► done
                              │
                              └─(3 lần lỗi hoặc lỗi không thử lại)──► failed ──(Chạy lại)──► running
```

Sửa thông tin khác `content` không đổi `revision`, câu, audio, chú thích.

## 2. roadmap

| Trường | Kiểu | Ý nghĩa |
|---|---|---|
| `_id` | string | luôn `"main"` |
| `lessonIds` | array ObjectID | thứ tự bài, không trùng |
| `updatedAt` | Date | |

Chưa có document → lộ trình trống.

## 3. jobs

| Trường | Kiểu | Ý nghĩa |
|---|---|---|
| `_id` | ObjectID | |
| `type` | string | `tts` \| `annotate` |
| `lessonId` | ObjectID | |
| `revision` | int | revision của bài khi tạo job |
| `status` | string | `pending` \| `running` \| `done` \| `failed` |
| `attempts` | int | số lần đã chạy (tối đa 3) |
| `error` | string | lỗi gần nhất |
| `runAt` | Date | thời điểm sớm nhất được chạy (backoff) |
| `createdAt`, `updatedAt` | Date | |

Index: `{status: 1, runAt: 1}`, `{lessonId: 1}`.

```text
pending ──(worker lấy)──► running ──ok──► done
   ▲                         │
   └── lỗi, attempts < 3 ────┤ (runAt = now + 30s / 2m)
                             └── lỗi, attempts = 3 hoặc không thử lại ──► failed
khởi động lại: running ──► pending
```

## 4. File audio

`{AUDIO_DIR}/{lessonId}/{revision}/{index}.mp3`. URL: `/api/audio/{lessonId}/{revision}/{index}`.

## 5. Interface (Go)

```go
// internal/lesson
type Repository interface {
    Create(ctx, Lesson) (Lesson, error)
    Get(ctx, id string) (Lesson, error)                 // ErrNotFound
    List(ctx, Filter) ([]Summary, error)                // Filter{Level, Topic}
    Update(ctx, Lesson) error                           // thông tin + nội dung, do service quyết định
    UpdateAnnotations(ctx, id string, []Annotation) error
    SetStatus(ctx, id string, kind JobKind, st Status, errMsg string) error
    // Ghi kết quả chỉ khi revision còn khớp; trả false nếu đã cũ.
    SaveAudio(ctx, id string, revision int, paths []string) (bool, error)
    SaveAnnotations(ctx, id string, revision int, []Annotation) (bool, error)
    Delete(ctx, id string) error
    Exists(ctx, ids []string) (bool, error)
}
type RoadmapRepository interface {
    Get(ctx) ([]string, error)
    Set(ctx, ids []string) error
}

// internal/job
type Repository interface {
    Enqueue(ctx, Job) error
    ClaimNext(ctx, now time.Time) (Job, bool, error)
    Complete(ctx, id string) error
    Retry(ctx, id string, runAt time.Time, errMsg string) error
    Fail(ctx, id string, errMsg string) error
    DeletePending(ctx, lessonID string) error
    DeleteForLesson(ctx, lessonID string) error
    ResetRunning(ctx) (int64, error)
}

// internal/tts
type Synthesizer interface { Synthesize(ctx, text string) ([]byte, error) }

// internal/ai
type Annotation struct { Text, Lemma, MeaningVi string; SentenceIndex int }
type Provider interface { Annotate(ctx, sentences []string, level string) ([]Annotation, error) }
```

`ai.Provider.Annotate` nhận danh sách câu (thay vì văn bản thô như khối 2) để AI trả `sentenceIndex` khớp với câu
đã tách.

## 6. Kiểu frontend

```ts
// core/models/lesson.ts
export type Level = 'A1' | 'A2' | 'B1' | 'B2' | 'C1' | 'C2';
export type JobStatus = 'running' | 'done' | 'failed';
export interface Sentence { index: number; text: string; audioUrl: string | null; }
export interface Annotation { text: string; lemma: string; meaningVi: string; sentenceIndex: number; editedByAdmin: boolean; }
export interface LessonSummary { id: string; title: string; level: Level; topic: string; audioStatus: JobStatus; annotationStatus: JobStatus; inRoadmap: boolean; createdAt: string; }
export interface Lesson extends LessonSummary { content: string; source: string; license: string; revision: number; audioError: string; annotationError: string; sentences: Sentence[]; annotations: Annotation[]; }
export interface Roadmap { lessons: LessonSummary[]; remaining: number; warning: boolean; }
```
