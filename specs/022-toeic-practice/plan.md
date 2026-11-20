# Implementation Plan: Luyện thi TOEIC Nghe–Đọc (F27)

**Branch**: `022-toeic-practice` (làm trên `main`) | **Date**: 2026-11-18 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/022-toeic-practice/spec.md`

## Summary

Thêm khu **Luyện thi TOEIC** (Listening & Reading) theo kiểu trang luyện thi chuyên dụng: thư viện đề → chi tiết đề (Luyện
tập / Làm full test) → **phòng thi** (đồng hồ, bảng câu, văn bản | câu hỏi, phiếu trả lời trên điện thoại, phần Nghe đọc theo
kịch bản bằng giọng trình duyệt) → **trang kết quả** (điểm ước tính, phân loại câu hỏi) → **đáp án chi tiết**; kèm sổ câu
sai, luyện theo Part / dạng câu, thống kê, mục tiêu. Backend có package mới `internal/toeic`: kho cụm câu nhập từ JSON (chạy
thử, nháp, đăng, nhập lại để sửa và chấm lại), ghép đề có kiểm tra đủ câu và đủ ảnh, bài làm lưu trên server (làm tiếp trên
máy khác, không lộ đáp án trước khi nộp), chấm và quy đổi điểm, sổ câu sai, ảnh Part 1 vẽ bằng AI khi admin bấm. Frontend có
feature `toeic` với một phòng thi dùng chung cho mọi chế độ và trình phát kịch bản Nghe. Nội dung do Claude soạn trong
`content/toeic/`.

## Technical Context

**Language/Version**: Go 1.25 (backend), TypeScript strict + Angular 21 standalone/signals (frontend)

**Primary Dependencies**: thư viện chuẩn Go + driver Mongo/MySQL đang dùng; Angular; Web Speech API (qua `SpeechService`
có sẵn). **Không thêm thư viện mới.**

**Storage**: MongoDB và MySQL (chọn bằng cấu hình, `internal/storage/factory`). Mới: `toeic_groups`, `toeic_images`,
`toeic_tests` (nội dung, admin), `toeic_attempts`, `toeic_mistakes`, `toeic_goals` (người học). MySQL migration `047_toeic`.

**Testing**: `go test` (fake repository như `games/fake_test.go`, MySQL test qua `testdb_test.go`); Vitest qua `ng test`;
`ng lint`

**Target Platform**: web responsive: điện thoại từ 360 px (phòng thi xếp dọc) và màn rộng ≥ 1024 px (phòng thi chia cột);
sáng/tối

**Project Type**: web application (backend Go REST + frontend Angular)

**Performance Goals**: chọn đáp án / nhảy câu phản hồi ngay trên máy (không chờ mạng); `GET` đề 200 câu < 500 ms; nộp và
chấm 200 câu < 2 giây (SC-005); lưu tiến độ mỗi 5 giây không làm giật giao diện

**Constraints**: không gọi AI khi người học làm bài; không gửi đáp án/giải thích trước khi nộp; dev và production dùng
chung một DB (schema chỉ thêm); prompt AI bằng tiếng Anh; chữ giao diện bằng tiếng Việt; Directions do Luna tự viết; không
chép nội dung ETS

**Scale/Scope**: vài người dùng; kho khởi đầu ~2 đề đầy đủ + 2 Mini Test (~600 câu); ~12 màn người học, 2 màn admin

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- [x] **I. docs/ là nguồn sự thật**: *ngoại lệ có lý do* (Complexity Tracking): người dùng yêu cầu không sửa
  `docs/phases/giai-doan-3.md` và `docs/mvp-features.md`; nguồn sự thật của F27 là `docs/spec-inputs/f27-luyen-thi-toeic.md`
  + spec đã clarify. Spec ghi mã F27; mỗi tiêu chí nghiệm thu có bước kiểm tra ở `quickstart.md`.
- [x] **II. Chi phí 0 đồng**: không thêm dịch vụ/thư viện. Ảnh Part 1 dùng nhà cung cấp ảnh đã có (ngoại lệ trả phí cho ảnh
  đã được người dùng đồng ý từ F23), chỉ chạy khi admin bấm. Giọng đọc là giọng có sẵn của trình duyệt.
- [x] **III. AI là phần bổ sung**: mọi phần người học chạy không cần AI. AI chỉ vẽ ảnh Part 1 qua `ai.ImageProvider` (Gemini /
  Codex), có trạng thái và chạy lại được (ảnh lưu, Vẽ lại từng ảnh). *Ngoại lệ*: vẽ một ảnh chạy đồng bộ trong request admin
  như ảnh cảnh của bài; vẽ hàng loạt chạy ngầm trong process (không qua hàng đợi job) — xem Complexity Tracking.
- [x] **IV. Mobile-first**: phòng thi thiết kế cho 360 px trước (phiếu trả lời, thanh dưới, chạm ≥ 44 px), màn rộng là bố cục
  thêm; chỉ token của `docs/design-system.md`; đúng/sai có ✓/✗ ngoài màu; biểu đồ dựng bằng HTML có chữ, đọc được bằng trình
  đọc màn hình.
- [x] **V. Dữ liệu người học**: kiểm tra quyền ở API (`requireAuth`, kiểm tra khách trong service + `RequireMember`,
  `RequireAdmin`); bài làm/sổ/mục tiêu lọc theo `userID`; DB qua repository; 3 collection người học có trong file xuất và bị
  xoá cùng tài khoản.
- [x] **VI. Kiểm thử**: unit test cho validate nhập, chấm và quy đổi điểm, dựng đề ngẫu nhiên/theo tag, sổ câu sai, chấm lại
  khi nhập thay thế, kiểm tra đề, ẩn đáp án trong `PaperView`, quyền khách, giới hạn `remainingMs`; test handler (thành công,
  đầu vào sai, sai quyền); frontend test logic thuần (kịch bản Nghe, đồng hồ, bảng câu, hàng đợi lưu, quy đổi hiển thị). Không
  test nào cần mạng hay AI thật.
- [x] **VII. Đơn giản trước**: không thêm thư viện; không lưu file âm thanh; biểu đồ bằng HTML/CSS; đồng hồ dùng lại
  `game-clock.logic.ts`; tô vàng chỉ lưu trên máy; thống kê tính khi đọc.

*Re-check sau Phase 1*: `data-model.md`, `contracts/api.md`, `contracts/content-format.md` giữ nguyên các điểm trên. **Đạt.**

## Project Structure

### Documentation (this feature)

```text
specs/022-toeic-practice/
├── spec.md
├── plan.md                   # file này
├── research.md               # R1–R13
├── data-model.md             # thực thể, kiểm tra nhập, repository, quyền
├── quickstart.md             # cách chạy và các bước kiểm tra thủ công
├── contracts/api.md          # REST API + đường dẫn frontend
├── contracts/content-format.md  # định dạng JSON nội dung trong content/toeic/
├── checklists/requirements.md
└── tasks.md                  # tạo bởi /speckit-tasks
```

### Source Code (repository root)

```text
backend/internal/
├── toeic/                          # MỚI
│   ├── toeic.go                    # kiểu: Part, Group, Question, Doc, Graphic, Test, Attempt, Result, Mistake, Goal, lỗi
│   ├── tags.go                     # danh sách dạng câu cố định + GrammarPoints (R6)
│   ├── deps.go                     # interface: GroupRepository, ImageRepository, TestRepository, AttemptRepository,
│   │                               #   MistakeRepository, GoalRepository, StudyDays, Cards (vocab), Images (ai), Timezones
│   ├── validate.go                 # Validate nhóm theo Part (data-model), ValidationError
│   ├── importer.go                 # chạy thử / nhập / thay thế, báo cáo (FR-040, FR-040a)
│   ├── rescore.go                  # chấm lại bài cũ + cập nhật sổ câu sai khi thay thế (R8)
│   ├── tests.go                    # ghép đề, Check (đủ câu, đủ ảnh, phút Nghe), đăng
│   ├── paper.go                    # dựng đề cho lượt: đề cố định, ngẫu nhiên, theo Part/tag, làm lại câu sai (R9); PaperView ẩn đáp án
│   ├── attempts.go                 # tạo, lưu tiến độ (remainingMs chỉ giảm, khoá phần Nghe), nộp
│   ├── grade.go                    # chấm, ByQuestion, quy đổi điểm (R5), Counted/retake
│   ├── mistakes.go                 # sổ câu sai: applyMistake, lượt làm lại, check
│   ├── stats.go                    # thư viện (trạng thái đề), thống kê, theo tag
│   ├── images.go                   # vẽ ảnh Part 1 (engine bắt buộc), tải lên, hàng loạt chạy ngầm
│   ├── access.go                   # quyền khách (R10)
│   ├── service.go                  # Service + NewService(Deps)
│   ├── handler.go                  # route người học + admin
│   └── *_test.go, fake_test.go
├── ai/image.go                     # + PhotoBrief, ImageRequest.Photo
├── ai/llm/image.go, ai/codexcli/   # + prompt ảnh chụp Part 1, op "toeic_photo"
├── storage/storage.go              # + ToeicGroups(), ToeicImages(), ToeicTests(), ToeicAttempts(), ToeicMistakes(), ToeicGoals()
├── storage/mongo/toeic.go          # MỚI: 6 repository; chỉ số trong indexes.go
├── storage/mysql/toeic.go          # MỚI: 6 repository, toeicSchema(); migrate.go + 047_toeic
├── storage/{mongo,mysql}/users.go  # xoá toeic_attempts, toeic_mistakes, toeic_goals theo tài khoản
├── export/                         # + CollToeicAttempts, CollToeicMistakes, CollToeicGoals
└── service/toeic.go, routes.go, service.go  # initToeic (sau initVocab, initProgress, initSettings), adapter, route

