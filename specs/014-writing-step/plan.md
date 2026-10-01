# Implementation Plan: Bước Viết

**Branch**: `014-writing-step` | **Date**: 2026-10-01 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/014-writing-step/spec.md` + khối 2 trong `docs/spec-inputs/f8-viet.md`

## Summary

Bước thứ 4 **Viết** sau Nghe:

- `progress.Steps` có thêm `StepWrite`.
- Cổng mới chặn hoàn thành bước Viết cho tới khi đã nộp bài, kiểm tra qua port `Writings.Submitted`.
- Bài đã xong trước F8 vẫn là xong.

Gói mới `internal/writing` và collection `writings` (một bài viết mỗi người mỗi bài học):

- Nháp tự lưu (`PUT /api/lessons/{id}/writing`).
- Nộp (`POST …/writing/submit`, 5–400 từ):
  - đề lấy từ bài, bài chưa có đề thì dùng "Tóm tắt bài bằng 3–5 câu.";
  - lưu lúc nộp;
  - xếp job `grade`.
- Client nối tiếp bằng `POST /api/today/steps/write/complete`, nên bài hôm nay xong ngay. Chỉ bài hôm nay đang ở bước Viết mới được viết (`CanWrite`).

Chấm nền:

- Job `grade` trên worker sẵn có. `job.Job` có thêm `TargetID`; job chấm không gắn bài học.
- `ai.GradeWriting` gửi 1 request Gemini, schema cố định 4 tiêu chí.
- Kết quả hỏng thì thử lại. Lỗi hẳn thì `failed` kèm lý do và cho **Chấm lại**.
- Có kết quả mới thì `seen = false` để báo người học.

Frontend:

- `lu-writing`: soạn bài, đếm từ, lưu nháp sau 1 s, Nộp. Dùng trong `/today` và ở `lessons/:id/write` để xem lại.
- Trang `writings` (danh sách và chi tiết):
  - so sánh theo từ bằng `wordDiff`: `<ins>`/`<del>` có nhãn ẩn;
  - nút Chấm lại.
- `WritingNotifier`:
  - poll `unseen-count` 30 s, chỉ khi còn bài đang chấm;
  - dấu báo ở liên kết "Bài viết" trên header;
  - toast `lu-toast`.
- Ngoài ra:
  - thanh Viết ở màn hình chính;
  - step-indicator 4 bước;
  - thống kê có số bài viết và điểm trung bình;
  - file xuất có `writings`.

## Technical Context

**Language/Version**: Go 1.25.1 (`GOTOOLCHAIN=local`); TypeScript 5.9 strict + Angular 21 (standalone, signals, zoneless, OnPush)

**Primary Dependencies**:
- Không thêm thư viện.
- Gemini `generateContent`, worker job sẵn có của F2.

**Storage**:
- Collection mới `writings`, có 3 index.
- `jobs` thêm `targetId`; `lessonId` không bắt buộc với job chấm.
- `lesson_progress.steps.write` tự có.
- Không migration.

**Testing**:
- Go: `writing` (service, handler, `ProcessGrade`/`JobFailed`/`Regrade`, quyền), `ai/gemini` (`GradeWriting`), `progress` (4 bước, write gate, `CanWrite`, dashboard/stats), `storage/mongo` (jobs `targetId`, chuyển đổi doc), `export`.
- Vitest: `word-diff`, `lu-writing`, danh sách và chi tiết bài viết, notifier/toast/header, today, home, stats, my-lessons.

**Target Platform / Project Type**: web app như F0–F15 (Docker compose)

**Performance Goals**:
- Nộp và hoàn thành bài < 1 s (SC-001), vì việc chấm chạy nền.
- Thường có kết quả < 2 phút (SC-003).
- Báo kết quả trong ≤ 1 phút nhờ poll 30 s (SC-004).

**Constraints**:
- AI lỗi không chặn nộp hay hoàn thành bài (FR-011).
- Bài viết chỉ người viết xem được (FR-017).
- So sánh không dựa vào màu (FR-014).
- Dùng được ở 360px và chỉ bằng bàn phím.

**Scale/Scope**: vài người học, 1 bài viết mỗi người mỗi ngày, ≤ 400 từ

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

Đối chiếu `.specify/memory/constitution.md` v1.0.0.

- [x] **I. docs/ là nguồn sự thật**:
  - F8 theo `docs/phases/giai-doan-2.md` (5 tiêu chí nghiệm thu).
  - Gói domain mới `internal/writing`, feature `features/writings`, `features/lesson/writing`, đúng `docs/architecture.md`.
- [x] **II. Chi phí 0 đồng**:
  - 1 request AI mỗi lần chấm (gói miễn phí).
  - Thông báo trong app, không dịch vụ đẩy.
- [x] **III. AI là phần bổ sung**:
  - Chấm chạy nền: có trạng thái, thử lại, Chấm lại; kết quả được lưu.
  - Nộp và hoàn thành bài không phụ thuộc AI.
- [x] **IV. Mobile-first**:
  - Ô soạn một cột ở 360px.
  - Thêm/bớt có kiểu chữ và nhãn ẩn.
  - Lưu nháp, đang chấm và có kết quả được báo qua `aria-live`.
  - Toast không che nội dung.
- [x] **V. Dữ liệu người học**:
  - Mọi truy vấn lọc theo `userId` của phiên; bài của người khác trả 404.
  - Gửi AI chỉ bài viết, đề và nội dung bài học.
  - Có trong file xuất.
- [x] **VI. Kiểm thử**: logic nháp, nộp, chấm, chấm lại, quyền, 4 bước và diff đều có test (R10).
- [x] **VII. Đơn giản trước**:
  - Dùng lại worker job, guard L, route hoàn thành bước.
  - Poll có điều kiện thay vì SSE.
  - Diff LCS tự viết ~40 dòng, không thêm thư viện.

**Kết quả**: ĐẠT, không có ngoại lệ.

## Project Structure

### Documentation (this feature)

```text
specs/014-writing-step/
├── plan.md, research.md, data-model.md, quickstart.md
├── contracts/writing-api.md
├── checklists/requirements.md
└── tasks.md                                     # /speckit-tasks
```

### Source Code (repository root)

```text
backend/
├── cmd/api/main.go                              # writing service/handler, ports, worker handler grade, onFailed theo loại
└── internal/
    ├── ai/ai.go, ai/gemini/gemini.go (+ _test)  # GradeRequest, Grade, Criterion, GradeWriting
    ├── job/model.go                             # TypeGrade, Job.TargetID
    ├── writing/                                 # MỚI
    │   ├── model.go                             # Writing, Grade, Criterion, Summary, UnseenCount, lỗi, hằng số
    │   ├── repository.go                        # Repository, Lessons, Steps ports
    │   ├── service.go (+ _test)                 # Get/SaveDraft/Submit/List/Detail/Seen/Regrade/Unseen
    │   ├── grade.go (+ _test)                   # ProcessGrade, CheckGrade, JobFailed, failureMessage
    │   ├── handler.go (+ _test)                 # routes /api/lessons/{id}/writing*, /api/writings*
    │   └── fake_test.go
    ├── progress/                                # StepWrite, Writings port, write gate, CanWrite, Write counts, stats writing
    ├── export/                                  # writings
    └── storage/mongo/
        ├── writings.go (+ _test)                # MỚI
        ├── jobs.go                              # targetId, lessonId tuỳ chọn
        ├── study.go                             # StepCounts write
        └── indexes.go                           # index writings

