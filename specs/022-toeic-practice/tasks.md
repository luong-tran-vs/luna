---

description: "Task list for F27 Luyện thi TOEIC (Nghe–Đọc)"
---

# Tasks: Luyện thi TOEIC Nghe–Đọc (F27)

**Input**: `specs/022-toeic-practice/` — spec.md (US1–US6, FR-001…FR-048, clarify 2026-11-18), plan.md, research.md
(R1–R13), data-model.md, contracts/api.md, contracts/content-format.md, quickstart.md. Phác thảo:
`design/toeic/de-xuat-luyen-thi-toeic-2.png`.

**Tests**: BẮT BUỘC theo constitution VI: unit test cho logic nghiệp vụ, test endpoint (thành công, đầu vào sai, sai quyền),
không cần mạng (AI giả lập, không gọi giọng đọc thật). Mỗi tiêu chí nghiệm thu có test hoặc bước kiểm tra ở quickstart.md.

**Quy ước của dự án (đọc trước khi làm)**:
- Tên trong code, API, commit bằng tiếng Anh; chữ giao diện, tài liệu bằng tiếng Việt. Prompt gửi AI bằng tiếng Anh.
- Backend: domain package + interface cho DB/AI, DI bằng constructor viết tay, nối trong `backend/internal/service/`. Làm
  repository cho **cả** Mongo (`internal/storage/mongo`, id ObjectID, `userId` ObjectID) và MySQL (`internal/storage/mysql`,
  `CHAR(24)` từ `newID()`, JSON ở cột `TEXT NOT NULL`, `tableOptions`). Dev và production chung một DB: chỉ thêm, không đổi
  hay xoá cái đang có.
- Frontend: Angular 21 standalone, signals, `input()`/`output()`, `ChangeDetectionStrategy.OnPush`, `@if/@for`, không
  `ngClass/ngStyle`; màu, chữ chỉ dùng token trong `frontend/src/styles/tokens.css`.
- Không sửa spec cũ và tài liệu giai đoạn cũ (`docs/phases/*`, `docs/mvp-features.md`); tài liệu sản phẩm của F27 là
  `docs/spec-inputs/f27-luyen-thi-toeic.md`.
- Không chép nội dung đề ETS; Directions do Luna tự viết. Không tự commit.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: làm song song được (khác file, không phụ thuộc task chưa xong)
- **[Story]**: US1 … US6 theo spec.md

---

## Phase 1: Setup

**Purpose**: kiểu dùng chung, danh sách dạng câu, khung thư mục

- [X] T001 Tạo package `backend/internal/toeic/` với `toeic.go`: kiểu `Part`, `PartQuestions`, `Section`, `Status`, `TestKind`, `Mode`, `SourceKind`, `Accent`, `Group`, `Turn`, `Graphic`, `GraphicCell`, `Bar`, `Doc`, `KV`, `Block`, `Table`, `Question`, `Ref`, `Vocab`, `Test`, `TestCheck`, `Attempt`, `Source`, `Result`, `SectionScore`, `Mistake`, `Goal`, `Image`, hàm `QuestionID(key string, no int) string`, `SectionOf(Part) Section`, và lỗi `ErrNotFound`, `ErrExists`, `ErrMembersOnly`, `ErrAttemptInProgress{ID}`, `ErrAttemptSubmitted`, `ErrNotEnough`, `ErrTestNotReady{Check}`, `ValidationError{Fields map[string]string}` (thông báo tiếng Việt) đúng như `specs/022-toeic-practice/data-model.md`
- [X] T002 [P] Tạo `backend/internal/toeic/tags.go`: danh sách cố định `Tags []Tag{Code, LabelVi, Parts []Part, GrammarPoints []string}` theo research R6, hàm `TagByCode(code) (Tag, bool)`, `TagAllowed(code string, p Part) bool`; test `backend/internal/toeic/tags_test.go`: mã không trùng, mỗi tag có ít nhất một Part, mọi `GrammarPoints` có trong `grammar.Default()`
- [X] T003 [P] Tạo `frontend/src/app/core/models/toeic.ts`: kiểu TypeScript cho mọi request/response trong `specs/022-toeic-practice/contracts/api.md` (`ToeicLibrary`, `TestCard`, `TestDetail`, `StartAttempt`, `AttemptView`, `PaperGroup`, `PaperQuestion`, `ToeicDoc`, `DocBlock`, `Graphic`, `Turn`, `SaveProgress`, `ResultView`, `TagRow`, `AnswerRow`, `QuestionDetail`, `MistakeItem`, `TagStat`, `ToeicStats`, `ToeicGoal`, `ImportReport`, `GroupAdmin`, `TestAdmin`, `TestCheck`)
- [X] T004 [P] Tạo thư mục nội dung `content/toeic/{part1,part2,part3,part4,part5,part6,part7,tests}/` với `content/toeic/README.md` (tiếng Việt, ngắn): trỏ tới `specs/022-toeic-practice/contracts/content-format.md`, quy tắc soạn (không chép ETS, tên hư cấu, mỗi câu một đáp án, `explainVi` nêu bẫy)

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: cổng phụ thuộc, storage, quyền, nối service, lối vào frontend

**⚠️ CRITICAL**: xong phase này mới làm các user story

### Backend

- [X] T005 Viết `backend/internal/toeic/deps.go`: interface `GroupRepository`, `ImageRepository`, `TestRepository`, `AttemptRepository`, `MistakeRepository`, `GoalRepository` đúng chữ ký ở data-model.md (thêm `TestRepository{ List(ctx, onlyPublished bool) ([]Test, error); Get(ctx, key) (Test, error); Save(ctx, Test) error; SetStatus(ctx, key, Status) error }`), và cổng ra ngoài: `StudyDays{MarkStudied(ctx, userID) error}`, `Cards{SaveWord(ctx, userID string, w CardInput) error}` (bọc `vocab.Service.Save`, coi `*vocab.ExistsError` là thành công), `Photos{Draw(ctx, engine string, brief PhotoBrief) (mime string, data []byte, err error); CodexAvailable() bool}`, `Timezones{Location(ctx, userID) (*time.Location, error)}`
- [X] T006 Viết `backend/internal/toeic/service.go`: `Deps` (các repo + cổng ở T005 + `Now func() time.Time` + `Rand *rand.Rand` gieo được), `type Service`, `NewService(Deps) *Service`
- [X] T007 [P] Viết `backend/internal/toeic/fake_test.go`: fake trong bộ nhớ cho 6 repository và các cổng ở T005, dùng chung cho mọi test của package
- [X] T008 Viết `backend/internal/toeic/access.go` theo research R10: `func (s *Service) guestTestKey(ctx) (string, error)` (Mini Test đã đăng có `Order` nhỏ nhất), `func (s *Service) checkStart(ctx, isGuest bool, src Source, parts []Part) error` (khách: chỉ đề `guestTestKey` hoặc `SourceKind=part` với Part 1, 2, 5; còn lại `ErrMembersOnly`), `locked(test Test, isGuest bool) bool`
- [X] T009 [P] Test `backend/internal/toeic/access_test.go`: khách làm Mini Test thứ tự nhỏ nhất được, Mini Test khác/đề full → `ErrMembersOnly`, luyện Part 2 được, Part 3 → lỗi, `random`/`tag`/`mistakes` → lỗi; thành viên mọi thứ được
- [X] T010 Thêm vào `backend/internal/storage/storage.go` các getter `ToeicGroups() toeic.GroupRepository`, `ToeicImages() toeic.ImageRepository`, `ToeicTests() toeic.TestRepository`, `ToeicAttempts() toeic.AttemptRepository`, `ToeicMistakes() toeic.MistakeRepository`, `ToeicGoals() toeic.GoalRepository`
- [X] T011 Mongo `backend/internal/storage/mongo/toeic.go`: repository cho `toeic_groups` (`Upsert` theo `key`, giữ `status`/`createdAt` khi thay thế, tăng `version`), `toeic_images` (khoá `(groupKey, slot)`), `toeic_tests` (khoá `key`), `toeic_attempts`, `toeic_mistakes`, `toeic_goals` (khoá `userId`); chỉ mục trong `backend/internal/storage/mongo/indexes.go`: `toeic_groups.key` duy nhất + `(part, status)`, `toeic_images (groupKey, slot)` duy nhất, `toeic_tests.key` duy nhất, `toeic_attempts (userId, status, startedAt)`, `(userId, source.testKey)`, `groupKeys`, `toeic_mistakes (userId, questionId)` duy nhất; nối trong `backend/internal/storage/mongo/store.go`. Hằng tên collection người học lấy từ `export` (T014)
- [X] T012 [P] MySQL `backend/internal/storage/mysql/toeic.go`: `toeicSchema()` tạo 6 bảng (`toeic_groups(id, group_key UNIQUE, part, level, status, source, version, body TEXT, created_at, updated_at)`, `toeic_images(group_key, slot, mime, data MEDIUMBLOB, origin, created_at, PRIMARY KEY(group_key, slot))`, `toeic_tests(id, test_key UNIQUE, title, kind, ord, status, group_keys TEXT, …)`, `toeic_attempts(id, user_id, status, test_key, mode, body TEXT, started_at, submitted_at, INDEX(user_id, status), INDEX(user_id, test_key))`, `toeic_attempt_groups(attempt_id, group_key, PRIMARY KEY(attempt_id, group_key), INDEX(group_key))` để chấm lại, `toeic_mistakes(user_id, question_id, …, PRIMARY KEY(user_id, question_id))`, `toeic_goals(user_id PRIMARY KEY, target, exam_date, updated_at)`); 6 repository; migration `{ID: "047_toeic", Stmts: toeicSchema()}` trong `backend/internal/storage/mysql/migrate.go`; nối trong `backend/internal/storage/mysql/mysql.go`
- [X] T013 [P] Test MySQL `backend/internal/storage/mysql/toeic_test.go` theo kiểu `games_test.go`: upsert/thay thế cụm giữ status, ảnh upsert, bài làm tạo/lưu/nộp/`WithGroup`, sổ câu sai duy nhất theo `(user, question)`, mục tiêu upsert
- [X] T014 Đưa 3 collection người học vào file xuất và xoá tài khoản: `CollToeicAttempts`, `CollToeicMistakes`, `CollToeicGoals` + trường `ToeicAttempts`, `ToeicMistakes`, `ToeicGoals []Doc` (`toeicAttempts`, `toeicMistakes`, `toeicGoals`) và `Allowed()` trong `backend/internal/export/model.go`; danh sách trong `backend/internal/export/service.go`; khoá MySQL trong `backend/internal/storage/mysql/export.go`; `userData` trong `backend/internal/storage/mongo/users.go` và `backend/internal/storage/mysql/users.go` (MySQL xoá cả `toeic_attempt_groups` của các bài của người đó); sửa test export liên quan
- [X] T015 Viết `backend/internal/toeic/handler.go`: `NewHandler(svc, log)`, `Register(mux, requireAuth)` với ba lớp `auth`, `member` (`httpx.RequireMember`), `admin` (`httpx.RequireAdmin`); hàm `fail(w, err)` map lỗi: `ValidationError` → `httpx.WriteFieldErrors`, `ErrMembersOnly` → `403 members_only`, `ErrNotFound` → `404 not_found`, `ErrAttemptInProgress` → `409 attempt_in_progress {attemptId}`, `ErrAttemptSubmitted` → `409 attempt_submitted`, `ErrNotEnough` → `409 not_enough_questions`, `ErrTestNotReady` → `409 test_not_ready {check}`, còn lại `500` + log
- [X] T016 Viết `backend/internal/service/toeic.go`: `initToeic()` tạo `toeic.NewService` với các repo từ `c.store`, adapter `toeicCards{c.vocab}`, `toeicPhotos` (bọc `ai.ImageProvider` Gemini và Codex như `lesson.Service.ImageAI`/`CodexImageAI`, để trống tới US4), `StudyDays: c.studyDays`, `Timezones: c.settings`; gọi `initToeic()` trong `backend/internal/service/service.go` **sau** `initVocab`, `initProgress`, `initSettings`; đăng ký route trong `backend/internal/service/routes.go`

