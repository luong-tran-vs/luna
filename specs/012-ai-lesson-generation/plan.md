# Implementation Plan: AI sinh bài học

**Branch**: `012-ai-lesson-generation` | **Date**: 2026-10-01 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/012-ai-lesson-generation/spec.md` + khối 2 trong `docs/spec-inputs/f7-ai-sinh-bai.md`

**Note**: This template is filled in by the `/speckit-plan` command. See `.specify/templates/plan-template.md` for the execution workflow.

## Summary

`ai.Provider` có thêm `GenerateLessons`. Client Gemini gửi **một request cho cả lượt**, dùng `responseSchema` để nhận một mảng
`{title, content}`. Prompt nêu trình độ CEFR, độ dài, dạng bài (hội thoại viết dạng `Tên: câu`) và các tiêu đề đã có trong chủ đề.

Route `POST /api/admin/topics/{id}/generate` (chỉ admin, đồng bộ, tối đa 60 giây) do `lesson.Service.Generate` xử lý:
- kiểm tra đầu vào;
- lấy chủ đề và tiêu đề các bài của chủ đề;
- gọi AI;
- lọc bản rỗng, trùng tiêu đề, sai độ dài ±20% hoặc quá giới hạn F2;
- trả bản nháp về client, không ghi DB.

Lưu bản nháp dùng lại `POST /api/admin/lessons`:
- nguồn "AI sinh", giấy phép "Nội dung do AI tạo";
- cờ mới `appendToRoadmap=true` nối bài vào cuối lộ trình trong cùng request (nối lỗi thì xoá bài vừa tạo);
- sau đó tạo job TTS và chú thích như F2.

Lỗi AI được ánh xạ như sau:
- 503 `ai_not_configured`: chưa cấu hình AI hoặc khoá sai;
- 429 `ai_quota`: hết lượt;
- 502 `ai_unusable` / `ai_failed`: nội dung không dùng được / lỗi khác.

Request này cần chạy lâu hơn mức 15 giây hiện có, nên nới thời gian riêng cho một route này:
- nginx: thêm location riêng `proxy_read_timeout 75s`;
- Go: `SetWriteDeadline` ngay trong handler.

Frontend, trang `admin/roadmap`:
- nút **Sinh bài bằng AI** mở `lu-generate-dialog` (`<dialog>` gốc);
- bản nháp giữ trong signal của trang và hiển thị bằng `lu-draft-list`: sửa, Lưu, Bỏ, Lưu tất cả (tuần tự), số từ;
- `unsavedChangesGuard` cùng `beforeunload` và hỏi xác nhận khi đổi chủ đề lúc còn bản nháp.

## Technical Context

**Language/Version**: Go 1.25.1 (`GOTOOLCHAIN=local`); TypeScript 5.9 strict + Angular 21 (standalone, signals, zoneless, OnPush)

**Primary Dependencies**:
- Không thêm thư viện.
- Gemini `generateContent` REST, đã dùng ở F2, model mặc định Flash-Lite theo cấu hình hiện có.

**Storage**:
- Không collection, index hay migration mới.
- Ghi vào `lessons` và `topics.lessonIds` qua repository có sẵn (`Create`, `AppendLesson`, `Delete`).

**Testing**:
- Go `testing`:
  - `ai/gemini` với httptest: schema, prompt, 429, JSON hỏng.
  - `lesson` với Provider giả: lọc, `ErrUnusableDraft`, kiểm tra đầu vào, timeout, tiêu đề đã có.
  - `Create` + `appendToRoadmap`: nối cuối; lỗi nối thì xoá bài và không tạo job.
  - handler: 200, 400, 403, 404, 503, 429, 502.
- Vitest:
  - dialog: mặc định theo trình độ, kiểm tra đầu vào, đang sinh, lỗi giữ giá trị.
  - trang lộ trình: lưu từng bài, lưu tất cả, bỏ, lỗi AI giữ bản nháp, lượt sau nối thêm.
  - guard.

**Target Platform / Project Type**: web app như F0–F13 (Docker compose: nginx + Go API + MongoDB + Kokoro)

**Performance Goals**:
- Một lượt ≤ 60 giây (SC-006). Thường 10–30 giây với 5 × 400 từ trên Flash-Lite.
- Lưu một bài < 1 giây (TTS và chú thích chạy nền như F2).

**Constraints**:
- Chỉ admin.
- 1 request AI mỗi lượt (FR-020).
- Bản nháp không bao giờ ghi DB (FR-008).
- AI lỗi không ảnh hưởng chức năng khác (nguyên tắc III).
- 360px, chỉ bàn phím (nguyên tắc IV).

**Scale/Scope**: 1 quản trị viên, vài lượt sinh mỗi tuần, ≤ 5 bài × 800 từ mỗi lượt

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

Đối chiếu `.specify/memory/constitution.md` v1.0.0.

- [x] **I. docs/ là nguồn sự thật**:
  - F7 theo `docs/phases/giai-doan-2.md` (4 tiêu chí nghiệm thu) và `docs/mvp-features.md` v9.
  - Mã nằm trong `internal/ai`, `internal/lesson`, `internal/topic`, `features/admin` đúng `docs/architecture.md`.
- [x] **II. Chi phí 0 đồng**:
  - Gemini gói miễn phí, 1 request mỗi lượt.
  - Không dịch vụ mới.
- [ ] **III. AI là phần bổ sung**:
  - Đạt:
    - gọi qua `ai.Provider`;
    - đổi nhà cung cấp bằng cấu hình;
    - AI lỗi chỉ ảnh hưởng nút sinh bài;
    - có trạng thái đang sinh / lỗi và thử lại được;
    - bài đã lưu dùng job nền của F2.
  - **Ngoại lệ**: lượt sinh chạy đồng bộ trong request, không phải job nền; bản nháp chỉ giữ ở client, không lưu ở server.
  - Giải trình ở Complexity Tracking.
- [x] **IV. Mobile-first**:
  - Dialog và danh sách bản nháp một cột ở 360px, nút ≥ 44px.
  - `<dialog>` gốc giữ focus, Escape đóng được.
  - Trạng thái đang sinh / lỗi / đã lưu có `role="status"` / `role="alert"`.
  - Đúng/sai không chỉ dựa vào màu.
- [x] **V. Dữ liệu người học**:
  - Không đọc hay gửi dữ liệu người học cho AI; chỉ gửi tên chủ đề, trình độ và tiêu đề bài.
  - Log không ghi nội dung hay khoá.
- [x] **VI. Kiểm thử**: lọc bản nháp, kiểm tra đầu vào, ánh xạ lỗi, nối lộ trình có bù trừ, client Gemini và giao diện đều có test
  (R9).
- [x] **VII. Đơn giản trước**:
  - Dùng lại POST lesson thay vì endpoint lưu riêng.
  - Không collection, job, polling.
  - Hai component nhỏ và một guard dùng chung.

**Kết quả**: ĐẠT có một ngoại lệ (nguyên tắc III, chạy nền), đã giải trình bên dưới.

## Project Structure

### Documentation (this feature)

```text
specs/012-ai-lesson-generation/
├── plan.md, research.md, data-model.md, quickstart.md
├── contracts/generate-api.md
├── checklists/requirements.md
└── tasks.md                                   # /speckit-tasks
```

### Source Code (repository root)

```text
backend/
├── cmd/api/main.go                            # lessonTopicsPort.AppendLesson
└── internal/
    ├── ai/
    │   ├── ai.go                              # + LessonKind, GenerateRequest, LessonDraft, Provider.GenerateLessons, Disabled
    │   └── gemini/gemini.go (+ _test)         # + GenerateLessons, generateSchema, generatePrompt; tách c.generate dùng chung
    ├── lesson/
    │   ├── generate.go (+ _test)              # MỚI: GenerateInput, ValidateGenerate, Draft, GenerateResult, Service.Generate, lọc
    │   ├── generate_handler.go (+ _test)      # MỚI: POST /api/admin/topics/{id}/generate, ánh xạ lỗi AI, SetWriteDeadline
    │   ├── model.go                           # + ErrUnusableDraft
    │   ├── repository.go                      # Topics + AppendLesson
    │   ├── validate.go                        # Input + AppendToRoadmap
    │   ├── service.go (+ _test)               # Create: append + bù trừ, enqueue sau cùng
    │   ├── handler.go                         # inputJSON + appendToRoadmap; Register thêm route generate
    │   └── fake_test.go                       # Provider giả có GenerateLessons; Topics giả có AppendLesson
    └── topic/service.go (+ _test)             # + AppendLesson