frontend/src/app/
├── core/models/writing.ts (+ spec), study.ts, dashboard.ts
├── core/services/writing-api.service.ts (+ spec), writing-notifier.service.ts (+ spec)
├── shared/utils/word-diff.ts (+ spec)
├── shared/components/toast/ (+ spec)            # MỚI
├── shared/components/app-header/                # liên kết Bài viết + dấu báo
├── shared/components/step-indicator/            # 4 bước
├── shared/components/progress-bar/              # tone write
├── features/lesson/writing/ (+ spec)            # MỚI: lu-writing
├── features/lesson/today/, lesson.routes.ts, my-lessons/
├── features/writings/ (list, detail + specs)    # MỚI
├── features/home/, features/stats/
└── app.routes.ts, app.html                      # routes writings, toast

README.md                                        # mục "Bước Viết và bài viết"
docs/phases/giai-doan-2.md                       # tick tiêu chí F8 khi xong
```

**Structure Decision**:

- Bài viết là domain riêng (`internal/writing`). Gói này chỉ gặp `lesson`, `progress` và `job` qua port, do `main` nối.
- `progress` biết bước Viết tồn tại và hỏi "đã nộp chưa" qua port, như cách đã làm với Nghe (F4) và Đọc (F15).
- Job chấm dùng chung worker nhưng không gắn bài học, nên việc dọn job của bài học không chạm tới nó.

## Post-Design Constitution Check

Thêm một collection, một loại job, một port AI, mười route, một bước, hai trang, một toast. Không có thư viện, dịch vụ hay ngoại lệ mới. **Vẫn ĐẠT.**

## Complexity Tracking

Không có vi phạm cần giải trình.