frontend/src/app/
├── core/models/toeic.ts                        # MỚI: kiểu API
├── core/services/toeic-api.service.ts          # MỚI: gọi API + hàng đợi lưu tiến độ chưa gửi (localStorage)
├── core/services/speech.service.ts             # + opts { lang, natural }, voiceCount() (R3)
├── features/toeic/                             # MỚI
│   ├── toeic.routes.ts
│   ├── library/                                # /toeic: thẻ đề, tab, cột mục tiêu
│   ├── test-detail/                            # /toeic/tests/:key: Luyện tập (tick Part, giờ) / Full test / Kết quả
│   ├── exam-room/                              # /toeic/attempts/:id: phòng thi dùng chung
│   │   ├── exam-room.ts|html|css               # bố cục ≥1024 chia cột / <1024 xếp dọc
│   │   ├── question-grid.ts                    # bảng câu theo Part
│   │   ├── answer-sheet.ts                     # phiếu trả lời ô tròn (điện thoại)
│   │   ├── exam-state.logic.ts                 # đáp án, đánh dấu, vị trí, khoá phần, autosave (thuần)
│   │   └── directions.ts                       # màn Directions (lời Luna tự viết + tóm tắt tiếng Việt)
│   ├── listening/
│   │   ├── listening-script.logic.ts           # dựng đoạn đọc + khoảng nghỉ, ước tính thời lượng (thuần)
│   │   └── listening-player.ts                 # chạy kịch bản qua SpeechService, tạm dừng/nghe lại/tốc độ, chế độ thi
│   ├── shared/
│   │   ├── doc-view.ts                         # văn bản Part 6/7 theo loại (email, web, hoá đơn, chat…), tô vàng
│   │   ├── graphic-view.ts                     # bảng, lịch, bảng giá, sơ đồ tầng, biểu đồ cột (HTML/CSS)
│   │   ├── question-view.ts                    # câu hỏi + phương án A–D
│   │   └── score.logic.ts                      # nhãn điểm "ước tính", định dạng thời gian
│   ├── result/                                 # trang kết quả
│   ├── answer-detail/                          # đáp án chi tiết (transcript, dịch, giải thích, + Sổ từ)
│   ├── practice/                               # luyện theo Part, luyện theo dạng câu
│   ├── mistakes/                               # sổ câu sai + lượt làm lại (chấm ngay)
│   ├── results/                                # kết quả của tôi + thống kê
│   └── goal/                                   # mục tiêu
├── features/admin/toeic/                       # MỚI: danh sách cụm + nhập JSON (lu-json-source) + xem trước;
│   └── test-page/                              #   trang đề: ghép đề, kiểm tra, ảnh (lu-engine-dialog requireChoice)
├── features/admin/ai-usage/ai-usage-catalog.ts # + op toeic_photo
├── layouts/admin-layout/admin-layout.ts        # + mục "TOEIC"
├── layouts/learner-layout/learner-layout.ts    # /toeic vào isPracticePath; /toeic/attempts/:id là màn tập trung
├── layouts/learner-layout/practice-menu/practice-menu.ts  # + ô "TOEIC"
└── app.routes.ts                               # + /toeic (lazy), khách đi qua access trong service (không chặn cả route)