### Frontend

- [X] T017 Tạo `frontend/src/app/core/services/toeic-api.service.ts` (`providedIn: 'root'`): một hàm cho mỗi route ở `contracts/api.md` (người học và admin); hàng đợi lưu tiến độ chưa gửi trong localStorage (khoá `toeic-pending:<attemptId>`, gộp `answers`, giữ `remainingMs` nhỏ nhất) với `flush()` gọi khi có mạng lại, đọc/ghi bọc `try/catch`
- [X] T018 [P] Test `frontend/src/app/core/services/toeic-queue.logic.spec.ts` cho phần gộp hàng đợi (tách thành `toeic-queue.logic.ts` thuần): gộp đáp án, −1 xoá chọn, `remainingMs` lấy nhỏ hơn, `phase` không lùi
- [X] T019 Tạo `frontend/src/app/features/toeic/toeic.routes.ts` theo bảng route ở `contracts/api.md` (lazy, component trống tạm), thêm `toeic` vào `frontend/src/app/app.routes.ts` trong khu người học; trong `frontend/src/app/layouts/learner-layout/learner-layout.ts` thêm `/toeic` vào `isPracticePath()` và `/toeic/attempts/:id` vào màn tập trung (ẩn thanh tab); cập nhật test tab trong `frontend/src/app/app.spec.ts`
- [X] T020 [P] Thêm ô **TOEIC** (icon phù hợp có sẵn trong `shared/components/icon`, ghi chú "Nghe–Đọc" hoặc "Còn N ngày" khi có ngày thi) vào `items` của `frontend/src/app/layouts/learner-layout/practice-menu/practice-menu.ts`, link `/toeic`
- [X] T021 [P] Thêm mục **TOEIC** vào `links` của `frontend/src/app/layouts/admin-layout/admin-layout.ts` (`/admin/toeic`) và route admin trong `frontend/src/app/features/admin/admin.routes.ts`

**Checkpoint**: backend biên dịch, route trả 404/501 tạm; lối vào hiện trên giao diện.

---

## Phase 3: User Story 1 - Nhập nội dung và ghép đề (Priority: P1) 🎯 MVP

**Goal**: admin nhập JSON (chạy thử, nháp, đăng/ẩn), xem trước, ghép đề có kiểm tra đủ câu và đủ ảnh.

**Independent Test**: quickstart bước 1–2; US1 kịch bản 1–6.

### Tests

- [X] T022 [P] [US1] Test `backend/internal/toeic/validate_test.go`: một cụm hợp lệ cho mỗi Part 1–7 qua; mỗi luật trong data-model.md "Kiểm tra khi nhập" có một ca sai với đúng khoá trường (`groups[i].questions[j].answer`, `…options`, `…tags`, `groups[i].script`, `groups[i].docs[0].body` thiếu `[[3]]`, `evidence.text` không có trong khối, tag `graphic` thiếu `graphic`, `key` sai mẫu, trùng `key` trong file)
- [X] T023 [P] [US1] Test `backend/internal/toeic/importer_test.go`: chạy thử không lưu gì; file có lỗi → nhập thật `ValidationError` và repo không đổi; cụm trùng không `replace` → `duplicates`; `replace` → giữ `status`, tăng `version`; thay thế cụm trong đề đã đăng mà đổi số câu → lỗi; `listeningMinutes` có khi file có Part 1–4
- [X] T024 [P] [US1] Test `backend/internal/toeic/tests_test.go` cho `Check`: đề full thiếu 4 câu Part 3 → `parts` ghi `have/need`, `ready=false`; đủ câu nhưng cụm còn nháp → `draftGroups`; thiếu ảnh Part 1 → `missingPhotos` có số câu; Mini 50/100 theo tỉ lệ R9 ±1 cụm; đăng đề chưa sẵn sàng → `ErrTestNotReady`
- [X] T025 [P] [US1] Test handler admin trong `backend/internal/toeic/handler_admin_test.go`: `POST /api/admin/toeic/import?dryRun=true` 200 + báo cáo; nhập thật có lỗi 400 `validation_failed`; thành viên gọi route admin → 403; `PATCH /groups/{key}` đổi status; `PATCH /tests/{key}` chưa sẵn sàng → 409 `test_not_ready`

### Implementation

- [X] T026 [US1] Viết `backend/internal/toeic/validate.go`: `Normalize(g *Group)` (trim, chữ thường `key`), `Validate(gs []Group) error` theo mọi luật ở data-model.md, dùng `TagAllowed`; `estimateListening(gs []Group) (minutes float64)` theo research R3 (150 từ/phút + các khoảng nghỉ cố định)
- [X] T027 [US1] Viết `backend/internal/toeic/importer.go`: `func (s *Service) Import(ctx, gs []Group, replace, dryRun bool) (ImportReport, error)` — validate, tra `ByKeys`, phân loại `created/replaced/duplicates`, kiểm tra cụm trong đề đã đăng (giữ số câu), báo cáo từng cụm; nhập thật thì `Upsert` (cụm mới `draft`); trường `Rescored` để 0 tới khi có T086
- [X] T028 [US1] Viết `backend/internal/toeic/tests.go`: `SaveTest(ctx, t Test)` (chỉ khi nháp; kiểm tra mọi `groupKey` tồn tại, thứ tự theo Part), `Check(ctx, key) (TestCheck, error)`, `SetTestStatus(ctx, key, status)` (đăng chỉ khi `Ready`), `ListTests`, `GetTest`; admin: `ListGroups(filter)`, `GetGroup(key)`, `SetGroupStatus(key, status)`
- [X] T029 [US1] Thêm route admin vào `backend/internal/toeic/handler.go`: `POST /api/admin/toeic/import` (`httpx.DecodeJSONDetailed`, giới hạn 2 MB, `dryRun` từ query), `GET /api/admin/toeic/groups`, `GET|PATCH /api/admin/toeic/groups/{key}`, `GET|POST /api/admin/toeic/tests`, `GET|PUT|PATCH /api/admin/toeic/tests/{key}` theo `contracts/api.md`
- [X] T030 [P] [US1] Tạo `frontend/src/app/features/toeic/shared/question-view.ts` (câu hỏi + phương án A–D/A–C, chế độ chỉ đọc có đánh dấu đáp án và ✓/✗), `doc-view.ts` (văn bản Part 6/7 theo `kind`: email/memo/letter có khối Meta, web có thanh URL, invoice/form/schedule có bảng, chat có bong bóng `who/time`, `**đậm**`, chỗ trống `[[n]]` hiện số câu; nhận danh sách đoạn cần tô) — dùng chung cho admin xem trước và phòng thi
- [X] T031 [P] [US1] Test `frontend/src/app/features/toeic/shared/doc-text.logic.spec.ts` cho hàm thuần tách `Block.text` thành đoạn (thường, đậm, chỗ trống, đoạn tô theo `evidence`) trong `doc-text.logic.ts`
- [X] T032 [US1] Tạo trang admin `frontend/src/app/features/admin/toeic/toeic-admin.ts|html|css`: lọc theo Part và trạng thái, bảng cụm (key, Part, câu, mức, nguồn, trạng thái, ảnh, thuộc đề), nút Xem trước (dialog dùng `question-view` + `doc-view` ở chế độ có đáp án), Đăng/Ẩn; dialog **Nhập JSON** dùng `lu-json-source` + `parseJson`/`errorBody`/`fieldLines` của `features/admin/json-import/json-import.ts`: Chạy thử → bảng báo cáo (ok/trùng/thay thế/lỗi theo trường, phút Nghe ước tính), ô "Thay thế cụm trùng", Nhập chỉ bật khi lần chạy thử cuối không lỗi và chữ chưa đổi (như `grammar-list/import-dialog`)
- [X] T033 [US1] Tạo trang đề `frontend/src/app/features/admin/toeic/test-page/test-page.ts|html|css` (`/admin/toeic/tests/:key`) và nút **Tạo đề** (dán JSON file đề): hiện `check` (số câu đã có/cần mỗi Part, cụm nháp, ảnh thiếu theo số câu, biểu đồ, văn bản, phút Nghe), danh sách cụm theo Part, nút **Đăng đề** khoá khi chưa `ready` (ghi lý do), Ẩn đề; khu ảnh Part 1 để trống tới US4
- [X] T034 [US1] Soạn lô nội dung đầu tiên (Claude): `content/toeic/part5/p5-set-01.json` (30 câu), `content/toeic/part6/p6-set-01.json` (4 văn bản), `content/toeic/part7/p7-set-01.json` (~15 cụm, có đoạn đơn/đôi/ba, nhiều loại văn bản) theo `contracts/content-format.md`; chạy thử qua API (dryRun) đến khi sạch lỗi
- [X] T035 [US1] Kiểm tra thủ công quickstart bước 1–2 và kịch bản US1 #1–6 ở 360 px và màn rộng, sáng và tối

