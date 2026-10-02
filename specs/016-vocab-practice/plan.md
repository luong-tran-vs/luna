# Implementation Plan: Trang chi tiết bài và luyện tập từ vựng

**Branch**: `016-vocab-practice` | **Date**: 2026-10-02 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/016-vocab-practice/spec.md` + khối 2 trong `docs/spec-inputs/f17-luyen-tap-tu-vung.md`

## Summary

**Backend.** `ai.Provider.Practice` gửi 1 request Gemini riêng (JSON mode + `responseSchema`), không đổi `Annotate`. Chuỗi job nền:

1. `ProcessAnnotate` lưu chú thích thành công → xoá phần luyện tập cũ, đặt `practiceStatus = running`, xếp hàng job `practice`.
2. `ProcessPractice`: gọi AI 1 lần → `CleanPractice` bỏ phần hỏng → `SavePractice` có điều kiện revision + version → xếp hàng
   `practice_audio`.
3. `ProcessPracticeAudio`: tạo mp3 cho câu ví dụ, từng lượt hội thoại, đáp án câu dịch vào
   `{audioDir}/{id}/{rev}/practice/{ver}/{kind}-{i}.mp3`; lỗi chỉ retry, không gọi lại AI.

Phần luyện tập lưu trong tài liệu bài (`practice`, `practiceStatus`, `practiceError`, `practiceVersion`); sửa nội dung bài thì xoá.
`GET /api/lessons/{id}/practice` (guard L) trả nội dung, `lessonNumber`, ô trống bước 3 do backend tính (`BuildFill`), ô từ câu dịch,
`audioUrl`. Quản trị viên: `POST /api/admin/lessons/{id}/practice/regenerate`, `GET` bài quản trị trả thêm phần luyện tập.

**Frontend.** Viết lại `features/lesson/lesson-detail` thành trang 4 bước (từ vựng, hội thoại, điền ô trống, ghép câu dịch) với thanh
trên, đầu trang, tab Bài học/Bài đọc, nút Tiếp theo cố định trên thanh tab đáy, tổng kết và Làm lại. Chấm ngay ở trình duyệt, không gửi
kết quả. Trang quản trị bài thêm mục "Phần luyện tập".

## Technical Context

**Language/Version**: Go 1.25.1 (`GOTOOLCHAIN=local`); TypeScript 5.9 strict + Angular 21

**Primary Dependencies**: không thêm thư viện. Gemini `generateContent`, Kokoro TTS (F4), `lu-audio-player` sẵn có.

**Storage**: MongoDB `lessons` thêm 4 trường; `jobs` thêm 2 loại; file mp3 dưới thư mục revision của bài. Không collection, không index mới.

**Testing**:

- Go: `lesson` (`CleanPractice`, `BuildFill`, tách ô, job practice/practice_audio, regenerate, handler + guard, route audio),
  `ai/gemini` (`Practice` với fake server), `storage/mongo` (chuyển đổi doc).
- Vitest: `practice-logic`, `lesson-detail`, `vocab-step`, `dialogue-step`, `fill-step`, `translate-step`, admin `practice-section`,
  `audio-player` (`timeupdate`), `reading-api`, `admin-api`.

**Target Platform / Project Type**: web app (Go REST + Angular SPA) chạy bằng Docker Compose, như các phần trước

**Performance Goals**:

- Phần luyện tập có trong ≤ 5 phút sau khi chú thích xong (SC-001): 1 request AI (timeout HTTP 90 giây) + TTS khoảng 20–30 câu ngắn.
- `GET /practice` < 300 ms: đọc 1 bài, tính ô trống trong bộ nhớ, `os.Stat` ≤ ~40 file.

**Constraints**:

- Đúng 1 request AI có kết quả cho mỗi lần sinh (SC-003): nội dung hỏng hết → failed, không retry; audio tách job.
- AI lỗi không chạm chú thích, câu hỏi, đề viết, các bước của bài (FR-008).
- Không ghi tiến độ, streak, thống kê (FR-020); không endpoint nộp kết quả (FR-019).

**Scale/Scope**: vài người học, vài trăm bài; mỗi bài ≤ 30 từ gửi AI, ≤ 10 lượt, ≤ 5 câu dịch, ≤ 5 ô trống.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

Đối chiếu `.specify/memory/constitution.md` v1.0.0.

- [x] **I. docs/ là nguồn sự thật**: F17 theo `docs/phases/giai-doan-3.md` (9 tiêu chí nghiệm thu, đều có test hoặc bước trong
  quickstart). Bố cục theo `design/screen1-3.png`, màu/font theo `docs/design-system.md`. Code theo `docs/architecture.md`: domain
  `internal/lesson`, port AI/DB, `features/lesson`, `features/admin`.
- [x] **II. Chi phí 0 đồng**: Gemini gói miễn phí, 1 request mỗi lần sinh; TTS Kokoro chạy trên máy; không thư viện mới.
- [x] **III. AI là phần bổ sung**: gọi qua `ai.Provider`; chạy nền bằng job có trạng thái `running/done/failed`; quản trị viên Tạo lại
  được; kết quả lưu, không gọi lặp. Không có phần luyện tập thì trang vẫn hiện từ vựng, tab Bài đọc, và luồng học hằng ngày không đổi.
- [x] **IV. Mobile-first**: 360px không cuộn ngang (ô từ `flex-wrap`), sáng/tối, chỉ token; đúng/sai có chữ + biểu tượng; bàn phím đầy
  đủ, kết quả `role="status"`, vùng chạm ≥ 44px.
- [x] **V. Dữ liệu người học**: `GET /practice` qua guard L ở API (403 bài sắp tới); admin route qua `RequireAdmin`; audio qua
  `requireAuth`; phần luyện tập không chứa dữ liệu người học, kết quả luyện tập không lên server; DB qua `lesson.Repository`.
- [x] **VI. Kiểm thử**: logic thuần (`CleanPractice`, `BuildFill`, chấm, xáo) có unit test; endpoint có test thành công / lỗi đầu vào
  (route audio) / sai quyền (403, không phải admin); AI, TTS giả lập (R13).
- [x] **VII. Đơn giản trước**: lưu trong tài liệu bài, không collection mới; ô trống tính lúc đọc, không lưu; `audioUrl` bằng
  `os.Stat`; không thêm thư viện; component con chỉ trong thư mục trang.

**Kết quả**: ĐẠT, không có ngoại lệ.

## Project Structure

### Documentation (this feature)

```text
specs/016-vocab-practice/
├── plan.md, research.md, data-model.md, quickstart.md
├── contracts/practice-api.md
├── checklists/requirements.md
└── tasks.md                                     # /speckit-tasks
```

### Source Code (repository root)

```text
backend/
├── cmd/api/main.go                              # handler practice/practice_audio cho worker; onFailed; NewReader(..., cfg.AudioDir);
│                                                #   lessonTopicsPort.Position
└── internal/
    ├── ai/ai.go                                 # PracticeRequest, Practice…, Provider.Practice, Disabled
    ├── ai/gemini/gemini.go (+ _test)            # practiceSchema, practicePrompt, Practice
    ├── job/model.go                             # TypePractice, TypePracticeAudio
    ├── lesson/
    │   ├── model.go                             # Lesson.Practice*, StatusNone, StatusOf switch, lỗi mới, Topics.Position
    │   ├── repository.go                        # SavePractice
    │   ├── practice.go (+ _test)                # MỚI: Practice, CleanPractice, containsWord, BuildFill, answerTiles
    │   ├── practice_job.go (+ _test)            # MỚI: ProcessPractice, ProcessPracticeAudio, RegeneratePractice, practiceWords
    │   ├── practice_handler.go (+ _test)        # MỚI: GET /api/lessons/{id}/practice, PracticeAudioHandler
    │   ├── service.go                           # ProcessAnnotate enqueue practice; JobFailed 2 loại mới
    │   ├── reading.go                           # Reader.audioDir, Reader.Practice (PracticeView)
    │   ├── reading_handler.go                   # đăng ký route practice (guard L)
    │   ├── handler.go                           # POST .../practice/regenerate, lessonJSON + practice, writeError 409 mới
    │   └── fake_test.go                         # fakeAI.Practice, fake SavePractice, fakeTopics.Position
    ├── writing/fake_test.go                     # fakeGrader.Practice
    └── storage/mongo/
        └── lessons.go (+ _test)                 # practiceDoc, SavePractice, SaveAnnotations/ReplaceContent/SetStatus

