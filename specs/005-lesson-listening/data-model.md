# Data Model: Nghe (F4)

## 1. dictation_results (MongoDB, mới)

| Trường | Kiểu | Ràng buộc |
|---|---|---|
| `_id` | ObjectID | |
| `userId` | ObjectID | từ phiên |
| `lessonId` | ObjectID | |
| `lessonRevision` | int | revision của bài lúc lưu (lấy ở backend) |
| `sentenceIndex` | int | 0 ≤ i < số câu |
| `typed` | string | câu người học gõ, 1–1000 ký tự |
| `correctWords` | int | 0 ≤ correct ≤ total |
| `totalWords` | int | 1–1000 |
| `checkedAt` | Date | |

Index: **unique `(userId, lessonId, sentenceIndex)`**. Ghi bằng upsert: lần kiểm tra mới nhất thay lần cũ.

## 2. Tổng kết (tính khi đọc, không lưu)

```go
type Summary struct {
    SentenceCount int
    CheckedCount  int
    CorrectWords  int
    TotalWords    int
    Rate          float64 // CorrectWords / TotalWords, 0 khi TotalWords = 0
    Completed     bool    // CheckedCount == SentenceCount (và SentenceCount > 0)
    Results       []Result // chỉ của revision hiện tại, theo sentenceIndex
}
```

## 3. Interface (Go, `internal/progress`)

```go
type Result struct { SentenceIndex int; Typed string; CorrectWords, TotalWords int; CheckedAt time.Time }

type DictationRepository interface {
    Upsert(ctx, userID, lessonID string, revision int, r Result) error
    List(ctx, userID, lessonID string) ([]StoredResult, error) // StoredResult = Result + Revision
}

// Port tới bài học (hiện thực bằng lesson.Repository trong main.go).
type Lessons interface {
    Info(ctx, lessonID string) (revision, sentenceCount int, err error) // ErrLessonNotFound
}
```

## 4. So sánh (frontend, không lưu)

```ts
// shared/utils/dictation-compare.ts
export type WordStatus = 'ok' | 'wrong' | 'missing';
export interface ComparedWord { status: WordStatus; word?: string; expected?: string; }
export interface Comparison { words: ComparedWord[]; correctWords: number; totalWords: number; }
export function normalizeWords(text: string): string[];
export function compareDictation(expected: string, typed: string): Comparison;
```

## 5. Kiểu API frontend

```ts
// core/models/dictation.ts
export interface DictationResult { sentenceIndex: number; typed: string; correctWords: number; totalWords: number; checkedAt: string; }
export interface DictationSummary { sentenceCount: number; checkedCount: number; correctWords: number; totalWords: number;
  rate: number; completed: boolean; results: DictationResult[]; }
export interface DictationInput { sentenceIndex: number; typed: string; correctWords: number; totalWords: number; }
```
