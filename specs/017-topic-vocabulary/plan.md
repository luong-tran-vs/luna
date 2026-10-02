# Implementation Plan: Từ vựng theo chủ đề

**Branch**: `017-topic-vocabulary` | **Date**: 2026-10-02 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/017-topic-vocabulary/spec.md` + khối 2 trong `docs/spec-inputs/f18-tu-vung-chu-de.md`

## Summary

**Backend.**

- `topic.Topic` có `Words`, `WordsSeeded`. Danh sách nhúng (`go:embed`, 42 chủ đề, 1.257 từ) được nạp khi khởi động cho chủ đề chưa từng
  xét, so tên đã chuẩn hoá (khoẻ = khỏe), ghi có điều kiện để không bao giờ ghi đè danh sách đã sửa.
- Gói mới `internal/wordmatch` khớp từ/cụm trong văn bản (nguyên từ, hoa thường, dạng biến đổi qua `dictionary.Candidates` và lemma chú
  thích). Dùng cho độ phủ (`topic`), từ còn thiếu của bản nháp và từ cần chú thích (`lesson`).
- API quản trị: độ phủ trong danh sách chủ đề; `GET/PUT …/words`; `GET …/word-plan` (chia từ: chưa dùng → dùng ít nhất, không trùng
  khi đủ từ).
- Sinh bài: `targetWords` mỗi bài, kiểm tra thuộc danh sách chủ đề, đưa vào prompt (vẫn 1 request), bản nháp có `targetWords`,
  `missingWords`.
- Chú thích: `ai.AnnotateRequest{FocusWords}`; từ của chủ đề có trong bài mà AI bỏ sót được thêm bằng nghĩa từ điển.

**Frontend.** Trang Chủ đề hiện "Từ vựng: đã dùng X/Y" và link tới trang mới **Từ vựng** (`admin/topics/:id/words`: nhãn Đã dùng/Chưa
dùng, thêm nhiều từ, xoá, lỗi theo từ). Hộp thoại sinh bài có "Từ mục tiêu mỗi bài" và nhóm từ sửa được; danh sách bản nháp hiện số từ
mục tiêu đã dùng và từ còn thiếu.

## Technical Context

**Language/Version**: Go 1.25.1 (`GOTOOLCHAIN=local`); TypeScript 5.9 strict + Angular 21

**Primary Dependencies**: không thêm thư viện. `embed` (thư viện chuẩn), `dictionary.Candidates` sẵn có, Gemini `generateContent`.

**Storage**: MongoDB `topics` thêm `words`, `wordsSeeded`; `lessons` không đổi. Không index mới.

**Testing**: Go (`wordmatch`, `topic`, `lesson`, `ai/gemini`, `storage/mongo`), Vitest (`topic-words`, `topics`, `generate-dialog`,
`draft-list`, `admin-api`). Xem research R9.

**Target Platform / Project Type**: web app (Go REST + Angular SPA), Docker Compose

**Performance Goals**: danh sách chủ đề tính độ phủ cho mọi chủ đề từ một lần đọc bài (vài trăm bài) < 500 ms; word-plan < 300 ms.

**Constraints**: 1 request AI mỗi lượt sinh và mỗi lần chú thích; không đổi luồng học (FR-019); dữ liệu ban đầu không ghi đè (FR-003).

**Scale/Scope**: ~45 chủ đề × ≤ 100 từ; vài trăm bài.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

Đối chiếu `.specify/memory/constitution.md` v1.0.0.

- [x] **I. docs/ là nguồn sự thật**: F18 theo `docs/phases/giai-doan-3.md` (7 tiêu chí, đều có test hoặc bước quickstart). Code theo
  `docs/architecture.md` (domain + port, interface chỉ cho DB và AI).
- [x] **II. Chi phí 0 đồng**: không thêm request AI (gửi kèm từ trong request sẵn có); dữ liệu ban đầu chỉ là danh sách từ tiếng Anh
  tham khảo, không chép phiên âm/nghĩa.
- [x] **III. AI là phần bổ sung**: AI lỗi thì danh sách từ, độ phủ, chia từ vẫn chạy; sinh bài báo lỗi như cũ và giữ hộp thoại; chú
  thích lỗi như cũ. Bổ sung từ bằng từ điển không cần AI.
- [x] **IV. Mobile-first**: trang Từ vựng và nhóm từ `flex-wrap`, nút ≥ 44px, nhãn có chữ, sáng/tối bằng token, bàn phím.
- [x] **V. Dữ liệu người học**: chỉ API quản trị (`RequireAdmin`); không dữ liệu người học; DB qua repository.
- [x] **VI. Kiểm thử**: hàm thuần (`CleanWords`, `SeedKey`, `PlanSeed`, `Coverage`, `PlanWords`, `wordmatch`) có unit test; endpoint có
  test thành công / lỗi đầu vào / sai quyền; AI giả lập.
- [x] **VII. Đơn giản trước**: không thư viện mới; độ phủ tính khi xem, không lưu; seed bằng cờ theo chủ đề. Gói `wordmatch` mới là
  cần thiết để `topic` và `lesson` dùng chung một cách khớp mà không import nhau.

**Kết quả**: ĐẠT, không ngoại lệ.

## Project Structure

### Documentation (this feature)

```text
specs/017-topic-vocabulary/
├── plan.md, research.md, data-model.md, quickstart.md
├── contracts/topic-words-api.md
├── checklists/requirements.md
└── tasks.md                                   # /speckit-tasks
```

### Source Code (repository root)

```text
backend/
├── cmd/api/main.go                            # topicLessons.Texts; lessonTopicsPort.Get điền Words; lesson.Deps.Dict
└── internal/
    ├── wordmatch/wordmatch.go (+ _test)       # MỚI: New, Find, Contains
    ├── ai/ai.go, ai/gemini/gemini.go (+ _test)# AnnotateRequest, FocusWords; GenerateRequest.TargetWords; prompt
    ├── topic/
    │   ├── model.go                           # Words, WordsSeeded, WordUse, LessonText, Summary.WordCount/UsedWordCount
    │   ├── words.go (+ _test)                 # MỚI: CleanWords, Coverage, PlanWords
    │   ├── seed.go (+ _test), seed/topic-words.json  # MỚI: go:embed, LoadSeed, SeedKey, PlanSeed
    │   ├── repository.go                      # SetWords; Lessons.Texts
    │   ├── service.go (+ _test)               # List độ phủ, Words, SetWords, WordPlan
    │   └── handler.go (+ _test)               # GET/PUT words, GET word-plan, topicJSON + wordCount
    ├── lesson/
    │   ├── model.go, generate.go, generate_handler.go (+ _test)  # TargetWords, MissingWords
    │   ├── focus.go (+ _test)                 # MỚI: focusWords, addMissedFocus
    │   ├── service.go                         # ProcessAnnotate gửi FocusWords, bổ sung từ; Deps.Dict
    │   └── fake_test.go                       # fakeAI.Annotate(req), fakeTopics words, fake dict
    ├── writing/fake_test.go                   # chữ ký Annotate mới
    └── storage/mongo/
        ├── topics.go (+ _test)                # words, wordsSeeded, SetWords
        ├── seed_topic_words.go                # MỚI: SeedTopicWords
        ├── lessons.go                         # TopicTexts
        └── indexes.go                         # PrepareInBackground gọi SeedTopicWords sau MigrateTopics

