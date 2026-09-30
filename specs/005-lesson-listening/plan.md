# Implementation Plan: Nghe

**Branch**: `005-lesson-listening` | **Date**: 2026-09-29 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/005-lesson-listening/spec.md` + khối 2 trong `docs/spec-inputs/f4-nghe.md`

**Note**: This template is filled in by the `/speckit-plan` command. See `.specify/templates/plan-template.md` for the execution workflow.

## Summary

Bước Nghe: người học nghe từng câu (trước/sau/lặp lại, 4 tốc độ, ẩn/hiện chữ), gõ lại câu và bấm Kiểm tra; kết quả so
sánh từng từ (đúng / sai kèm từ đúng / thiếu) tính ngay trên trình duyệt. Mỗi lần kiểm tra được gửi lên backend để lưu
kết quả mới nhất của câu; backend trả tổng kết của bài (đã kiểm tra X/N, tỷ lệ đúng, hoàn thành). Frontend thêm
`shared/utils/dictation-compare.ts` (căn chỉnh bằng khoảng cách chỉnh sửa theo từ), `shared/components/audio-player`,
trang `/lessons/:id/listen`. Backend thêm domain `internal/progress` với collection `dictation_results`.

## Technical Context

**Language/Version**: Go 1.25; TypeScript 5.9 strict + Angular 21

**Primary Dependencies**: không thêm thư viện mới (backend: thư viện chuẩn + mongo-driver; frontend: Angular, dùng
`HTMLAudioElement.playbackRate`)

**Storage**: MongoDB **mới** `dictation_results` (unique `userId + lessonId + sentenceIndex`). Audio câu đã có từ F2.

**Testing**: Go `testing` + `httptest` với repository giả; Vitest (bộ mẫu so sánh đầy đủ)

**Target Platform / Project Type**: như F0–F3

**Performance Goals**: so sánh < 100ms cho câu 40 từ (SC-002, thuật toán O(n·m) với n, m ≤ ~60); lặp lại câu phát < 1 giây
(audio file cục bộ, đã cache theo URL immutable của F2)

**Constraints**: kết quả theo người học (không nhận `userId` từ client); không cần mạng khi so sánh; 360px với bàn phím ảo

**Scale/Scope**: 1 người học; bài ≤ 200 câu

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

Đối chiếu `.specify/memory/constitution.md` v1.0.0.

- [x] **I. docs/ là nguồn sự thật**: F4 theo `docs/phases/`; `internal/progress`, `features/lesson/listening`,
  `shared/components/audio-player`, `shared/utils/dictation-compare.ts` đúng `docs/architecture.md`; spec bao phủ 5 tiêu chí
  nghiệm thu F4.
- [x] **II. Chi phí 0 đồng**: không dịch vụ hay thư viện mới.
- [x] **III. AI là phần bổ sung**: F4 không dùng AI.
- [x] **IV. Mobile-first**: thanh trả lời dính đáy màn hình, viewport `interactive-widget=resizes-content`; sai/thiếu bằng
  gạch ngang/gạch chân chấm + nhãn cho trình đọc màn hình; chữ luôn `--color-text`, màu chỉ ở đường gạch (research R4).
- [x] **V. Dữ liệu người học**: kết quả lọc theo `userId` của phiên; DB qua `progress.DictationRepository`; bài qua port
  `progress.Lessons`.
- [x] **VI. Kiểm thử**: unit test so sánh (bộ mẫu SC-001), service (lưu mới nhất, tổng kết, bỏ kết quả của nội dung cũ, kiểm
  tra đầu vào), handler (thành công, lỗi đầu vào, người khác không đọc được, 401); frontend test player, trang Nghe.
- [x] **VII. Đơn giản trước**: so sánh chỉ ở frontend, backend lưu số liệu client gửi (research R2); không thư viện diff.

**Kết quả**: ĐẠT, không có ngoại lệ.

## Project Structure

### Documentation (this feature)

```text
specs/005-lesson-listening/
├── plan.md, research.md, data-model.md, quickstart.md
├── contracts/dictation-api.md
└── tasks.md
```

### Source Code (repository root)

```text
backend/
├── cmd/api/main.go                          # + progress service, routes
└── internal/
    ├── progress/
    │   ├── model.go, repository.go          # DictationResult, Summary, DictationRepository, Lessons port
    │   ├── service.go (+ _test)             # Record, Summary
    │   ├── handler.go (+ _test)             # POST dictation, GET dictation/summary
    │   └── fake_test.go
    └── storage/mongo/
        ├── dictation.go                     # progress.DictationRepository (upsert)
        └── indexes.go                       # + dictation_results unique index

frontend/src/
├── index.html                               # viewport interactive-widget=resizes-content
└── app/
    ├── core/models/dictation.ts
    ├── shared/utils/dictation-compare.ts (+ spec)
    ├── shared/components/audio-player/ (+ spec)
    ├── features/lesson/listening-api.service.ts (+ spec)
    ├── features/lesson/listening/ (+ spec)  # trang Nghe
    ├── features/lesson/lesson.routes.ts     # + ':id/listen'
    └── features/admin/lesson-detail/        # + "Mở bước Nghe"
```

**Structure Decision**: `internal/progress` là domain tiến độ theo `docs/architecture.md` (L và F6 sẽ mở rộng). Trang Nghe
dùng lại `GET /api/lessons/{id}` của F3 để lấy câu và URL audio.

## Post-Design Constitution Check

Thêm 2 interface nhỏ (`DictationRepository` — nguyên tắc V; `Lessons` — để `progress` không phụ thuộc gói `lesson`) và 1
collection. **Vẫn ĐẠT.**

## Complexity Tracking

Không có vi phạm cần giải trình.