**Checkpoint**: admin nhập, đăng nội dung và ghép được đề (phần Đọc).

---

## Phase 4: User Story 2 - Luyện tập phần Đọc trong phòng thi (Priority: P1)

**Goal**: thư viện đề → chi tiết đề → phòng thi (Part 5–7) → nộp → kết quả → đáp án chi tiết.

**Independent Test**: quickstart bước 3–7; US2 kịch bản 1–8.

### Tests

- [X] T036 [P] [US2] Test `backend/internal/toeic/paper_test.go`: `PaperView` của một đề không có `answer`/`evidence`/`explainVi`/`translateVi`/`vocab` (duyệt JSON đã mã hoá, tìm khoá); `number` đúng 1–200 cho đề full, đánh lại từ 1 cho lượt luyện tập; luyện tập Part 5+6 chỉ có cụm của hai Part đó; luyện theo Part ưu tiên cụm có ít câu đã làm
- [X] T037 [P] [US2] Test `backend/internal/toeic/grade_test.go`: đếm đúng/sai/bỏ qua và `ByQuestion`; `scaled` theo các mốc R5 (0→5, 50 Listening→245, 50 Reading→215, 100→495, bước 5, nội suy giữa mốc); Mini 100 câu quy đổi theo tỉ lệ; lượt thiếu Part → `Total=0`, `Listening/Reading=nil`
- [X] T038 [P] [US2] Test `backend/internal/toeic/attempts_test.go`: tạo lượt luyện tập (giới hạn giờ, Part đã chọn); đang có bài dở cùng đề → `ErrAttemptInProgress` kèm id; `Save` gộp đáp án, −1 xoá, `remainingMs` tăng bị bỏ qua, câu không có trong đề bị từ chối; `Save` sau khi nộp → `ErrAttemptSubmitted`; `Submit` chấm, gọi `MarkStudied` một lần, gọi lại trả kết quả cũ; `remainingMs = 0` khi mở lại → tự nộp
- [X] T039 [P] [US2] Test handler người học `backend/internal/toeic/handler_learner_test.go`: `GET /api/toeic` 200/401; `POST /api/toeic/attempts` 201, 400 (Part lạ, thiếu `testKey`), 403 khách làm đề khác, 409 bài dở; `GET /attempts/{id}` của người khác → 404; `GET /result` khi chưa nộp → 409; `GET /questions/{qid}` sau khi nộp có `answer`, `explainVi`

### Backend

- [X] T040 [US2] Viết `backend/internal/toeic/paper.go`: `buildPaper(ctx, userID string, src Source, parts []Part) ([]Group, error)` cho `test` (cụm của đề, lọc Part) và `part` (`count` câu/cụm từ cụm đã đăng, ưu tiên ít câu đã làm theo `AnsweredQuestions`, rồi ngẫu nhiên bằng `Rand`); `PaperView(groups, numbering) []PaperGroup` bỏ mọi trường đáp án, gắn `imageUrl` (`/api/toeic/images/{key}/{slot}`) khi có ảnh; đánh số câu (đề full theo thứ tự chuẩn 1–200)
- [X] T041 [US2] Viết `backend/internal/toeic/grade.go`: `grade(groups []Group, answers map[string]int) Result` (đúng/sai/bỏ qua, `ByQuestion`, theo Section), `scaled(section Section, correct, total int) int` theo R5, `fullParts(parts) bool`, ghi `Version` từng cụm
- [X] T042 [US2] Viết `backend/internal/toeic/attempts.go`: `Start(ctx, userID string, isGuest bool, req StartReq) (Attempt, []PaperGroup, error)` (gọi `checkStart`, kiểm tra bài dở, `buildPaper`, lưu `GroupKeys`, `QuestionIDs`, `TimeLimitMs`, `RemainingMs`), `Open(ctx, userID, id)` (tự nộp khi `RemainingMs == 0` và có giới hạn giờ), `SaveProgress(ctx, userID, id, p Progress)` (luật ở `contracts/api.md`), `Submit(ctx, userID, id, last Progress) (Result, error)` (chấm, `Counted` = lần nộp đầu có đủ 7 Part của đề/đề ngẫu nhiên, gọi `StudyDays.MarkStudied`; móc cập nhật sổ câu sai để trống tới US3)
- [X] T043 [US2] Viết `backend/internal/toeic/stats.go` phần thư viện: `Library(ctx, userID, isGuest) (LibraryView, error)` (đề đã đăng theo `Order`, trạng thái từ bài dở/bài nộp gần nhất, `locked`, `latest` chỉ từ bài `Counted`, `recent` 5 lượt), `TestDetail(ctx, userID, isGuest, key)` (Part, số câu, tag có trong đề, các lần làm có cờ `retake`)
- [X] T044 [US2] Viết `ResultView(ctx, userID, id)` và `QuestionDetail(ctx, userID, id, qid)` trong `backend/internal/toeic/review.go`: bảng phân loại theo Part → tag (đúng/sai/bỏ qua, danh sách số câu), danh sách đáp án (chữ A–D), chi tiết câu (cụm đầy đủ có `translateVi`, `evidence`, `explainVi`, `vocab`, `tags` kèm `labelVi` và `grammarPointId` đầu tiên); chỉ khi bài đã nộp
- [X] T045 [US2] Thêm route người học vào `backend/internal/toeic/handler.go`: `GET /api/toeic`, `GET /api/toeic/tests/{key}`, `POST /api/toeic/attempts`, `GET|PUT /api/toeic/attempts/{id}`, `POST /api/toeic/attempts/{id}/submit`, `GET /api/toeic/attempts/{id}/result`, `GET /api/toeic/attempts/{id}/questions/{qid}`, `GET /api/toeic/images/{groupKey}/{slot}` (ảnh cụm đã đăng; admin cả nháp; `ETag`), và `POST /api/toeic/vocab` `{word, ipa, meaningVi, sentence}` → `Cards.SaveWord` (thêm vào sổ từ, FR-028)

### Frontend

- [X] T046 [P] [US2] Tạo `frontend/src/app/features/toeic/exam-room/exam-state.logic.ts` (thuần): trạng thái đề (câu theo thứ tự, Part, đáp án, đánh dấu, vị trí), `select`, `toggleFlag`, `goto`, `next/prev`, `unanswered()`, trạng thái ô bảng câu (`answered|flagged|empty|current`), nhóm theo Part, dựng `SaveProgress` khác biệt so với lần gửi trước; test `exam-state.logic.spec.ts`
- [X] T047 [P] [US2] Tạo `frontend/src/app/features/toeic/exam-room/autosave.ts`: lưu khi chọn đáp án (gộp 1 giây), mỗi 5 giây khi đồng hồ chạy, khi `visibilitychange` ẩn (`fetch` `keepalive`); lỗi mạng → hàng đợi T017; đồng hồ dùng `startClock/pauseClock/resumeClock/remaining` của `frontend/src/app/features/games/shared/game-clock.logic.ts`
- [X] T048 [US2] Tạo thư viện đề `frontend/src/app/features/toeic/library/library.ts|html|css` (`/toeic`): hàng tab (Đề thi, Luyện theo Part, Luyện theo dạng câu, Kết quả của tôi, Sổ câu sai · N), lưới thẻ đề (tên, ⏱ phút · phần · câu, trạng thái chưa làm / đang làm câu N → "Làm tiếp" / đã làm + điểm hoặc số đúng, khoá "chỉ dành cho thành viên" cho khách), cột bên (mục tiêu, điểm ước tính gần nhất, gần đây); xếp dọc dưới 1024 px; trạng thái rỗng "Chưa có đề"
- [X] T049 [US2] Tạo chi tiết đề `frontend/src/app/features/toeic/test-detail/test-detail.ts|html|css` (`/toeic/tests/:key`): tab **Luyện tập** (tick Part có câu, mỗi Part hiện số câu + `#tag`, chọn giới hạn giờ: gợi ý ≈ 0,6 phút/câu Đọc, vài mức khác, "Không giới hạn"; nút Luyện tập → `POST /attempts` → phòng thi; 409 bài dở → hỏi "Làm tiếp bài dở?"), tab **Làm full test** ("Sắp có" tới US5), tab **Kết quả** (các lần làm, nhãn "làm lại")
- [X] T050 [US2] Tạo phòng thi `frontend/src/app/features/toeic/exam-room/exam-room.ts|html|css` (`/toeic/attempts/:id`): thanh trên (tên đề, chế độ, Thoát — lưu rồi về chi tiết đề), hàng tab Part, ≥ 1024 px chia cột (văn bản `doc-view` | câu hỏi `question-view` | cột bên đồng hồ + NỘP BÀI + bảng câu), < 1024 px xếp dọc với thanh dưới (☰ Bảng câu, Nộp bài); Part 5 nhiều câu một trang; đánh dấu ⚑; hết giờ tự nộp; nộp khi còn câu chưa làm → hộp thoại xác nhận; nộp lỗi mạng → "Chưa gửi được, thử lại", không mất bài
- [X] T051 [P] [US2] Tạo `frontend/src/app/features/toeic/exam-room/question-grid.ts` (bảng câu theo Part, 4 trạng thái có chữ/biểu tượng ngoài màu, chạm để nhảy) và `answer-sheet.ts` (phiếu trả lời ô tròn A–D/A–C cho điện thoại, mở từ ☰)
- [X] T052 [P] [US2] Tô vàng trong `doc-view`: bôi đen chữ → nút "Tô vàng" (và bỏ tô), lưu theo bài trong localStorage `toeic-highlights:<attemptId>` (bọc `try/catch`), dựng lại khi mở bài (FR-009); logic thuần khoảng tô trong `frontend/src/app/features/toeic/shared/highlight.logic.ts` + spec
- [X] T053 [US2] Tạo trang kết quả `frontend/src/app/features/toeic/result/result.ts|html|css` (`/toeic/attempts/:id/result`): hàng thẻ (kết quả x/y, độ chính xác, thời gian; ✓ đúng, ✗ sai, – bỏ qua; Listening, Reading, Tổng · **ước tính** khi có, so với mục tiêu), bảng **Phân loại câu hỏi** theo tab Part (tag, đúng, sai, bỏ qua, độ chính xác, ô số câu tô màu có ✓/✗ → link đáp án chi tiết), cột **Đáp án** (số câu, đáp án đúng : bạn chọn, [chi tiết]); nút "Làm lại các câu sai" để ẩn tới US3; `score.logic.ts` (định dạng điểm, thời gian, độ chính xác) + spec
- [X] T054 [US2] Tạo đáp án chi tiết `frontend/src/app/features/toeic/answer-detail/answer-detail.ts|html|css` (`/toeic/attempts/:id/q/:qid`): nhãn Part, `#tag` (link sang `/grammar/:pointId` khi có), "Bạn chọn X · Đáp án Y"; văn bản `doc-view` tô `evidence`, nút hiện bản dịch; khối Giải thích; khối Từ vựng với "+ Sổ từ" (`POST /api/toeic/vocab`, báo "Đã thêm"); nút câu trước/sau
- [X] T055 [US2] Kiểm tra thủ công quickstart bước 3–7 và kịch bản US2 #1–8 (360 px và ≥ 1024 px, sáng và tối); kiểm tra trong DevTools không có `answer`/`explainVi` trước khi nộp