content/toeic/                                  # MỚI: JSON do Claude soạn (contracts/content-format.md)
```

**Structure Decision**: web application có sẵn (backend Go chia theo domain + ports, frontend Angular core/shared/features
theo `docs/architecture.md`). Toàn bộ logic TOEIC nằm trong package `toeic` mới; các package khác chỉ thêm đúng phần cần
(một trường ảnh AI, tuỳ chọn giọng của `SpeechService`, đăng ký storage/export/xoá tài khoản). `toeic` phụ thuộc vocab,
progress, settings, ai qua interface nhỏ, nối bằng adapter trong `internal/service` như `games`.

## Thứ tự làm (map với user story và đợt)

1. **Nền + US1 (nhập, ghép đề)**: kiểu, tags, validate, storage `toeic_groups`/`toeic_tests` (+ `toeic_images` để Check),
   importer, tests.Check, route admin, trang admin (danh sách, nhập JSON, xem trước, trang đề). Soạn lô nội dung Part 5–7 đầu
   tiên trong `content/toeic/` và nhập thử.
2. **US2 (phần Đọc)**: storage `toeic_attempts`, paper/PaperView, attempts (tạo/lưu/nộp), grade + quy đổi, stats thư viện,
   route người học; frontend thư viện, chi tiết đề, phòng thi (Part 5–7, bảng câu, phiếu trả lời, đồng hồ, autosave, hàng đợi),
   trang kết quả, đáp án chi tiết; lối vào Luyện; `MarkStudied`; export + xoá tài khoản cho `toeic_attempts`.
3. **US3 (sổ câu sai)**: storage `toeic_mistakes`, mistakes.go, cập nhật khi nộp, route; frontend sổ + lượt làm lại; "Làm lại
   các câu sai"; export + xoá tài khoản.
   → **Hết đợt 1**: phát hành được (tab Full test hiện "sắp có").
4. **US4 (phần Nghe)**: `SpeechService` opts, listening-script + player, Directions, Part 1–4 trong phòng thi, graphic-view;
   ảnh: `ai.PhotoBrief`, images.go, route ảnh, hàng loạt, op `toeic_photo`; soạn nội dung Part 1–4.
5. **US5 (full test, Mini Test, ngẫu nhiên)**: khoá phần, phát liền, `listeningSegment`, đồng hồ phần Đọc, đề ngẫu nhiên,
   Counted/retake, điểm thang 990; FR-040a chấm lại (rescore.go); soạn đủ 2 đề + 2 Mini Test.
6. **US6 (dạng câu, thống kê, mục tiêu)**: storage `toeic_goals`, stats/tags, route; frontend luyện theo dạng câu (link
   ngữ pháp), thống kê, mục tiêu; export + xoá tài khoản.
7. Quyền khách (access.go) làm cùng bước 2 và kiểm tra lại ở mỗi bước.

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| I: nguồn sự thật không nằm trong `docs/phases/` / `docs/mvp-features.md` | Người dùng yêu cầu rõ "đây là spec mới, không sửa vào spec cũ" | Ghi F27 vào `giai-doan-3.md` trái yêu cầu; file `docs/spec-inputs/f27-luyen-thi-toeic.md` + spec đã clarify là nguồn đầy đủ |
| III: vẽ một ảnh Part 1 chạy **đồng bộ** trong request admin; vẽ hàng loạt chạy ngầm trong process, không qua hàng đợi job | Giống ảnh cảnh của bài và ảnh Kho từ (`wordbank/batch.go`): admin bấm và xem ngay; ảnh được lưu nên không gọi lặp, có Vẽ lại | Hàng đợi job dùng chung giữa máy local và production (cùng DB) nên production có thể lấy job; Codex chỉ có ở máy local |
| IV: phòng thi có bố cục riêng cho màn rộng | Spec và phác thảo đã duyệt (kiểu trang luyện thi, máy tính chia cột) | Chỉ một bố cục điện thoại trên màn rộng thì văn bản Part 7 và câu hỏi không nhìn cùng lúc; vẫn giữ mobile-first (thiết kế 360 px trước) |
| Constitution "mỗi tính năng một nhánh riêng" | Người dùng làm các spec 018–021 thẳng trên `main` | Tạo nhánh sẽ mang theo các thay đổi chưa commit không liên quan |