docs/spec-inputs/f18-topic-words.json          # sửa "Intensive care unit (ICU)"

frontend/src/app/
├── core/models/topic.ts, generate.ts          # wordCount, TopicWord, targetWords, missingWords
├── features/admin/admin-api.service.ts (+ spec)          # topicWords, setTopicWords, wordPlan
├── features/admin/admin.routes.ts                        # topics/:id/words
├── features/admin/topics/ (+ spec)                       # nhãn độ phủ, link Từ vựng
├── features/admin/topic-words/ (+ spec)                  # MỚI
├── features/admin/generate-dialog/ (+ spec)      # từ mục tiêu, nhóm từ
├── features/admin/draft-list/ (+ spec)           # Dùng a/b, Còn thiếu
└── features/admin/roadmap/roadmap.ts (+ spec)            # nạp từ chủ đề cho hộp thoại

README.md                                      # mục từ vựng theo chủ đề
docs/phases/giai-doan-3.md                     # tick tiêu chí F18 khi xong
```

**Structure Decision**: danh sách từ thuộc chủ đề → `internal/topic`; sinh bài và chú thích ở `internal/lesson` nhận từ qua port
`Topics` sẵn có (`TopicRef.Words`); độ phủ cần nội dung bài → port `topic.Lessons.Texts` sẵn có kiểu adapter trong main. Cách khớp chung
ở `internal/wordmatch`. Frontend thêm một trang quản trị riêng thay vì panel trong danh sách chủ đề (danh sách tới 100 từ).

## Post-Design Constitution Check

- 1 gói mới, 3 route mới, 2 trường chủ đề, 1 file nhúng; không collection, không thư viện mới.
- Lệch khối 2 đều ghi ở research: gói `wordmatch` (R3), bổ sung từ bằng từ điển theo FR-018 (R7), sửa 1 từ không hợp lệ trong dữ liệu
  (R2).

**Vẫn ĐẠT.** Không có mục Complexity Tracking.