frontend/src/app/
├── core/models/practice.ts                      # MỚI: PracticeStatus, PracticeView, AdminPractice
├── core/models/lesson.ts                        # Lesson + practice*, isRunning
├── features/lesson/reading-api.service.ts (+ spec)   # practice(id)
├── features/lesson/lesson-detail/               # VIẾT LẠI
│   ├── lesson-detail.ts/.html/.css (+ spec)     # thanh trên, đầu trang, tab, bước, Tiếp theo, tổng kết
│   ├── practice-logic.ts (+ spec)               # xáo có seed, chấm, nhãn mức độ, tổng kết
│   ├── vocab-step/ (+ spec)
│   ├── dialogue-step/ (+ spec)                  # phát cả đoạn, tốc độ, từng lượt, bật/tắt nghĩa
│   ├── fill-step/ (+ spec)                      # ô trống, ngân hàng từ, Kiểm tra, mẹo ngữ pháp
│   ├── translate-step/ (+ spec)                 # ghép ô, Làm lại, Kiểm tra
│   └── practice-summary/                        # tổng kết, Làm lại, nút về bài
├── features/admin/admin-api.service.ts (+ spec)      # regeneratePractice(id)
├── features/admin/lesson-detail/lesson-detail.html   # chèn <lu-practice-section>
├── features/admin/lesson-detail/practice-section/ (+ spec)  # MỚI
├── shared/components/audio-player/ (+ spec)     # output timeupdate
└── shared/components/icon/icon.ts               # play, pause, volume

README.md                                        # mục trang chi tiết bài + phần luyện tập
docs/phases/giai-doan-3.md                       # tick tiêu chí F17 khi xong
```

**Structure Decision**:

- Phần luyện tập là dữ liệu của bài theo revision, sinh bằng job như chú thích → nằm trong `internal/lesson` (`Service` cho job/quản trị,
  `Reader` cho người học). Không tách gói mới vì dùng chung `Lesson`, `current()`, `writeFileAtomic`, guard và route audio.
- Frontend: trang và các bước chỉ dùng ở `/lessons/:id` nên ở trong `features/lesson/lesson-detail/`; chỉ sửa `shared` ở chỗ thật sự
  dùng chung (`audio-player`, `icon`).

## Post-Design Constitution Check

- Thêm 1 method Provider, 2 loại job, 3 route, 4 trường bài; không thư viện, không collection mới.
- Các chỗ lệch khối 2 đều làm đơn giản hơn hoặc đúng nguyên tắc hơn, ghi ở research: trạng thái `running` thay `pending` (R2), audio
  thành job riêng (R7, giữ 1 request AI), đường dẫn audio thêm `{version}` (R7, tránh cache cũ).

**Vẫn ĐẠT.** Không có mục Complexity Tracking.
