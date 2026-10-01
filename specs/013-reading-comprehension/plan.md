# Implementation Plan: Câu hỏi hiểu bài và ghi chú ngữ pháp

**Branch**: `013-reading-comprehension` | **Date**: 2026-10-01 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/013-reading-comprehension/spec.md` + khối 2 trong
`docs/spec-inputs/f15-hieu-bai-ngu-phap.md`

## Summary

**Chú thích bài (AI).** `ai.Provider.Annotate` trả `ai.LessonExtras` gồm chú thích từ, 3–5 câu hỏi, ghi chú ngữ pháp và đề
viết, vẫn trong **một** request Gemini (`responseSchema` thành OBJECT). `lesson.CleanExtras`:

- bỏ câu hỏi hỏng;
- bỏ ví dụ ngữ pháp không có trong bài;
- phần nào hỏng thì để trống phần đó, không làm chú thích từ thất bại.

**Lưu bài.** `lessons` có thêm `questions`, `grammarNote`, `writingPrompt`, `extrasEditedByAdmin` và `quizVersion`. Bộ đếm
`quizVersion` tăng mỗi khi bộ câu hỏi bị thay.

**Quản trị viên.**

- Sửa câu hỏi, ngữ pháp, đề viết qua `PUT /api/admin/lessons/{id}/extras`.
- **Chạy lại chú thích** dùng được cả khi chú thích đã xong. Nếu có phần đã sửa tay thì hiện cảnh báo trước.

**Người học.**

- `GET /api/lessons/{id}` trả quiz không kèm đáp án, kèm câu trả lời đã có của chính người học, và ghi chú ngữ pháp.
- `POST /api/lessons/{id}/answers` ghi mỗi câu đúng một lần (index unique trên `reading_answers`) và trả về đúng/sai, đáp án,
  giải thích.
- Bước Đọc: `CompleteStep(read)` kiểm tra ở server rằng đã trả lời hết câu hỏi. Bài không có câu hỏi giữ nút "Đã đọc xong".
- `/api/stats` có thêm tỷ lệ hiểu bài.
- Xuất dữ liệu có thêm `readingAnswers`.

**Frontend.**

- `lu-comprehension-quiz`: ✓/✗ kèm chữ "Đúng"/"Sai", khoá câu sau khi chọn, tiếp tục đúng câu khi vào lại.
- `lu-grammar-note`: dùng `<details>`.
- `lu-lesson-extras` ở trang chi tiết bài.
- Thêm thẻ "Hiểu bài" ở trang thống kê.

## Technical Context

**Language/Version**: Go 1.25.1 (`GOTOOLCHAIN=local`); TypeScript 5.9 strict + Angular 21 (standalone, signals, zoneless,
OnPush)

**Primary Dependencies**: không thêm thư viện; Gemini `generateContent` như F2/F7

**Storage**:

- MongoDB, thêm collection `reading_answers` (index unique 4 trường).
- `lessons` thêm 5 trường, không cần migration.

**Testing**:

- Go `testing`: `CleanExtras`, `ProcessAnnotate`, `Retry`, `UpdateExtras`, `Reader.View/Answer`, progress đọc có/không câu
  hỏi, stats, Gemini httptest, handler.
- Vitest: quiz, grammar note, reading, lesson-extras, lesson-detail, stats.

**Target Platform / Project Type**: web app như F0–F7 (Docker compose)

**Performance Goals**:

- Trả lời một câu < 1 s (SC-004): 1 lần đọc bài + 1 insert.
- GET bài thêm 1 truy vấn câu trả lời (theo index).

**Constraints**:

- 1 request AI mỗi lần chú thích (SC-001).
- Không lộ đáp án trước khi trả lời (FR-011).
- Đúng/sai phân biệt được không cần màu (FR-013).
- Hoàn thành bước do server kiểm tra (FR-015).

**Scale/Scope**: vài người học, tối đa 5 câu mỗi bài, vài trăm câu trả lời mỗi người mỗi tháng

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

Đối chiếu `.specify/memory/constitution.md` v1.0.0.

- [x] **I. docs/ là nguồn sự thật**:
  - F15 theo `docs/phases/giai-doan-2.md` (4 tiêu chí nghiệm thu).
  - Mã đặt trong `internal/ai`, `internal/lesson`, `internal/progress`, `features/lesson/reading`, `features/admin`.
- [x] **II. Chi phí 0 đồng**: không thêm request AI; dùng gói miễn phí sẵn có.
- [x] **III. AI là phần bổ sung**:
  - Phần mới sinh trong job chú thích nền sẵn có: có trạng thái, chạy lại được, kết quả lưu vào bài.
  - AI lỗi thì bước Đọc dùng nút "Đã đọc xong".
- [x] **IV. Mobile-first**:
  - Lựa chọn là nút ≥ 44px; kết quả bằng chữ + ký hiệu + `aria-live`.
  - `<details>` gốc.
  - Một cột ở 360px.
- [x] **V. Dữ liệu người học**:
  - Câu trả lời lọc theo `userId` của phiên.
  - GET bài chỉ trả câu trả lời của chính người học.
  - Có trong file xuất (F13).
  - Không gửi dữ liệu người học cho AI.
- [x] **VI. Kiểm thử**: logic làm sạch, chấm, ghi một lần, hoàn thành bước, thống kê đều có test (R12).
- [x] **VII. Đơn giản trước**:
  - Một collection, một bộ đếm phiên bản, hai route mới.
  - Hoàn thành bước dùng lại route sẵn có.

**Kết quả**: ĐẠT, không có ngoại lệ.

## Project Structure

### Documentation (this feature)

```text
specs/013-reading-comprehension/
├── plan.md, research.md, data-model.md, quickstart.md
├── contracts/reading-quiz-api.md
├── checklists/requirements.md
└── tasks.md                                       # /speckit-tasks
```

### Source Code (repository root)

```text
backend/
├── cmd/api/main.go                                # readingQuiz adapter, answers repo, Reader deps
└── internal/
    ├── ai/ai.go                                   # LessonExtras, Question, GrammarNote; Annotate → LessonExtras
    ├── ai/gemini/gemini.go (+ _test)              # schema OBJECT, prompt mở rộng, LimitReader 2 MB
    ├── lesson/
    │   ├── model.go                               # Question, GrammarNote, Extras; Lesson + Extras, ExtrasEditedByAdmin, QuizVersion; lỗi mới
    │   ├── extras.go (+ _test)                    # MỚI: CleanExtras, ValidateExtras (lỗi theo ô), so sánh bộ câu hỏi
    │   ├── service.go (+ _test)                   # ProcessAnnotate lưu extras; Retry annotate khi done; UpdateExtras
    │   ├── handler.go (+ _test)                   # lessonJSON + extras; PUT /extras
    │   ├── quiz.go (+ _test)                      # MỚI: Answer, AnswerRepository, Reader.Answer, QuizStatus, quiz view
    │   ├── reading.go, reading_handler.go (+ _test)   # View(userID,…), quiz/grammarNote JSON, POST /answers
    │   ├── repository.go                          # SaveAnnotations(+extras), ReplaceExtras
    │   └── fake_test.go                           # fakes mới
    ├── progress/
    │   ├── study.go, study_repository.go (+ _test)  # ReadingQuiz port, ErrReadIncomplete trong completeStep(read)
    │   ├── stats.go, dashboard_handler.go (+ _test) # ReadingTotals, ReadingRate, JSON reading
    │   └── study_handler.go                       # 409 read_incomplete
    ├── export/model.go, service.go (+ _test)      # reading_answers
    └── storage/mongo/
        ├── lessons.go                             # trường mới, SaveAnnotations/ReplaceContent/ReplaceExtras, $inc quizVersion
        ├── reading_answers.go (+ _test nếu có)    # MỚI: Insert (dup → ErrAlreadyAnswered), Get, List, Totals
        └── indexes.go                             # unique index reading_answers

