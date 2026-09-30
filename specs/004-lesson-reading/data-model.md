# Data Model: Đọc (F3)

## 1. cards (MongoDB, mới)

| Trường | Kiểu | Ràng buộc |
|---|---|---|
| `_id` | ObjectID | |
| `userId` | ObjectID | chủ sổ; luôn lấy từ phiên |
| `text` | string | từ/cụm như gặp trong bài, 1–100 ký tự |
| `lemma` | string | dạng gốc, chữ thường, trim, gộp khoảng trắng, 1–100 ký tự |
| `ipa` | string | có thể rỗng |
| `meaningVi` | string | 1–200 ký tự |
| `contextSentence` | string | câu chứa từ, ≤ 1000 ký tự |
| `lessonId` | ObjectID | bài nguồn |
| `source` | string | `ai` \| `dictionary` \| `manual` |
| `createdAt` | Date | |

Index: **unique `(userId, lemma)`** (chống trùng, FR-015), `(userId, createdAt)`.
F5 sẽ thêm trường lịch ôn (FSRS) và audio.

## 2. Từ điển (SQLite chỉ đọc, ngoài MongoDB)

Bảng dùng: `words`, `word_definitions`, `definitions`, `pronunciations` (research R1).

```go
// internal/dictionary
type Meaning struct { POS, Text string }
type Entry struct { Word, IPA string; Meanings []Meaning } // Meanings ≤ 3, definition_lang = vi
```

## 3. Kiểu trong domain lesson

```go
// Port do internal/dictionary hiện thực
type Dictionary interface {
    Lookup(ctx context.Context, word string) (dictionary.Entry, bool, error)
}

type LookupResult struct {
    Source   string // "ai" | "dictionary"
    Text     string // như người học chọn
    Lemma    string
    IPA      string
    Meanings []Meaning // POS có thể rỗng với nguồn AI
}

type ReadingView struct {
    ID, Title string; Level Level; Topic string
    Sentences  []Sentence        // index, text, audioPath
    Paragraphs [][]int           // chỉ số câu theo đoạn
    Lemmas     map[string]string // token chữ thường → dạng gốc
    Phrases    []Phrase          // chú thích nhiều từ {Text, Lemma}
}
```

## 4. Âm thanh từ

`{AUDIO_DIR}/words/{sha256(text chuẩn hoá)}.mp3`, dùng chung mọi người học, không hết hạn.

## 5. Interface vocab

```go
type Card struct { ID, UserID, Text, Lemma, IPA, MeaningVi, ContextSentence, LessonID, Source string; CreatedAt time.Time }
type Repository interface {
    Create(ctx, Card) (Card, error)                         // ErrExists khi trùng (userId, lemma)
    FindByLemma(ctx, userID, lemma string) (Card, error)    // ErrNotFound
    Words(ctx, userID string) ([]WordRef, error)            // WordRef{Lemma, Text}
}
```

## 6. Kiểu frontend

```ts
// core/models/reading.ts
export interface ReadingLesson { id: string; title: string; level: Level; topic: string;
  sentences: Sentence[]; paragraphs: number[][]; lemmas: Record<string, string>; phrases: { text: string; lemma: string }[]; }
export interface LookupResult { source: 'ai' | 'dictionary'; text: string; lemma: string; ipa: string;
  meanings: { pos: string; text: string }[]; }
export interface Token { text: string; isWord: boolean; index: number; }
// core/models/vocab.ts
export interface Card { id: string; text: string; lemma: string; ipa: string; meaningVi: string;
  contextSentence: string; lessonId: string; source: 'ai' | 'dictionary' | 'manual'; createdAt: string; }
export interface WordRef { lemma: string; text: string; }
```