**Checkpoint**: luyện tập phần Đọc hoàn chỉnh, có kết quả và đáp án chi tiết.

---

## Phase 5: User Story 3 - Sổ câu sai (Priority: P2)

**Goal**: câu sai/bỏ qua vào sổ; làm lại chấm ngay, đúng 2 lần liên tiếp thì ra khỏi sổ; "Làm lại các câu sai" từ trang kết quả.

**Independent Test**: quickstart bước 8; US3 kịch bản 1–5.

- [X] T056 [P] [US3] Test `backend/internal/toeic/mistakes_test.go`: `applyMistake` (bài nộp sai → thêm/đặt 0; bài nộp đúng → không đổi; lượt sổ đúng → 1 rồi xoá ở 2; lượt sổ sai → 0); `Submit` thêm mục cho câu sai và bỏ qua; `round` xếp sai lâu nhất trước, câu thuộc cụm trả cả cụm; `check` của khách → `ErrMembersOnly`; `retry` từ bài nộp chỉ gồm câu sai của bài đó
- [X] T057 [US3] Viết `backend/internal/toeic/mistakes.go`: `applyMistake(entry *Mistake, right bool, drill bool) (keep bool)`, `recordSubmitted(ctx, userID, groups, result)` (gọi từ `Submit` ở T042), `Mistakes(ctx, userID, part)`, `MistakeRound(ctx, userID, part, count)`, `CheckMistake(ctx, userID, qid, choice) (CheckReply, error)` (chấm ngay, trả đáp án + giải thích), `FinishMistakes(ctx, userID)` → `MarkStudied`; thêm `SourceKind=retry` vào `buildPaper` (câu sai của `FromAttempt`, cụm đầy đủ, chỉ chấm câu sai: các câu khác của cụm không tính vào kết quả)
- [X] T058 [US3] Thêm route vào `backend/internal/toeic/handler.go` (thành viên): `GET /api/toeic/mistakes`, `GET /api/toeic/mistakes/round`, `POST /api/toeic/mistakes/{qid}/check`, `POST /api/toeic/mistakes/finish`; test trong `backend/internal/toeic/handler_learner_test.go` (200, 400 `choice` ngoài khoảng, 403 khách)
- [X] T059 [US3] Tạo `frontend/src/app/features/toeic/mistakes/mistakes.ts|html|css` (`/toeic/mistakes`): lọc chip Part (kèm số), danh sách câu (trích câu hỏi, Part, `#tag`, chấm "Đúng liền ●○"), nút "Làm lại 10 câu"; màn làm lại: cụm đầy đủ (`doc-view`/`question-view`), chọn → chấm ngay (✓/✗, đáp án, giải thích, evidence tô), "Ra khỏi sổ" khi đủ 2; xong lượt gọi `finish`; khách thấy khoá
- [X] T060 [US3] Hiện nút **Làm lại các câu sai (N)** ở `frontend/src/app/features/toeic/result/result.ts` → `POST /attempts {source: {kind: "retry", fromAttempt}}` → phòng thi
- [X] T061 [US3] Kiểm tra thủ công quickstart bước 8 và US3 #1–5; mở sổ ở trình duyệt khác thấy cùng nội dung

**Checkpoint (hết đợt 1)**: US1 + US2 + US3 phát hành được; tab Full test hiện "Sắp có".

---

## Phase 6: User Story 4 - Phần Nghe (Priority: P2)

**Goal**: Part 1–4 trong phòng thi: Directions, kịch bản đọc nhiều giọng, ảnh Part 1, biểu đồ, transcript sau khi nộp.

**Independent Test**: quickstart bước 9; US4 kịch bản 1–7.

### Tests

- [X] T062 [P] [US4] Test `frontend/src/app/features/toeic/listening/listening-script.logic.spec.ts`: Part 1 có câu dẫn "Number n…", (A)–(D), nghỉ 5 giây; Part 2 câu hỏi + (A)–(C); Part 3 câu dẫn "Questions 41 through 43…", các lượt với `accent` đúng người nói, đọc từng câu hỏi + nghỉ 8 giây; Directions đứng đầu mỗi Part (bỏ được ở luyện tập); `segmentOfGroup`, `groupStart(segment)`; ước tính thời lượng khớp R3
- [X] T063 [P] [US4] Test `frontend/src/app/core/services/speech-voice.logic.spec.ts` cho hàm thuần chọn giọng `pickVoice(voices, lang, index)` (tách khỏi `speech.service.ts`): khớp `en-GB` khi có, không có thì giọng tiếng Anh xếp hạng cao, chỉ số khác nhau → giọng khác nhau khi có ≥ 2 giọng; `voiceCount`
- [X] T064 [P] [US4] Test `backend/internal/toeic/images_test.go`: vẽ thiếu `engine` → `ValidationError{engine}`; engine lạ → lỗi; prompt nhận `Scene` + câu mô tả đúng, không có 3 câu sai; Codex không có → lỗi map đúng; tải lên ảnh > 5 MB hoặc sai định dạng → lỗi; ảnh được thu nhỏ ≤ 800 px; hàng loạt chỉ vẽ ảnh còn thiếu, `409 batch_running` khi đang chạy

### Backend

- [X] T065 [US4] Thêm `PhotoBrief{Scene, Statement string}` và trường `Photo *PhotoBrief` vào `ai.ImageRequest` trong `backend/internal/ai/image.go`; prompt tiếng Anh ảnh chụp thật (research R7) trong `backend/internal/ai/llm/image.go` (op ghi sử dụng `toeic_photo`) và `backend/internal/ai/codexcli/codexcli.go`; test prompt trong file test hiện có của `ai/llm`
- [X] T066 [US4] Viết `backend/internal/toeic/images.go`: `DrawPhoto(ctx, groupKey, slot, engine string)` (bắt buộc engine, chỉ Part 1 `photo` hoặc `doc-n` có trong cụm, lưu `imageutil.Shrink(data, 800)` với `Origin = engine`), `UploadImage(ctx, groupKey, slot, data)`, `DeleteImage`, `Image(ctx, groupKey, slot, isAdmin)`; hàng loạt `StartPhotoBatch(ctx, testKey, engine)` / `BatchStatus` / `StopBatch` chạy ngầm trong process theo mẫu `backend/internal/wordbank/batch.go` (1 lượt mỗi lúc); nối `toeicPhotos` thật trong `backend/internal/service/toeic.go`
- [X] T067 [US4] Thêm route admin ảnh vào `backend/internal/toeic/handler.go`: `POST /api/admin/toeic/images/{groupKey}/{slot}/generate`, `POST|DELETE /api/admin/toeic/images/{groupKey}/{slot}` (multipart ≤ 5 MB), `POST|GET /api/admin/toeic/tests/{key}/images/batch`, `POST …/batch/stop`; lỗi vẽ map như `drawError` của `backend/internal/lesson/handler.go`
- [X] T068 [US4] Mở rộng `Check` trong `backend/internal/toeic/tests.go` dùng `ImageRepository.Has` cho ảnh Part 1 thật (thay chỗ để trống ở T028) và đếm biểu đồ; `PaperView` gắn `imageUrl` cho Part 1 và `doc-n`

### Frontend

