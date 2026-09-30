# Implementation Plan: Sổ từ và ôn tập

**Branch**: `006-vocabulary-review` | **Date**: 2026-09-30 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/006-vocabulary-review/spec.md` + khối 2 trong `docs/spec-inputs/f5-so-tu-on-tap.md`

**Note**: This template is filled in by the `/speckit-plan` command. See `.specify/templates/plan-template.md` for the execution workflow.

## Summary

Mở rộng `internal/vocab` (F3) thành sổ từ có lịch ôn FSRS: thẻ trong collection `cards` thêm trạng thái FSRS (`due`,
`stability`, `difficulty`, …), collection mới `review_logs`. Thư viện `go-fsrs/v3` (tham số mặc định, không fuzz) tính lịch
sau mỗi đánh giá; thẻ mới đến hạn lúc 0 giờ hôm sau theo múi giờ người học; thẻ F3 cũ được coi là thẻ mới mà không cần
migration. API: danh sách thẻ (tìm, lọc bài, trang, ngày theo múi giờ), thêm/sửa/xoá, lưu hàng loạt từ bài, thẻ đến hạn và
đánh giá (chống đánh giá kép bằng `reps`). Gói `lesson` thêm `GET /api/lessons/{id}/vocabulary` từ chú thích + từ điển
offline. Frontend thêm `features/vocabulary` (sổ từ, form, trang ôn tự do), `shared/components/review-session` (dùng lại ở
L), mục Từ vựng trong trang Đọc, link **Sổ từ** trên header.

## Technical Context

**Language/Version**: Go 1.25.1 (`GOTOOLCHAIN=local`); TypeScript 5.9 strict + Angular 21

**Primary Dependencies**: **mới** `github.com/open-spaced-repetition/go-fsrs/v3` v3.3.1 (MIT, không phụ thuộc, `go 1.21`);
frontend không thêm thư viện (dùng lại `AudioPlayer` của F4, `@angular/cdk` đã có)

**Storage**: MongoDB `cards` (thêm trường lịch, index `(userId, due)`, `(userId, lessonId, createdAt)`); **mới**
`review_logs` (index `(userId, cardId)`, `(userId, reviewedAt)`)

**Testing**: Go `testing` + `httptest` với repository giả và đồng hồ giả (`now func() time.Time`); Vitest

**Target Platform / Project Type**: như F0–F4 (web app, backend + frontend, Docker Compose)

**Performance Goals**: thẻ tiếp theo hiện < 1 giây sau khi chọn đánh giá (thẻ đã tải sẵn trong phiên, chỉ chờ POST); danh
sách 30 thẻ/trang

**Constraints**: ngày và hạn ôn theo múi giờ tài khoản (không theo trình duyệt); không AI ở mục Từ vựng; dữ liệu theo người
học; 360px + bàn phím

**Scale/Scope**: 1 người học, vài nghìn thẻ, ≤ 200 thẻ mỗi phiên ôn

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

Đối chiếu `.specify/memory/constitution.md` v1.0.0.

- [x] **I. docs/ là nguồn sự thật**: F5 theo `docs/phases/giai-doan-1.md`; `internal/vocab`, `features/vocabulary`,
  `shared/components/review-session` đúng `docs/architecture.md`; spec bao phủ 7 tiêu chí nghiệm thu F5.
- [x] **II. Chi phí 0 đồng**: `go-fsrs` MIT, chạy cục bộ; audio thẻ dùng TTS cục bộ của F3.
- [x] **III. AI là phần bổ sung**: F5 không gọi AI; mục Từ vựng chỉ đọc chú thích đã có + từ điển offline; ôn tập chạy đủ
  khi không có AI.
- [x] **IV. Mobile-first**: phiên ôn một cột, nút đánh giá ≥ 44px, ô gõ Nghe rồi gõ dùng thanh trả lời dính đáy như F4;
  đúng/sai có chữ và biểu tượng, không chỉ màu; bàn phím đầy đủ (Space lật, 1–4 đánh giá).
- [x] **V. Dữ liệu người học**: mọi truy vấn lọc `userId` của phiên; DB qua `vocab.Repository`, `vocab.ReviewLogRepository`;
  bài và múi giờ qua port (`LessonVocabulary`, `LessonTitles`, `Timezones`).
- [x] **VI. Kiểm thử**: lịch FSRS sau từng mức (so giá trị tham chiếu), thẻ mới đến hạn hôm sau (sát nửa đêm, nhiều múi
  giờ), sửa giữ lịch, xoá xoá log, bulk không trùng, nhóm ngày sát nửa đêm, bài chưa có chú thích, chống đánh giá kép; handler
  mọi endpoint (thành công, lỗi đầu vào, người khác, 401); frontend review-session, sổ từ, mục Từ vựng.
- [x] **VII. Đơn giản trước**: không migration (thẻ cũ suy lịch khi đọc, research R2); không transaction (cập nhật có điều
  kiện theo `reps`, xoá idempotent); `day` luôn trả, không cần `groupBy`; ✓ tính từ tập từ đã lưu trang Đọc đã tải.

**Kết quả**: ĐẠT, không có ngoại lệ.

## Project Structure

### Documentation (this feature)

```text
specs/006-vocabulary-review/
├── plan.md, research.md, data-model.md, quickstart.md
├── contracts/vocab-api.md
└── tasks.md
```

### Source Code (repository root)

```text
backend/
├── go.mod                                   # + go-fsrs/v3 v3.3.1
├── cmd/api/main.go                          # + ports Timezones, LessonVocabulary, LessonTitles; review log repo
└── internal/
    ├── vocab/
    │   ├── model.go, repository.go          # + Schedule, ReviewLog, Rating, Mode, ports
    │   ├── schedule.go (+ _test)            # FirstDue, toFSRS/fromFSRS, Intervals (bọc go-fsrs)
    │   ├── service.go (+ _test)             # Save (lessonId tuỳ chọn, due), SaveBulk, List, Lessons, Update, Delete, Due, Review
    │   ├── handler.go (+ _test)             # các endpoint mới trong contracts/vocab-api.md
    │   └── fake_test.go                     # repo + log repo + ports giả
    ├── lesson/
    │   ├── reading.go (+ _test)             # Reader.Vocabulary
    │   └── reading_handler.go (+ _test)     # GET /api/lessons/{id}/vocabulary
    └── storage/mongo/
        ├── cards.go                         # + lịch, List, Get, UpdateDetails, UpdateSchedule, Due, NextDue, Delete, LessonCounts
        ├── reviewlogs.go                    # vocab.ReviewLogRepository
        └── indexes.go                       # + index mới

