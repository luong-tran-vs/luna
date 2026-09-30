# Implementation Plan: Trang quản trị bài học

**Branch**: `003-lesson-admin` | **Date**: 2026-09-29 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/003-lesson-admin/spec.md` + khối 2 trong
`docs/spec-inputs/f2-quan-tri-bai-hoc.md`

**Note**: This template is filled in by the `/speckit-plan` command. See `.specify/templates/plan-template.md` for the execution workflow.

## Summary

Quản trị viên tạo, sửa, xoá bài học và xếp một lộ trình chung. Backend thêm domain `lesson` (bài, tách câu,
lộ trình), `job` (hàng đợi việc nền trong MongoDB + một worker goroutine), `tts` (interface `Synthesizer`, hiện
thực gọi Kokoro-FastAPI CPU), `ai` (interface `Provider`, hiện thực Gemini `generateContent` với JSON schema).
Mỗi bài có `revision`; việc nền mang revision và chỉ ghi kết quả khi revision còn khớp, nên sửa nội dung giữa
chừng không bị kết quả cũ ghi đè. Audio lưu trên volume, phục vụ qua `GET /api/audio/...` (cần đăng nhập).
Frontend thay trang admin tạm của F1 bằng: danh sách bài (chip trạng thái, Chạy lại, lọc), form bài, chi tiết
bài (xem trước, nghe từng câu, bảng chú thích sửa được), lộ trình (kéo thả `@angular/cdk` + nút Lên/Xuống);
polling 5 giây khi còn việc đang chạy.

## Technical Context

**Language/Version**: Go 1.25 (backend); TypeScript 5.9 strict + Angular 21 (frontend)

**Primary Dependencies**:
- Backend: thư viện chuẩn, `mongo-driver/v2`, `x/crypto` (đã có). Không thêm thư viện Go mới: gọi Kokoro và
  Gemini bằng `net/http`.
- Frontend: **mới** `@angular/cdk` (MIT, bản 21.x khớp Angular) cho drag-drop.
- Dịch vụ: **mới** container `ghcr.io/remsky/kokoro-fastapi-cpu` (Apache 2.0, model Kokoro-82M Apache 2.0);
  Gemini API gói miễn phí (model mặc định `gemini-3.5-flash-lite`, đổi bằng `GEMINI_MODEL`).

**Storage**: MongoDB: `lessons`, `roadmap` (1 document), `jobs`. File audio: volume `/data/audio`.

**Testing**: Go `testing` + `httptest` với repository, `Synthesizer`, `Provider` giả; client Kokoro/Gemini test
bằng `httptest.Server`. Frontend Vitest.

**Target Platform**: như F0/F1; Kokoro chạy CPU trong Docker (không GPU).

**Project Type**: Web application

**Performance Goals**: Bài 20 câu có audio + chú thích ≤ 5 phút trên CPU (SC-002); danh sách cập nhật ≤ 10 giây
(polling 5 giây, SC-008).

**Constraints**: Chi phí 0 đồng; lưu bài không phụ thuộc AI/TTS; đúng 1 request AI mỗi lần tạo/sửa/chạy lại; audio
không tạo lại; test không cần mạng; 360px.

**Scale/Scope**: vài chục đến vài trăm bài; 1 quản trị viên; 1 worker.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

Đối chiếu `.specify/memory/constitution.md` v1.0.0.

- [x] **I. docs/ là nguồn sự thật**: F2 là tính năng kế tiếp; cấu trúc `internal/{lesson,job,tts,ai}`,
  `features/admin` đúng `docs/architecture.md`; spec bao phủ 5 tiêu chí nghiệm thu F2. Bộ lọc trạng thái học để
  sang L (ghi trong spec Assumptions).
- [x] **II. Chi phí 0 đồng**: Kokoro chạy trên máy (Apache 2.0); Gemini gói miễn phí; `@angular/cdk` MIT.
- [x] **III. AI là phần bổ sung**: lưu bài và tạo audio không phụ thuộc AI; AI qua interface `ai.Provider`, chọn
  bằng `AI_PROVIDER`; việc nền có trạng thái, chạy lại được, kết quả lưu vào `lessons.annotations` để không gọi
  lặp. TTS cũng qua interface `tts.Synthesizer`.
- [x] **IV. Mobile-first**: mọi trang một cột ở 360px; kéo thả có nút Lên/Xuống thay thế (bàn phím, cảm ứng);
  trạng thái bằng chữ + biểu tượng; chỉ token.
- [x] **V. Dữ liệu người học**: bài học không phải dữ liệu cá nhân; mọi API quản trị qua `RequireAdmin`, audio qua
  `RequireAuth`; DB qua `lesson.Repository`, `lesson.RoadmapRepository`, `job.Repository`; `GEMINI_API_KEY` chỉ
  trong biến môi trường.
- [x] **VI. Kiểm thử**: unit test tách câu, validate bài, service lesson, worker (thành công, thử lại, quá số lần,
  revision cũ bị bỏ, khôi phục job running), client Kokoro/Gemini với server giả, kiểm tra kết quả AI; handler cho
  thành công, lỗi đầu vào, sai quyền. Không test nào cần mạng.
- [x] **VII. Đơn giản trước**: hàng đợi trong MongoDB + 1 goroutine (không Redis/queue); không SDK Gemini; một
  document lộ trình. Thêm 1 thư viện frontend (cdk) vì kéo thả truy cập được tự viết tốn công hơn nhiều.

**Kết quả**: ĐẠT, không có ngoại lệ.

## Project Structure

### Documentation (this feature)

```text
specs/003-lesson-admin/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   ├── admin-lessons-api.md   # /api/admin/lessons*, /api/admin/roadmap, /api/audio
│   ├── external-services.md   # Kokoro-FastAPI, Gemini generateContent
│   └── config.md              # biến môi trường mới
├── checklists/requirements.md
└── tasks.md                   # /speckit-tasks
```

### Source Code (repository root)

```text
backend/
├── cmd/api/main.go                    # + nối lesson, job worker, tts, ai, audio route
└── internal/
    ├── lesson/
    │   ├── model.go                   # Lesson, Sentence, Annotation, Level, Status, Roadmap, lỗi
    │   ├── repository.go              # Repository, RoadmapRepository
    │   ├── split.go (+ _test)         # SplitSentences
    │   ├── validate.go (+ _test)      # kiểm tra input bài, chú thích
    │   ├── annotate.go (+ _test)      # làm sạch kết quả AI, gắn sentenceIndex
    │   ├── service.go (+ _test)       # CRUD, retry, annotations, roadmap, xử lý job
    │   ├── handler.go (+ _test)       # /api/admin/lessons*, /api/admin/roadmap
    │   ├── audio.go (+ _test)         # GET /api/audio/{lessonId}/{rev}/{index}
    │   └── fake_test.go
    ├── job/
    │   ├── model.go                   # Job, Type, Status
    │   ├── repository.go              # Repository
    │   ├── worker.go (+ _test)        # vòng lặp, thử lại, backoff, khôi phục
    │   └── fake_test.go
    ├── tts/
    │   ├── tts.go                     # interface Synthesizer
    │   └── kokoro.go (+ _test)        # client Kokoro-FastAPI
    ├── ai/
    │   ├── ai.go                      # interface Provider, Annotation, ErrNotConfigured
    │   ├── gemini/gemini.go (+ _test) # client generateContent + responseSchema
    │   └── disabled.go                # Provider luôn trả ErrNotConfigured
    ├── admin/                         # xoá (ping tạm của F1 được thay bằng API thật)
    ├── platform/config/config.go      # + TTS_URL, TTS_VOICE, AUDIO_DIR, AI_PROVIDER, GEMINI_*
    └── storage/mongo/
        ├── lessons.go, roadmap.go, jobs.go
        └── indexes.go                 # + jobs {status, runAt}, lessons {createdAt}