- [X] T069 [US4] Mở rộng `frontend/src/app/core/services/speech.service.ts`: tham số `opts?: { lang?: 'en-US'|'en-GB'|'en-AU'|'en-CA'; natural?: boolean }` cho `speak` (dùng `pickVoice` ở T063; `natural: false` bỏ `NaturalVoiceService`), hàm `voiceCount()`; hành vi cũ không đổi khi không truyền `opts`
- [X] T070 [US4] Tạo `frontend/src/app/features/toeic/listening/listening-script.logic.ts` (dựng đoạn đọc từ `PaperGroup[]`, Directions tiếng Anh do Luna tự viết cho từng Part + tóm tắt tiếng Việt, câu dẫn, khoảng nghỉ, ước tính thời lượng) và `listening-player.ts` (chạy đoạn qua `SpeechService` với `accent → lang` và chỉ số giọng theo người nói, tự sang đoạn sau sau `pauseAfterMs`; chế độ luyện tập: tạm dừng = dừng và đọc lại từ đầu đoạn, nghe lại cụm, tốc độ 0.75/1/1.25, nhảy tới câu → đầu cụm; phát ra `segment`, `group`, `speaking`)
- [X] T071 [US4] Tạo `frontend/src/app/features/toeic/exam-room/directions.ts`: màn Directions mỗi Part (chữ tiếng Anh + dòng tóm tắt tiếng Việt, thanh âm thanh, nút "Bỏ qua hướng dẫn" khi luyện tập); nút bấm mở khoá âm thanh khi trình duyệt chặn tự phát
- [X] T072 [P] [US4] Tạo `frontend/src/app/features/toeic/shared/graphic-view.ts`: `table`, `schedule`, `price-list` (bảng có tiêu đề cột), `floor-plan` (lưới CSS theo `row/col/span`), `bar-chart` (cột CSS có nhãn và giá trị bằng chữ); đọc rõ ở 360 px, sáng và tối, có văn bản thay thế cho trình đọc màn hình
- [X] T073 [US4] Đưa Part 1–4 vào `frontend/src/app/features/toeic/exam-room/exam-room.ts|html`: thanh âm thanh luôn ở trên cùng (cụm đang phát, tiến độ, thời lượng); Part 1 ảnh lớn (chạm để phóng to trên điện thoại) + nút A–D, không hiện chữ câu mô tả; Part 2 "Mark your answer" + A–C; Part 3/4 câu hỏi + phương án bằng chữ + `graphic-view`; con trỏ bảng câu đi theo cụm đang phát; lưu `listeningSegment` khi đổi cụm
- [X] T074 [US4] Đáp án chi tiết phần Nghe trong `frontend/src/app/features/toeic/answer-detail/answer-detail.ts|html`: thanh nghe lại cả cụm (`listening-player` chế độ luyện tập), transcript dạng bong bóng ghi người nói + giọng, tô `evidence` và ghi số câu, nút bản dịch từng lượt; Part 1/2 hiện chữ các câu (A)–(D)/(A)–(C)
- [X] T075 [US4] Khu ảnh Part 1 trong `frontend/src/app/features/admin/toeic/test-page/test-page.ts|html`: lưới ảnh theo câu (có ảnh / "Chưa có"), Vẽ lại từng ảnh, Tự tải ảnh lên, **Vẽ N ảnh còn thiếu…** (hàng loạt, tiến độ, Dừng) — mọi lần vẽ mở `lu-engine-dialog` với `requireChoice` (không nhớ lựa chọn); xem trước cụm Part 1 trong `toeic-admin` hiện ảnh
- [X] T076 [P] [US4] Thêm op `toeic_photo` ("Ảnh Part 1 TOEIC") vào `OP_LABELS` và một dòng bảng chi phí (`FeatureCost`, `paid: true`) trong `frontend/src/app/features/admin/ai-usage/ai-usage-catalog.ts`
- [X] T077 [US4] Soạn nội dung Part 1–4 (Claude): `content/toeic/part1/p1-set-01.json` (≥ 12 câu, `scene` mô tả ảnh chụp), `part2/p2-set-01.json` (≥ 50 câu), `part3/p3-set-01.json` (≥ 26 cụm, có cụm 3 người và ≥ 6 cụm `graphic`), `part4/p4-set-01.json` (≥ 20 cụm, có `graphic`); chạy thử đến khi sạch lỗi; vẽ ảnh Part 1 khi người dùng chọn engine
- [ ] T078 [US4] Kiểm tra thủ công quickstart bước 9, bước 17 và US4 #1–7 (điện thoại thật có nhiều giọng tiếng Anh, 360 px tối)

**Checkpoint**: luyện tập đủ 7 Part.

---

## Phase 7: User Story 5 - Full test, Mini Test và điểm ước tính (Priority: P3)

**Goal**: làm full test/Mini Test như thi thật, đề ngẫu nhiên, điểm thang 990, làm lại đề; nhập lại để sửa và chấm lại (FR-040a).

**Independent Test**: quickstart bước 5, 10, 11; US5 kịch bản 1–8; US1 kịch bản 7.

### Tests

