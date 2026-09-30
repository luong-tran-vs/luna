# Implementation Plan: Luồng một ngày học

**Branch**: `008-daily-study-flow` | **Date**: 2026-09-30 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/008-daily-study-flow/spec.md` + khối 2 trong `docs/spec-inputs/l-luong-mot-ngay-hoc.md`

**Note**: This template is filled in by the `/speckit-plan` command. See `.specify/templates/plan-template.md` for the execution workflow.

## Summary

Nối F3 (Đọc), F4 (Nghe), F5 (Ôn), F14 (chủ đề, lộ trình) thành một ngày học. Backend mở rộng `internal/progress` với mục tiêu
(`goals`), tiến độ từng bài (`lesson_progress`) và ngày học (`study_days`); mọi quy tắc (ngày theo múi giờ, bài hôm nay, hiệu lực
đổi chủ đề, streak, giới hạn thẻ) là hàm thuần trong `rules.go`. API `/api/goals`, `/api/today…`, `/api/lessons/mine`; middleware
chặn người học mở bài chưa tới lượt. F5 thêm `context` cho log ôn để đếm đúng 30 thẻ/ngày ở bước Ôn. Frontend thêm trang `/goal`,
`/today` (thanh bước + bước hiện tại, dùng lại `review-session`, `Reading`, `Listening`), `/lessons`, nút "Học hôm nay" trên trang
chủ, link "Bài học" trên header.

## Technical Context

**Language/Version**: Go 1.25.1 (`GOTOOLCHAIN=local`); TypeScript 5.9 strict + Angular 21

**Primary Dependencies**: không thêm thư viện

**Storage**: MongoDB **mới** `goals` (unique `userId+topicId`), `lesson_progress` (unique `userId+lessonId`), `study_days` (unique
`userId+dayKey`); `review_logs` thêm `context` + index `(userId, context, reviewedAt)`

**Testing**: Go `testing` (rules thuần: nửa đêm, đổi múi giờ, nghỉ nhiều ngày, hết lộ trình, đổi chủ đề trước/sau khi bắt đầu, quay
lại chủ đề cũ), service/handler với fake repository, port và đồng hồ; Vitest cho step-indicator, trang goal/today/my-lessons, các
input mới của Reading/Listening/review-session

**Target Platform / Project Type**: như F0–F14

**Performance Goals**: `GET /api/today` < 300 ms với vài trăm bài và vài nghìn thẻ (vài truy vấn có index); vào bước hiện tại trong ≤ 2
lần bấm (SC-008)

**Constraints**: ngày theo múi giờ tài khoản; server kiểm tra thứ tự bước; hoàn thành bước idempotent; dữ liệu theo người học;
360px + bàn phím

**Scale/Scope**: 1 người học, lộ trình vài chục bài, streak hàng trăm ngày

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

Đối chiếu `.specify/memory/constitution.md` v1.0.0.

- [x] **I. docs/ là nguồn sự thật**: L theo `docs/phases/giai-doan-1.md` và `mvp-features.md` mục 4 (R1–R4); spec bao phủ 9 tiêu chí
  nghiệm thu; `features/lesson` và `internal/progress` đúng `docs/architecture.md`.
- [x] **II. Chi phí 0 đồng**: không dịch vụ hay thư viện mới.
- [x] **III. AI là phần bổ sung**: L không gọi AI.
- [x] **IV. Mobile-first**: thanh bước ✓/●/🔒 kèm chữ cho trình đọc màn hình, không chỉ màu; một cột ở 360px; mọi nút ≥ 44px.
- [x] **V. Dữ liệu người học**: mọi truy vấn lọc `userId` của phiên; DB qua repository; middleware chặn bài chưa tới lượt; quản trị
  viên chỉ được bỏ qua khoá bài, không đọc tiến độ người khác.
- [x] **VI. Kiểm thử**: `rules.go` thuần phủ đủ tình huống khối 2; service (thứ tự bước, idempotent, quota, hiệu lực đổi chủ đề);
  handler (thành công, lỗi, 401, 403 khoá bài); frontend các trang mới và input mới.
- [x] **VII. Đơn giản trước**: tiến độ goal tính khi đọc (không bộ đếm), không job nền, không transaction (upsert theo khoá duy
  nhất, idempotent); đếm thẻ bằng log sẵn có.

**Kết quả**: ĐẠT, không có ngoại lệ.

## Project Structure

### Documentation (this feature)

```text
specs/008-daily-study-flow/
├── plan.md, research.md, data-model.md, quickstart.md
├── contracts/study-api.md
└── tasks.md
```

### Source Code (repository root)

```text
backend/
├── cmd/api/main.go                          # + ports Roadmaps, LessonTitles, Reviews, Timezones; study service/handler; guard
└── internal/
    ├── progress/
    │   ├── rules.go (+ _test)               # MỚI: DayKey, EffectiveGoal, TodayLesson, Streak, ReviewQuota, NextStep
    │   ├── study_model.go, study_repository.go   # MỚI: Goal, LessonProgress, StudyDay, repos, ports
    │   ├── study.go (+ _test)               # MỚI: StudyService: Goals, SetGoal, Today, ReviewCards, CompleteStep, SetPosition, MyLessons, CanOpen
    │   ├── study_handler.go (+ _test)       # MỚI: /api/goals, /api/today…, /api/lessons/mine; Guard middleware
    │   ├── handler.go                       # dictation routes nhận guard
    │   └── fake_test.go                     # + fake repos/ports
    ├── vocab/                               # review context (daily|free), ReviewedSince
    ├── lesson/reading_handler.go            # Register nhận guard
    └── storage/mongo/
        ├── goals.go, lessonprogress.go, studydays.go   # MỚI
        ├── reviewlogs.go                    # + context, CountSince
        └── indexes.go                       # + index mới

frontend/src/app/
├── app.routes.ts                            # + 'today', 'goal'
├── core/models/study.ts
├── core/services/study-api.service.ts (+ spec)
├── shared/components/step-indicator/ (+ spec)
├── shared/components/review-session/        # + input context
├── shared/components/app-header/            # + "Bài học"
├── features/home/                           # + nút "Học hôm nay"
└── features/lesson/
    ├── lesson.routes.ts                     # '' → my-lessons
    ├── goal/ (+ spec)                       # chọn trình độ → chủ đề, chúc mừng
    ├── today/ (+ spec)                      # thanh bước + bước hiện tại
    ├── my-lessons/ (+ spec)                 # Hôm nay / Đã học / Sắp tới
    ├── reading/                             # + input lessonId, startSentence, mode; output position
    └── listening/                           # + input lessonId, startSentence, mode; output position
```

**Structure Decision**: Mọi thứ "tiến độ người học" nằm trong `progress` như `docs/architecture.md` (F4 dictation đã ở đây, F6 sẽ
đọc từ đây). Các trang học đặt trong `features/lesson` cạnh reading/listening để dùng lại.

## Post-Design Constitution Check

Thêm 3 collection, 4 port, 1 middleware; sửa nhỏ F3/F4/F5 (input, context, guard). Không thư viện, không job, không transaction.
**Vẫn ĐẠT.**

## Complexity Tracking

Không có vi phạm cần giải trình.