frontend/src/app/
├── app.routes.ts                      # admin → loadChildren admin.routes
├── core/models/lesson.ts
├── features/admin/
│   ├── admin.routes.ts                # '', lessons/new, lessons/:id, lessons/:id/edit, roadmap
│   ├── admin-api.service.ts (+ spec)
│   ├── lesson-list/ (+ spec)          # danh sách, lọc, chip, Chạy lại, polling
│   ├── lesson-form/ (+ spec)          # tạo/sửa, cảnh báo chú thích sửa tay
│   ├── lesson-detail/ (+ spec)        # xem trước, nghe câu, bảng chú thích
│   ├── roadmap/ (+ spec)              # cdk drag-drop + Lên/Xuống, cảnh báo
│   └── status-chip/                   # chip trạng thái dùng chung
└── shared/utils/poll-while.ts (+ spec)

deploy/docker-compose.yml              # + kokoro, volume audio-data, env mới
```

**Structure Decision**: Theo `docs/architecture.md` (domain + ports). `admin` (F1 tạm) bị thay bằng API bài học;
trang `/admin` giờ là danh sách bài.

## Post-Design Constitution Check

Thiết kế Phase 1 thêm 5 interface (3 repository — nguyên tắc V; `Provider` — nguyên tắc III; `Synthesizer` —
để test không cần mạng theo nguyên tắc VI), 1 thư viện frontend, 1 container. Không có lớp trừu tượng nào khác.
**Vẫn ĐẠT.**

## Complexity Tracking

Không có vi phạm cần giải trình.