- [ ] T079 [P] [US5] Bổ sung `backend/internal/toeic/attempts_test.go`: full test bắt đầu ở `phase=listening`, `Save` không cho `phase` lùi, không nhận đáp án câu Part 5–7 khi đang ở phần Nghe, không nhận đáp án câu của cụm Nghe sau cụm đang phát (`listeningSegment`); đồng hồ 75 phút chỉ bắt đầu khi sang `reading`; lần nộp đầu đủ 7 Part → `Counted=true`, lần làm lại → `Counted=false` và thẻ đề hiện lần mới (`Library`)
- [ ] T080 [P] [US5] Test `backend/internal/toeic/random_test.go`: đề ngẫu nhiên 50/100 câu đủ 7 Part theo tỉ lệ R9 (±1 cụm), ưu tiên cụm chưa làm, đề được lưu trong bài (mở lại giống hệt); kho thiếu → `ErrNotEnough`; khách → `ErrMembersOnly`
- [X] T081 [P] [US1] Test `backend/internal/toeic/rescore_test.go` (FR-040a, US1 #7): thay thế cụm đổi đáp án → bài đã nộp có cụm đó được chấm lại (`Result`, điểm), bài không có cụm đó giữ nguyên; câu trước sai nay đúng → xoá mục sổ; câu trước đúng nay sai → thêm mục `Streak=0`; bài dở không bị chấm; `ImportReport.Rescored` đúng số bài

### Implementation

- [ ] T082 [US5] Mở rộng `backend/internal/toeic/attempts.go`: `Mode=test` cho đề cố định (đủ Part, không chọn giờ: Mini 50 = 38 phút, Mini 100 = 60 phút, full: phần Nghe theo kịch bản + 75 phút phần Đọc), luật `phase`/`listeningSegment` theo `contracts/api.md`, mở lại phần Nghe trả `listeningSegment` của đầu cụm
- [ ] T083 [US5] Thêm `SourceKind=random` vào `backend/internal/toeic/paper.go` theo R9 (thành viên) và route `POST /api/toeic/attempts {source: {kind: "random", size}}` (đã có), thẻ **Đề ngẫu nhiên** trong `frontend/src/app/features/toeic/library/library.html` (chọn 50/100)
- [ ] T084 [US5] Tab **Làm full test** trong `frontend/src/app/features/toeic/test-detail/test-detail.ts|html`: cảnh báo điều kiện thi thật, thời lượng từng phần, ô kiểm tra (tai nghe, nơi yên tĩnh), "▶ Nghe thử giọng đọc" (đọc một câu bằng 2 giọng, báo khi `voiceCount() < 2`), nút **Bắt đầu thi**
- [ ] T085 [US5] Chế độ thi trong `frontend/src/app/features/toeic/exam-room/` và `listening-player.ts`: phần Nghe phát liền (không nút tạm dừng/tua/tốc độ, Directions không bỏ được), khoá tab Part 5–7 khi đang Nghe và khoá Part 1–4 khi sang Đọc, bảng câu không nhảy trong phần Nghe, chỉ sửa đáp án cụm đang phát; hết phần Nghe → màn chuyển "Bắt đầu phần Đọc" → đồng hồ 75 phút; mở lại giữa phần Nghe → phát từ đầu cụm
- [X] T086 [US1] Viết `backend/internal/toeic/rescore.go`: sau `Import` với `replace` (gọi từ `importer.go`), lấy `AttemptRepository.WithGroup`, chấm lại từng bài bằng `grade`, `UpdateResult`; cập nhật sổ câu sai theo research R8 (câu trả lời ở bài nộp gần nhất có câu đó); điền `ImportReport.Rescored`; báo cáo chạy thử ghi "sẽ chấm lại N bài"
- [ ] T087 [US5] Trang kết quả: hiện điểm thang 990 + "ước tính" + so với mục tiêu cho lượt đủ 7 Part, nhãn "làm lại" khi `retake` trong `frontend/src/app/features/toeic/result/result.html`; thẻ đề trong thư viện hiện kết quả lần gần nhất
- [ ] T088 [US5] Soạn đề (Claude): đủ câu cho `content/toeic/tests/luna-test-01.json`, `luna-test-02.json` (200 câu mỗi đề, phần Nghe 40–50 phút ước tính), `mini-01.json` (Mini 100), `mini-02.json` (Mini 50) — bổ sung các file `content/toeic/part*/` còn thiếu; chạy thử, nhập, ghép đề, vẽ ảnh, đăng
- [ ] T089 [US5] Kiểm tra thủ công quickstart bước 5, 10, 11 và US5 #1–8 (làm trọn một full test ít nhất một lần, đo thời lượng phần Nghe)

**Checkpoint**: thi thử hoàn chỉnh.

---

## Phase 8: User Story 6 - Luyện theo dạng câu, thống kê và mục tiêu (Priority: P4)

**Goal**: mục tiêu + ngày thi, luyện theo dạng câu yếu nhất, thống kê có biểu đồ điểm.

**Independent Test**: quickstart bước 12; US6 kịch bản 1–5.

- [ ] T090 [P] [US6] Test `backend/internal/toeic/stats_test.go`: độ chính xác theo tag từ mọi bài nộp (xếp tăng dần, tag chưa làm ở cuối, `available` đếm câu đã đăng); điểm cho biểu đồ chỉ từ bài `Counted`; mục tiêu: `target` ngoài 450/600/750/900 hoặc ngày sai → `ValidationError`, `daysLeft` theo múi giờ người học; luyện theo tag chỉ có câu mang tag đó, ưu tiên chưa làm
- [ ] T091 [US6] Viết phần còn lại của `backend/internal/toeic/stats.go`: `Tags(ctx, userID)`, `Stats(ctx, userID)` (điểm `Counted`, độ chính xác theo Part và tag), `SetGoal`/`DeleteGoal`/`goal` (dùng `Timezones` cho `daysLeft`); `SourceKind=tag` trong `paper.go` (10 câu hoặc 3 cụm có tag)
- [ ] T092 [US6] Thêm route vào `backend/internal/toeic/handler.go`: `GET /api/toeic/tags`, `GET /api/toeic/stats` (thành viên), `PUT|DELETE /api/toeic/goal`; test trong `backend/internal/toeic/handler_learner_test.go` (200, 400, 403 khách cho tags/stats)
- [ ] T093 [P] [US6] Tạo `frontend/src/app/features/toeic/goal/goal.ts|html|css` (`/toeic/goal`): 4 mốc 450/600/750/900 (mô tả ngắn), ngày thi (không bắt buộc, có nút xoá), Lưu; cột bên thư viện và ô TOEIC trong Luyện hiện mục tiêu + số ngày còn lại
- [ ] T094 [P] [US6] Tạo `frontend/src/app/features/toeic/practice/practice.ts|html|css`: `/toeic/practice` (chọn Part + số câu/cụm → lượt `part`; khách chỉ Part 1, 2, 5) và `/toeic/tags` (danh sách tag: chip Part, nhãn, độ chính xác có màu + chữ, nút "10 câu"/"N cụm", link "Học điểm ngữ pháp" sang `/grammar/:pointId` khi có)
- [ ] T095 [US6] Tạo `frontend/src/app/features/toeic/results/results.ts|html|css` (`/toeic/results`): biểu đồ đường điểm ước tính (SVG tự vẽ, không thư viện, đường mục tiêu nét đứt, nhãn điểm, bảng dữ liệu ẩn cho trình đọc màn hình), độ chính xác theo Part và tag (thanh + %), danh sách mọi lần làm (link trang kết quả)
- [ ] T096 [US6] Kiểm tra thủ công quickstart bước 12 và US6 #1–5

---

## Phase 9: Polish & Cross-Cutting Concerns

- [ ] T097 [P] Rà khả năng tiếp cận phòng thi và trang kết quả: nút có nhãn đọc được, ✓/✗ và chữ ngoài màu, `aria-live="polite"` báo đã lưu/đã nộp và thời gian còn lại ở các mốc (10, 5, 1 phút; không đọc mỗi giây), bảng câu và phiếu trả lời dùng được bằng bàn phím, tương phản WCAG AA ở sáng và tối
- [ ] T098 [P] Kiểm tra quyền khách toàn bộ (quickstart bước 13): mọi route thành viên trả 403, giao diện khoá đúng chỗ, khách mở đề khoá bằng đường dẫn trực tiếp không thấy nội dung
- [ ] T099 [P] Kiểm tra mất mạng và chuỗi ngày học (quickstart bước 14–15), xuất dữ liệu và xoá tài khoản (bước 16)
- [ ] T100 Chạy `go test ./...`, `go vet ./...` trong `backend/`, `npm test` và `npm run lint` trong `frontend/`; sửa hết lỗi
- [ ] T101 Đối chiếu tiêu chí nghiệm thu trong `docs/spec-inputs/f27-luyen-thi-toeic.md` và SC-001…SC-010 của spec với kết quả kiểm tra; ghi các mục chưa đạt vào cuối file này

---

## Dependencies & Execution Order

- **Setup (T001–T004)** → **Foundational (T005–T021)** → các user story.
- **US1 (P1)** chỉ cần Foundational. **US2 (P1)** cần nội dung đã đăng (US1, ít nhất T026–T029 + T034).
- **US3 (P2)** cần US2 (`Submit`, trang kết quả).
- **US4 (P2)** cần US2 (phòng thi); độc lập với US3.
- **US5 (P3)** cần US2 + US4 (đủ 7 Part, trình phát Nghe). T081/T086 (chấm lại, thuộc US1 kịch bản 7) cần US2 + US3.
- **US6 (P4)** cần US2 (bài đã nộp); làm sau cùng.
- Trong mỗi story: test → backend → frontend logic → component → nội dung → kiểm tra thủ công.

## Parallel Opportunities

- Phase 1: T002 ∥ T003 ∥ T004 (sau T001).
- Phase 2: T007 ∥ T009 ∥ T012/T013 ∥ T017/T018 ∥ T020 ∥ T021; T011 và T012 khác thư mục.
- US1: T022 ∥ T023 ∥ T024 ∥ T025; T030/T031 (frontend) ∥ T026–T029 (backend); T034 (nội dung) chạy song song với code.
- US2: T036 ∥ T037 ∥ T038 ∥ T039; T046 ∥ T047 ∥ T051 ∥ T052.
- US4: T062 ∥ T063 ∥ T064; T072 ∥ T076; T077 (nội dung) song song với code.
- US6: T093 ∥ T094.
- Nội dung (T034, T077, T088) do Claude soạn có thể làm song song với mọi phase code.

### Ví dụ chạy song song (US2)

```text
T036 backend/internal/toeic/paper_test.go
T037 backend/internal/toeic/grade_test.go
T046 frontend/src/app/features/toeic/exam-room/exam-state.logic.ts (+ spec)
T051 frontend/src/app/features/toeic/exam-room/question-grid.ts, answer-sheet.ts
```

## Implementation Strategy

1. **MVP (đợt 1)**: Phase 1 + 2 + US1 + US2 + US3 → kiểm tra quickstart 1–8 → dừng lại cho người dùng thử phần Đọc.
2. **Đợt 2**: US4 (phần Nghe, ảnh) → quickstart 9, 17.
3. **Đợt 3**: US5 (full test, Mini Test, ngẫu nhiên, chấm lại) + soạn đủ 2 đề + 2 Mini Test → quickstart 5, 10, 11.
4. **Đợt 4**: US6 → quickstart 12.
5. Phase 9 sau cùng; mỗi đợt chạy test + lint trước khi báo xong.

## Ghi chú khi làm đợt 1 (2026-11-19)

- **Đã xong (code + test tự động)**: Phase 1–2, US1, US2, US3; thêm sớm T068 (Check dùng ảnh thật, `imageUrl`), T081/T086
  (nhập thay thế thì chấm lại bài cũ và cập nhật sổ câu sai) để nhập lại nội dung không làm sai kết quả.
- **Quyết định trong lúc làm**:
  - Đề được **đăng khi đủ Part 5–7** và mọi Part đã có đều đủ câu (đợt 1 chưa có phần Nghe); `check.complete` cho biết đủ 7
    Part. Tab "Làm full test" và Part 1–4 hiện "sắp có"; `mode: test` trả 400 tới đợt 3.
  - Số câu: đề full dùng số của đề thật (Part 5 bắt đầu ở 101 kể cả khi chưa có Part 1–4), Mini Test đếm từ 1; lượt
    luyện theo Part đếm từ 1; lượt "làm lại câu sai" giữ số câu của bài gốc.
  - Thêm `elapsedMs` vào tiến độ (thời gian làm thực, chỉ tăng) để trang kết quả có thời gian cả khi không giới hạn giờ.
  - Thêm `POST /api/admin/toeic/groups/status` (đăng nhiều cụm nháp một lần) và trang `/toeic/practice` (Luyện theo Part,
    khách chỉ Part 1, 2, 5) vì tab đã có trong thư viện.
  - Lượt làm lại câu sai chỉ hiện và chấm các câu sai (văn bản của cụm vẫn hiện đủ).
  - Mongo: nội dung cụm, trạng thái và kết quả bài làm lưu thành tài liệu con theo tên trường JSON (không thêm tag bson vào
    domain); ảnh TOEIC nằm ở DB ảnh như các ảnh khác.
- **Nội dung**: `content/toeic/part5/p5-set-01.json` (30 câu), `part6/p6-set-01.json` (4 văn bản, 16 câu),
  `part7/p7-set-01.json` (15 cụm, 54 câu: 10 đoạn đơn, 2 đôi, 3 ba), đề `tests/luna-test-01.json` (100 câu phần Đọc).
  Kiểm tra bằng `go test ./internal/toeic -run TestContentFiles` (chạy đúng `Validate` của lần nhập chạy thử, không cần
  mạng/tài khoản admin) thay cho dryRun qua API.
- **Kiểm tra thủ công (T035, T055, T061)**: chạy app local, lái Chromium bằng Playwright trên database tạm riêng
  (`MONGO_DATABASE=luna_toeic_check*`, tài khoản đầu tiên là admin): 43/44 mục đạt (mục còn lại do kịch bản; kiểm lại riêng
  thì đạt). Đã kiểm: nhập lỗi bị chặn, nhập/đăng/xem trước, tạo và đăng đề, Luyện → TOEIC 2 lần chạm, giờ gợi ý theo Part,
  đề không lộ đáp án, đánh dấu, nhảy câu, tô vàng, thoát rồi làm tiếp giữ đáp án/thời gian/tô vàng, 409 bài dở, hỏi lại khi
  nộp còn câu, kết quả và phân loại, đáp án chi tiết (tô bằng chứng, bản dịch, "+ Sổ từ"), sổ câu sai (1/2 → 0/2 → ra sổ),
  làm lại câu sai, 360 px không cuộn ngang, phiếu trả lời, chế độ tối, quyền khách (403 và khoá trên giao diện), file xuất
  có `toeicAttempts`/`toeicMistakes`/`toeicGoals`.
- **Lỗi tìm thấy khi kiểm và đã sửa**: thanh phòng thi bị thanh app che ở màn rộng; nút ⚑ đè chữ phương án trên điện thoại;
  tiêu đề trang kết quả bị ép từng chữ trên điện thoại; dòng Part ở chi tiết đề vỡ hàng; tiêu đề cột Đúng/Sai bị bẻ chữ;
  số các chỗ trống Part 6 ở đáp án chi tiết hiện (2)(3)(4); prompt chèn câu Part 6 ghi "blank (2)" thay vì số thật.
- **Nội dung đã vào DB thật** (tài khoản admin người dùng cấp): 49 cụm đã đăng, đề `luna-test-01` đã đăng. Tài khoản học
  viên người dùng cấp đã làm thử một lượt Part 6 (16 câu đang nằm trong sổ câu sai của tài khoản đó).
- **Chưa làm**: test MySQL `toeic_test.go` chỉ chạy khi có `MYSQL_TEST_DSN`; chưa thử xoá một tài khoản thật (chỉ kiểm bằng
  code và test).

## Sửa trang quản trị TOEIC (2026-11-20)

- Người dùng thấy trang đầu `/admin/toeic` giống chi tiết một đề (bảng toàn bộ cụm nằm ngay dưới danh sách đề chỉ có 1 dòng).
  Tách theo FR-042a, phác thảo `design/toeic/de-xuat-trang-quan-tri-toeic.png`:
  - `toeic-admin` (trang đầu): thẻ đề + "Tạo đề" + số liệu kho cụm câu (Nhập JSON, link "N cụm chưa thuộc đề nào").
  - `groups-page/` (`/admin/toeic/groups`, mới): bảng cụm chuyển từ trang đầu, thêm lọc "Thuộc đề" (`?test=none` mở sẵn
    "Chưa thuộc đề nào"), mã đề trong bảng là link tới trang đề.
  - `test-page`: "← TOEIC" đổi thành dòng đường dẫn "TOEIC › tên đề".
  - Hàm dùng chung (`STATUS_LABELS`, `reasonsOf`, `byTest`, `testCard`, `bankSummary`) chuyển vào `toeic-admin.logic.ts` (có spec).
- Kiểm thủ công bằng Playwright trên DB tạm `luna_toeic_check_admin`: 37/37 mục đạt (thẻ đề, hộp thoại, bấm thẻ/Enter mở đề,
  đường dẫn, lọc kho, đăng cụm nháp, 360 px không cuộn ngang, axe sáng/tối, không lỗi console).
- **Để sau** (đã bàn với người dùng): nút **Ghép đề tự động** (lấy cụm đã đăng đủ số câu mỗi Part, ưu tiên cụm chưa thuộc đề
  nào, tạo đề nháp) làm cùng đợt 3 khi kho có nhiều cụm hơn một đề; "Nhờ AI soạn cụm" và ảnh minh hoạ Part 6/7 chỉ làm nếu
  người dùng muốn.

## Đợt 2 + thêm (2026-11-20): phần Nghe, ảnh, Ghép đề tự động, AI soạn cụm

- **Ghép đề tự động** (FR-043a): `backend/internal/toeic/compose.go` (`POST /api/admin/toeic/tests/compose`, `dryRun`,
  `partial`; chọn bằng quy hoạch động "tổng câu đúng bằng cần", cụm chưa dùng rẻ nhất), hộp thoại
  `features/admin/toeic/compose-dialog.ts` ở trang đầu (nút + thẻ "Ghép đề tự động").
- **AI soạn cụm** (FR-042b): `ai.ToeicGroups` + prompt tiếng Anh `ai/llm/toeic.go` (op `toeic_groups`, thêm vào
  `aimodel.Ops`), `toeic/draft.go` (`POST /api/admin/toeic/groups/draft`, mã `p{part}-ai-yyMMdd-HHmm-n`), hộp thoại
  `draft-dialog.ts` ở Kho cụm câu. Gọi đồng bộ, không qua hàng đợi job (local và production chung DB).
- **Ảnh** (T064–T067, T075, T076): `toeic/images.go` (vẽ Gemini/Codex bắt buộc chọn, tải lên ≤ 5 MB, xoá, vẽ hàng loạt ảnh
  Part 1 còn thiếu chạy ngầm trong process, 409 khi đang chạy); prompt ảnh chụp thật `ai/toeic_image.go` chỉ gửi cảnh +
  câu đúng; ảnh văn bản Part 6/7 ở ô `doc-n` (mọi văn bản đều có ô, không cần `imageSlot` trong file); op sử dụng
  `toeic_photo`. Giao diện: `image-slots.ts` trong Xem trước cụm, `test-page/photo-panel.ts` trên trang đề.
- **Phần Nghe** (T062–T074): `features/toeic/listening/` (kịch bản đọc + Directions do Luna viết, trình phát tự chạy,
  tạm dừng/nghe lại cụm/tốc độ/bỏ hướng dẫn khi luyện tập; nhảy câu → đầu cụm), `SpeechService.speakAs` + `pickVoice`
  (giọng theo vùng en-US/GB/AU/CA, không dùng giọng tự nhiên), `shared/graphic-view.ts`, phòng thi (Part 1/2 ẩn chữ,
  trang theo âm thanh, lưu `listeningSegment`), đáp án chi tiết (lời thoại bong bóng, tô câu chứa đáp án, bản dịch, nghe
  lại cụm).
- **Nội dung** (T077, Claude soạn): `part1/p1-set-01.json` (12), `part2/p2-set-01.json` (50), `part3/p3-set-01.json`
  (26 cụm, 6 cụm 3 người, 7 biểu đồ), `part4/p4-set-01.json` (20 cụm, 6 biểu đồ); `TestContentFiles` sạch. Chưa nhập DB thật.
- **Kiểm thủ công** bằng Playwright trên DB tạm `luna_toeic_check_admin` (server thử cổng 8090, giọng đọc giả lập):
  ghép đề đủ 200 câu, Mini 50, ảnh Part 1 (tải lên/xoá/đếm, hỏi engine), AI soạn thật 2 cụm Part 2 + 2 cụm Part 5
  (Gemini, ~3 giây), ô ảnh văn bản Part 7, phòng thi phần Nghe (hướng dẫn, tự sang câu, tạm dừng, tốc độ 1.25, nhảy Part
  3, nhiều giọng, biểu đồ, 360 px), lưu listeningSegment, đáp án chi tiết; axe đạt mọi trang. Chưa vẽ ảnh thật bằng AI
  (tốn phí). Ảnh qua `<img>` từ cổng 4200 sang server khác cổng không kèm cookie Strict trong Chrome của Playwright (401 →
  ORB) — chung cho mọi ảnh của app, không riêng TOEIC.
- **Chưa làm**: T078 (thử trên điện thoại thật nhiều giọng); khoá Part 5–7 khi đang nghe ở chế độ thi thật thuộc US5.
- **AI soạn cụm, bản 2 (2026-11-20)**: prompt viết lại kiểu người ra đề (`ai/llm/toeic_prompt.go`: vai trò, mức điểm 450–900,
  quy cách từng Part, quy tắc phương án nhiễu, giải thích/bản dịch, NGÔN NGỮ bắt buộc, tự kiểm tra cuối); schema có enum cho
  loại văn bản/khối/biểu đồ/giọng; mỗi lần ~10 câu (bỏ ô chọn số cụm); bộ lọc **Nguồn** ở Kho cụm câu. Máy chủ tự sửa trước
  khi kiểm tra (`toeic/repair.go`): vị trí evidence, tên loại đồng nghĩa, tách cụm Part 5 nhiều câu, bỏ dòng To/From lặp
  trong thân bài; loại cụm có tiếng Việt trong phần đề. Model mặc định của op `toeic_groups` = gemini-3.6-flash
  (`aimodel.OpDefaults`): thử thật 3.5-flash-lite trích câu không có trong bài và có lần viết cả bài bằng tiếng Việt,
  3.8-flash chất lượng tốt nhưng 90 giây/Part 6 và quá 4 phút với Part 5; 3.6-flash đạt 7/7 Part, 35–56 giây. Hạn ghi
  phản hồi của request soạn được nới (server WriteTimeout 15 giây cắt mất kết quả Part 3).
- **Ảnh (2026-11-20, tiếp)**: ảnh không hiện khi frontend (4200) và API khác cổng là vì `<img>` thường không gửi cookie
  phiên (SameSite=Strict) → 401; thêm `crossorigin="use-credentials"` cho mọi ảnh TOEIC (phòng thi, đáp án, sổ câu sai,
  văn bản, ô ảnh admin) → Playwright: ảnh admin 6/6 và ảnh phòng thi đều hiện. Vẽ thật chưa được: Gemini image model trả
  429, Codex CLI máy này (C:\nvm4w\nodejs\codex.cmd, ghi đè CODEX_BIN chỉ cho server thử) cũng hết lượt; đường lỗi báo đúng
  "Đã hết lượt AI". Ảnh chủ đề / ví dụ ngữ pháp cũ dùng `<img>` không có `crossorigin` → có thể cũng không hiện khi chạy local.
- **Vẽ ảnh nhanh theo Part (2026-11-20)**: Kho cụm câu có khu **Ảnh minh hoạ** (`groups-page/bank-pictures.ts`): mỗi Part
  1/6/7 số ảnh còn thiếu + "Vẽ N ảnh còn thiếu…" (chọn engine mỗi lần, chạy ngầm, tiến độ, Dừng). Backend `toeic/batch.go`:
  `POST /api/admin/toeic/images/batch {part|testKey, engine}`, `GET /api/admin/toeic/images/missing`; Part 1 vẽ ảnh chụp
  mọi cụm (cả nháp, bỏ cụm ẩn), Part 6/7 chỉ vẽ văn bản loại ad/web/article/notice/review (giống đề thật; email, thư, memo,
  hoá đơn, mẫu đơn, lịch, chat không có hình — vẫn thêm tay được ở Xem trước). Cột Ảnh của bảng hiện "a/b" cho Part 6/7.
  Playwright: đếm 20/6/14 ảnh còn thiếu, vẽ hàng loạt Part 7 dừng đúng lý do "hết lượt AI", axe đạt.
- **Vẽ ảnh cho các cụm vừa soạn (2026-11-20)**: sau "Nhờ AI soạn" Part 1/6/7, hộp thoại có nút "Vẽ ảnh cho N cụm vừa
  soạn…" (chọn engine, tiến độ, Dừng, "Vẽ tiếp ảnh còn thiếu…" khi bị dừng); backend `BatchInput.Keys` (≤ 50 cụm, chỉ vẽ ô
  còn thiếu, Part 6/7 chỉ văn bản loại có hình). Đóng hộp thoại thì khu Ảnh minh hoạ tiếp tục theo dõi. Playwright: soạn
  thật 10 cụm Part 1 → nút hiện, vẽ dừng đúng "hết lượt AI", khu Ảnh minh hoạ thấy lượt vẽ.
