# Implementation Plan: Màn hình chính và tiến độ

**Branch**: `009-home-dashboard` | **Date**: 2026-09-30 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/009-home-dashboard/spec.md` + khối 2 trong `docs/spec-inputs/f6-man-hinh-chinh.md`

**Note**: This template is filled in by the `/speckit-plan` command. See `.specify/templates/plan-template.md` for the execution workflow.

## Summary

Thay trang chủ tạm của F0 bằng màn hình chính theo mockup `mvp-features.md` mục 4.1. Backend thêm vào `internal/progress` hai API chỉ
đọc: `GET /api/dashboard` (mục tiêu, thanh kỹ năng Đọc/Nghe trong lộ trình hiện tại, bài hôm nay và các bước từ logic L, nút hành
động, streak, số thẻ đến hạn tới hết ngày mai) và `GET /api/stats` (số thẻ, số câu đã chép, tỷ lệ đúng, số bài theo kỹ năng). Số liệu
tính khi đọc bằng aggregation MongoDB trên `lesson_progress`, `dictation_results`, `cards` (các index cần đã có từ F4/F5/L). Frontend
viết lại `features/home` (thanh mục tiêu, thanh kỹ năng, thẻ bài hôm nay, nút Tiếp tục/Bắt đầu, 3 trạng thái đặc biệt), thêm
`shared/components/progress-bar` (dùng trong `step-indicator` nên `/today` cũng có), chuyển trạng thái kết nối xuống chân trang (chỉ
hiện khi lỗi) và thêm trang `/stats`.

## Technical Context

**Language/Version**: Go 1.25.1 (`GOTOOLCHAIN=local`); TypeScript 5.9 strict + Angular 21

**Primary Dependencies**: không thêm thư viện

**Storage**: không thêm collection. Đọc `goals`, `lesson_progress`, `study_days`, `cards`, `dictation_results`. Index cần đã có:
`cards (userId, due)`, `lesson_progress (userId, completedAt)` và `(userId, lessonId)`, `dictation_results (userId, lessonId,
sentenceIndex)` (tiền tố `userId`)

**Testing**: Go `testing` với fake repository/port và đồng hồ: dashboard ở 4 trạng thái (chưa có mục tiêu, đang học, xong hôm nay,
hết bài), nút Bắt đầu/Tiếp tục (kể cả không có thẻ ôn → "Bắt đầu: Đọc"), kỹ năng chỉ tính trong lộ trình, mốc "hết ngày mai" theo múi
giờ, dashboard không ghi dữ liệu; stats (người học mới, tỷ lệ); handler (200, 401). Vitest cho `progress-bar`, `step-indicator`,
`connection-status`, trang home (4 trạng thái, nút, tải lại) và stats

**Target Platform / Project Type**: như F0–L (web app, Docker compose)

**Performance Goals**: `GET /api/dashboard` < 300 ms với vài trăm bài, vài nghìn thẻ (SC-004 < 1 s cả trang): khoảng 8 truy vấn có
index, không quét toàn bộ

**Constraints**: số liệu của người học trong phiên; dashboard chỉ đọc (không tự hoàn thành bước Ôn như `/api/today`); ngày theo múi
giờ tài khoản; 360 × 640 thấy mục tiêu + bài hôm nay + nút Tiếp tục không cần cuộn, sáng và tối; bàn phím + trình đọc màn hình

**Scale/Scope**: 1 người học, lộ trình vài chục bài, sổ từ vài nghìn thẻ

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

Đối chiếu `.specify/memory/constitution.md` v1.0.0.

- [x] **I. docs/ là nguồn sự thật**: F6 theo `docs/phases/giai-doan-1.md` và `mvp-features.md` mục 4.1 (mockup) + F6; spec bao phủ 3
  tiêu chí nghiệm thu; code ở `internal/progress` và `features/home` đúng `docs/architecture.md`.
- [x] **II. Chi phí 0 đồng**: không dịch vụ hay thư viện mới.
- [x] **III. AI là phần bổ sung**: F6 không gọi AI.
- [x] **IV. Mobile-first**: một cột ở 360px, phần quan trọng ở màn hình đầu; thanh tiến độ có `role="progressbar"` + nhãn + chữ
  "x/y"; trạng thái bước ✓/●/🔒 không chỉ bằng màu; token `--color-primary`, `--color-skill-*` cho cả hai chế độ.
- [x] **V. Dữ liệu người học**: mọi truy vấn lọc `userId` của phiên; DB qua repository; không API nào nhận `userId` từ client.
- [x] **VI. Kiểm thử**: service dashboard/stats phủ 4 trạng thái và các biên (lộ trình rỗng, múi giờ, người học mới); handler; các
  component mới.
- [x] **VII. Đơn giản trước**: tính khi đọc, không bộ đếm, không cache, không job; dùng lại `load`/`view` của L.

**Kết quả**: ĐẠT, không có ngoại lệ.

## Project Structure

### Documentation (this feature)

```text
specs/009-home-dashboard/
├── plan.md, research.md, data-model.md, quickstart.md
├── contracts/dashboard-api.md
└── tasks.md                                 # /speckit-tasks
```

### Source Code (repository root)

```text
backend/
├── cmd/api/main.go                          # dailyReviews thêm DueBefore, CardCount
└── internal/
    ├── progress/
    │   ├── dashboard.go (+ _test)           # MỚI: StudyService.Dashboard, Stats; hàm thuần Action, TomorrowCutoff
    │   ├── dashboard_handler.go (+ _test)   # MỚI: GET /api/dashboard, GET /api/stats (đăng ký trong StudyHandler.Register)
    │   ├── study_repository.go              # ProgressRepository.StepCounts; Reviews.DueBefore, CardCount
    │   ├── repository.go, service.go        # DictationRepository.Totals; Service.Totals
    │   └── study_fake_test.go, fake_test.go # + fake mới
    ├── vocab/
    │   ├── repository.go                    # Repository.CountDue, Count
    │   └── review.go / notebook.go          # Service.DueBefore, Service.Count
    └── storage/mongo/
        ├── study.go                         # LessonProgress.StepCounts (aggregation)
        ├── dictation.go                     # DictationResults.Totals (aggregation)
        └── cards.go                         # Cards.CountDue, Count

frontend/src/app/
├── app.html, app.css                        # + <footer> với lu-connection-status
├── app.routes.ts                            # + 'stats'
├── core/models/dashboard.ts
├── core/services/dashboard-api.service.ts (+ spec)
├── shared/components/progress-bar/ (+ spec)         # MỚI
├── shared/components/step-indicator/        # + thanh "N/3 bước" bằng progress-bar
├── shared/components/connection-status/ (+ spec)    # MỚI: chuyển từ home, chỉ hiện khi lỗi
├── features/home/ (+ spec)                  # viết lại: màn hình chính
└── features/stats/ (+ spec)                 # MỚI: trang thống kê
```

**Structure Decision**: Số liệu tiến độ nằm trong `progress` như `docs/architecture.md` (F4 dictation, L study đã ở đây); dashboard
là phương thức của `StudyService` để dùng lại `load`/`view` của L thay vì tính lại bài hôm nay. Vocab chỉ lộ thêm hai phép đếm qua port
`Reviews`. Frontend giữ `features/home` như khối 2 yêu cầu; `/stats` là feature riêng vì là trang riêng.

## Post-Design Constitution Check

Hai endpoint chỉ đọc, ba phương thức repository dùng aggregation/count trên index sẵn có, ba component dùng chung. Không collection,
thư viện, job hay cache mới. **Vẫn ĐẠT.**

## Complexity Tracking

Không có vi phạm cần giải trình.