frontend/
├── nginx.conf                                 # + location generate, proxy_read_timeout 75s
└── src/app/
    ├── core/
    │   ├── models/generate.ts (+ spec)        # MỚI: LessonKind, GenerateInput/Result, Draft, DEFAULT_WORDS, countWords, AI_SOURCE/LICENSE
    │   ├── models/lesson.ts                   # LessonInput.appendToRoadmap?
    │   └── guards/unsaved-changes.guard.ts (+ spec)   # MỚI: CanDeactivateFn<CanLeave>
    └── features/admin/
        ├── admin-api.service.ts (+ spec)      # + generateLessons
        ├── admin.routes.ts                    # roadmap: canDeactivate
        ├── generate-dialog/ (+ spec)          # MỚI: <dialog> form số bài, độ dài, dạng bài, ý chính
        ├── draft-list/ (+ spec)               # MỚI: danh sách bản nháp (sửa, số từ, Lưu, Bỏ, Lưu tất cả)
        └── roadmap/ (+ spec)                  # nút sinh, state drafts, lưu, canLeave, beforeunload, đổi chủ đề

README.md                                      # + mục "Sinh bài bằng AI"
docs/phases/giai-doan-2.md                     # tick tiêu chí F7 khi xong
```

**Structure Decision**:
- Việc sinh bài nằm trong package `lesson`, vì package này đã giữ `ai.Provider`, cổng `Topics` và repository bài học (cần để lấy tiêu
  đề đã có và giới hạn F2).
- Route nằm dưới `/api/admin/topics/{id}` theo khối 2 nhưng do `lesson.Handler` đăng ký; `topic` không phụ thuộc AI.
- `topic` chỉ có thêm `AppendLesson`, lộ ra qua cổng cho `lesson`.
- Phía frontend tách hai component con để trang lộ trình không phình quá; state vẫn ở trang theo khối 2.

## Post-Design Constitution Check

- Một method mới trên interface Provider, một route, một cờ trên route có sẵn, một location nginx.
- Hai component, một guard.
- Không collection, job, thư viện hay cấu hình mới.
- Ngoại lệ nguyên tắc III không đổi.

**Vẫn ĐẠT (có ngoại lệ đã giải trình).**

## Complexity Tracking

| Điểm | Vì sao cần | Phương án đơn giản hơn / đúng nguyên tắc hơn bị loại vì |
| --- | --- | --- |
| Nguyên tắc III: sinh bài chạy đồng bộ trong request (≤ 60 giây), bản nháp chỉ ở client, không job nền, không lưu kết quả | Quản trị viên đang chờ trên trang để duyệt ngay. Spec không cho lưu tự động (FR-008) và chấp nhận mất bản nháp khi rời trang. Vẫn có trạng thái đang sinh / lỗi, thử lại được, và AI lỗi không chạm chức năng khác | Job nền + collection `lesson_drafts` + polling cần thêm collection, job type, dọn bản nháp cũ và màn hình trạng thái cho một thao tác quản trị vài lần mỗi tuần (trái nguyên tắc VII). Gọi lặp chỉ xảy ra khi quản trị viên chủ động bấm sinh lại |
| Nới thời gian cho riêng route sinh bài (nginx 75s, `SetWriteDeadline` 75s) | Server và nginx cắt request ở 15 giây, mà AI cần 10–30 giây | Nới toàn cục sẽ làm mọi request treo lâu hơn; streaming phức tạp hơn nhiều |
| Bù trừ khi lưu (xoá bài nếu nối lộ trình lỗi) | MongoDB chạy standalone, không có transaction, mà FR-012 cấm để lại bài chưa vào lộ trình | Chuyển sang replica set chỉ cho một thao tác thì quá tay |