- **Vẽ ảnh theo mức điểm (2026-11-23)**: bấm "Vẽ ảnh còn thiếu…" của một Part → hộp thoại chọn mức
  (`groups-page/level-dialog.ts`: 450/600/750/900, mỗi mức ghi "Còn N ảnh" hoặc "✓ Đủ ảnh"); chọn mức đã đủ ảnh thì báo
  "Mức … đã đủ ảnh, sẽ không vẽ lại", các mức còn thiếu vẫn vẽ phần thiếu; mọi mức chọn đều đủ thì "Tiếp tục" bị tắt;
  "Tiếp tục" → chọn engine (có "Quay lại" về bước chọn mức, giữ nguyên các mức đã tick; `EngineDialog.backLabel`) → vẽ.
  Backend `BatchInput.Levels`
  (mức lạ → `400 levels`), `GET /images/missing` thêm `byLevel`; tiến độ ghi "Part 1 (mức 450)". Ảnh TOEIC cũng chuyển
  sang prompt ảnh chụp chân thực (`ai/toeic_image.go`, `photoStyle` + `photoAvoid`).
- **Trùng cụm khi AI soạn + mọi văn bản Part 6/7 có ảnh (2026-11-20)**: `toeic/dedup.go` so mỗi cụm AI trả về với cả kho
  của Part (cả nháp) và với cụm khác trong cùng lần soạn (từ giống ≥ 80%, không gọi AI); cụm trùng bị bỏ, báo "Trùng với cụm
  … (giống N%)", hộp thoại tách "bỏ N cụm trùng" khỏi "không qua kiểm tra". Theo yêu cầu admin, vẽ hàng loạt giờ vẽ **mọi**
  văn bản Part 6/7 (bỏ giới hạn ad/web/article/notice/review ghi ở trên); email, thư, memo, hoá đơn, mẫu đơn, lịch, chat được
  vẽ cảnh của tình huống, không vẽ chính văn bản (`ai.docPicturePrompt`).