frontend/src/app/
├── app.routes.ts                            # + 'vocabulary' (lazy, authGuard)
├── core/models/vocab.ts                     # + lịch, DueCard, CardPage, VocabItem, …
├── core/services/vocab-api.service.ts (+ spec)   # dùng chung cho sổ từ, review-session, mục Từ vựng
├── shared/utils/answer-match.ts (+ spec)
├── shared/utils/interval-label.ts (+ spec)
├── shared/components/review-session/ (+ spec)    # 2 kiểu ôn, 4 nút, output finished
├── shared/components/app-header/            # + link "Sổ từ"
├── features/vocabulary/
│   ├── vocabulary.routes.ts                 # '' (sổ từ), 'review'
│   ├── notebook/ (+ spec)                   # nhóm ngày, tìm, lọc, xem thêm, sửa, xoá
│   ├── card-form/ (+ spec)                  # thêm / sửa thẻ
│   └── review/ (+ spec)                     # trang ôn tự do
└── features/lesson/reading/
    ├── lesson-vocabulary/ (+ spec)          # mục Từ vựng, Lưu, Lưu tất cả, ✓
    └── reading.{ts,html}                    # gắn mục Từ vựng, dùng chung tập từ đã lưu
```

**Structure Decision**: `vocab` giữ mọi thứ về thẻ và lịch ôn (F3 + F5). Gọi API thẻ được chuyển vào
`core/services/vocab-api.service.ts` vì dùng ở ba nơi (sổ từ, trang Đọc, `review-session` trong `shared`); `ReadingApiService`
giữ `getLesson`, `lookup` và chuyển `saveCard`/`words` sang service mới. `review-session` tự gửi đánh giá để L dùng lại mà không
phải nối API.

## Post-Design Constitution Check

Thêm 1 thư viện nhỏ (FSRS, bắt buộc theo đầu vào), 1 collection, 3 port và 1 service frontend dùng chung. Không migration,
không transaction. **Vẫn ĐẠT.**

## Complexity Tracking

Không có vi phạm cần giải trình.