frontend/src/app/
├── core/models/lesson.ts, reading.ts, dashboard.ts
├── features/lesson/
│   ├── reading-api.service.ts (+ spec)            # answer()
│   └── reading/
│       ├── reading.ts/.html/.css (+ spec)         # quiz + grammar note, bỏ "Đã đọc xong" khi có quiz, nút Tiếp tục
│       ├── comprehension-quiz/ (+ spec)           # MỚI
│       └── grammar-note/ (+ spec)                 # MỚI
├── features/admin/
│   ├── admin-api.service.ts (+ spec)              # updateExtras()
│   ├── lesson-extras/ (+ spec)                    # MỚI: form câu hỏi, ngữ pháp, đề viết
│   └── lesson-detail/ (+ spec)                    # nhúng lesson-extras; Chạy lại chú thích khi done + cảnh báo
└── features/stats/ (+ spec)                       # thẻ Hiểu bài

README.md                                          # mục bước Đọc / quản trị: câu hỏi, ngữ pháp, chạy lại chú thích
docs/phases/giai-doan-2.md                         # tick tiêu chí F15 khi xong
```

**Structure Decision**:

- Câu hỏi, câu trả lời và việc chấm nằm trong `lesson`, vì gắn với nội dung bài và `Reader` đã phục vụ bước Đọc.
- `progress` chỉ hỏi qua port `ReadingQuiz` để giữ chiều phụ thuộc hiện có: `main` nối hai bên, `progress` không import `lesson`.
- `export` thêm một collection theo mẫu sẵn có.

## Post-Design Constitution Check

- Thêm một collection có index unique, hai route mới, mở rộng ba route sẵn có, thêm một port.
- Không job, thư viện hay request AI mới.

**Vẫn ĐẠT.**

## Complexity Tracking

| Điểm | Vì sao cần | Phương án đơn giản hơn bị loại vì |
| --- | --- | --- |
| `quizVersion` riêng thay vì `revision` (lệch khối 2) | Câu hỏi bị thay khi quản trị viên sửa hoặc chạy lại chú thích, mà `revision` không đổi; câu trả lời cũ không được áp vào bộ câu hỏi mới (FR-018) | Dùng `revision` sẽ hiện câu trả lời cũ cho câu hỏi mới và tính sai "đã trả lời hết" |