- **Kho Part 1 theo mức (2026-11-20)**: Claude soạn 120 câu Part 1, mỗi mức 30 câu: `content/toeic/part1/p1-450.json`,
  `p1-600.json`, `p1-750.json`, `p1-900.json` (mã `p1-<mức>-NN`, đáp án chia đều A–D, cảnh không lặp `p1-set-01.json`).
  `TestContentFiles` đạt; so trùng 80% trên 132 câu Part 1: không cặp nào (giống nhất 33%). Đã nhập vào DB thật (nháp), dryRun sạch trước.
- **Kho cụm câu: phân trang + sắp xếp theo mức (2026-11-20)**: bảng cắt trang bằng `lu-pager` (ô "Mỗi trang" 10/30/50/100,
  mặc định 30), ô "Sắp xếp" (theo Part và mã cụm / mức 450 → 900 / 900 → 450, `byLevel` giữ thứ tự cụm cùng mức); đổi bộ
  lọc, thứ tự hay số mỗi trang thì về trang 1. Nút đăng loạt đổi thành "Đăng N cụm nháp theo bộ lọc" (mọi trang, không chỉ
  trang đang xem). Thêm lọc "Mức" (450/600/750/900, `atLevel`; `TOEIC_LEVELS` dùng chung với hộp thoại Nhờ AI soạn).
  Playwright trên DB thật (chỉ đọc): 30 dòng, "1–30 / 209 cụm", trang 2, 10/trang, Part 1 sắp đúng hai chiều, Part 1 + mức 750
  = 30 cụm, axe đạt.
- **Xoá cụm (2026-11-20, FR-042c)**: `POST /api/admin/toeic/groups/delete {keys}` (≤ 500) → `{deleted, blocked[{key, reason}], missing}`;
  `toeic/delete.go` giữ lại cụm đang thuộc đề hoặc có bài làm đã nộp, xoá ảnh rồi `GroupRepository.Delete` (Mongo `DeleteMany`,
  MySQL `DELETE … IN`). Kho cụm câu: cột ô chọn (cả trang, có trạng thái nửa chọn), thanh "Đã chọn N cụm · Xoá N cụm đã chọn… ·
  Bỏ chọn", nút "Xoá" từng dòng, `lu-confirm-dialog` kiểu nguy hiểm, báo kết quả và lý do giữ lại. Playwright trên server thử
  8090 (DB tạm): xoá 2 cụm đã chọn, xoá 1 cụm, cụm thuộc đề bị giữ với lý do, axe đạt khi hộp thoại mở.
- **Kho Part 2 theo mức (2026-11-20)**: Claude soạn 120 câu Part 2, mỗi mức 30 câu: `content/toeic/part2/p2-450.json`,
  `p2-600.json`, `p2-750.json`, `p2-900.json` (mã `p2-<mức>-NN`, đáp án 10/10/10 mỗi mức; độ khó tăng: 450 trả lời thẳng,
  600–750 thêm câu trả lời gián tiếp, câu hỏi phủ định/đuôi, 900 phần lớn gián tiếp và thành ngữ). `TestContentFiles` đạt;
  so trùng 80% trên 170 câu Part 2: không cặp nào (giống nhất 45%). Đã nhập vào DB thật (nháp), dryRun sạch trước.
- **Kho Part 3 theo mức (2026-11-20)**: 120 hội thoại Part 3 (360 câu), mỗi mức 30: `content/toeic/part3/p3-450.json`,
  `p3-600.json`, `p3-750.json`, `p3-900.json` (4 agent soạn song song, chủ đề chia theo mức). Mỗi mức: 8 cụm có biểu đồ, 7 cụm
  3 người nói, 10 câu hàm ý, đáp án A–D 22–23 mỗi vị trí, lời thoại 90–119 từ. `TestContentFiles` đạt; kiểm tra lại độc lập:
  evidence/câu trích khớp nguyên văn, lời giải nhắc đúng chữ đáp án, mỗi lượt có bản dịch, giọng không đổi trong cụm; so trùng
  146 hội thoại Part 3: không cặp nào ≥ 50% (giống nhất 24%). Đã nhập vào DB thật (nháp), dryRun cả 4 file sạch trước.
- **Kho Part 4 + Part 5 theo mức (2026-11-20)**: 120 bài nói Part 4 (`content/toeic/part4/p4-450|600|750|900.json`, 360 câu;
  mỗi mức 7–8 cụm biểu đồ, 10 câu hàm ý, 1 người nói/1 giọng, 101–134 từ) và 120 câu Part 5 (`content/toeic/part5/p5-450|600|750|900.json`,
  trải đều từ loại/thì/giới từ/liên từ/đại từ/quan hệ/so sánh/từ vựng-kết hợp từ; mức 900 có đảo ngữ, giả định, at the discretion of…),
  6 agent soạn song song. Kiểm lại độc lập: `TestContentFiles` đạt, evidence/câu trích khớp, lời giải nhắc đúng chữ đáp án, A–D đều;
  không trùng (Part 4 giống nhất 27%, Part 5 giống nhất 31%, kể cả nội dung đã có trong DB). Còn: ở mức 750–900 (Part 3 và 4) đáp án
  thường là phương án dài nhất (31–48%). Đã sửa 7 file (chỉ đổi chữ phương án, 141 phương án; mọi trường khác giữ nguyên, đối chiếu
  bản gốc): còn 12–19/90 mỗi file. Đã nhập DB thật: Part 3 thay thế (120 cụm, vẫn nháp, 0 bài làm chấm lại), Part 4 + Part 5
  tạo mới (nháp); dryRun cả 12 file sạch trước.
- **Kho Part 6 + Part 7 theo mức (2026-11-20)**: 120 văn bản Part 6 (`content/toeic/part6/p6-450|600|750|900.json`, 480 câu,
  mỗi văn bản 1 câu chèn câu + 1 từ nối + 1 từ vựng + 1 ngữ pháp, A–D 30/30/30/30) và 120 cụm Part 7
  (`content/toeic/part7/p7-450|600|750|900.json`, 444 câu; mỗi mức 20 cụm 1 văn bản + 6 cụm 2 văn bản + 4 cụm 3 văn bản, câu
  cross-reference, từ trong ngữ cảnh, hàm ý chat theo giờ), 8 agent song song. Kiểm lại độc lập: `TestContentFiles` đạt, blank
  [[1]]–[[4]] đúng thứ tự, không dòng tiêu đề trong body, evidence/câu trích/giờ chat khớp, lời giải nhắc đúng chữ đáp án; đáp án
  là phương án dài nhất 1–19%, ngắn nhất 9–20%; không trùng (giống nhất 24% / 20%, kể cả nội dung trong DB). Đã nhập DB thật
  (nháp), dryRun cả 8 file sạch trước; ảnh còn thiếu sau khi nhập: Part 1 130, Part 6 124, Part 7 199.
