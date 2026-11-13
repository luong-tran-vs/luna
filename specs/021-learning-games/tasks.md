---

description: "Task list for F26 Trò chơi — vừa chơi vừa học"
---

# Tasks: Trò chơi — vừa chơi vừa học (F26)

**Input**: `specs/021-learning-games/` — spec.md, plan.md, research.md (R1–R8), data-model.md, contracts/api.md, quickstart.md

**Tests**: BẮT BUỘC theo constitution VI: unit test cho logic nghiệp vụ, test endpoint (thành công, đầu vào sai, sai quyền),
không cần mạng (AI giả lập). Mỗi tiêu chí nghiệm thu có một test hoặc một bước kiểm tra thủ công (quickstart.md).

**Quy ước của dự án (đọc trước khi làm)**:
- Tên trong code, API, commit bằng tiếng Anh; chữ trên giao diện, tài liệu bằng tiếng Việt. Prompt gửi AI bằng tiếng Anh.
- Backend: domain package + interface cho DB/AI, DI bằng constructor viết tay; nối trong `backend/internal/service/`. Làm
  repository cho **cả** Mongo (`internal/storage/mongo`) và MySQL (`internal/storage/mysql`). Dev và production chung một
  DB: chỉ thêm collection/bảng/cột, không đổi hay xoá cái đang có.
- Frontend: Angular 21 standalone, signals, `input()`/`output()`, `ChangeDetectionStrategy.OnPush`, control flow `@if/@for`,
  không `ngClass/ngStyle`; màu, chữ chỉ dùng token trong `frontend/src/styles/tokens.css` (theo `docs/design-system.md`).
- Đổi biến môi trường thì sửa cả `backend/.env` và `backend/.env.example` (tính năng này không cần biến mới).
- Không tự commit.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: làm song song được (khác file, không phụ thuộc task chưa xong)
- **[Story]**: US1 … US6 theo spec.md

---

## Phase 1: Setup

**Purpose**: khung package và kiểu dùng chung

- [ ] T001 Tạo package `backend/internal/games/` với `games.go`: kiểu `Game` và các hằng `picture|verbs|letters|memory|grammar|daily`, bảng `MaxScore` (picture 30000, verbs 500, letters 20000, memory 10000, grammar 5000, daily 3000), lỗi `ErrNotFound`, `ErrExists`, `ErrNotEnough` (`not_enough_words`/`not_enough_sentences`), `ErrAlreadyDone`, `ValidationError{Fields map[string]string}` (thông báo tiếng Việt), và các kiểu câu hỏi `PictureQuestion`, `LetterWord`, `MemoryPair`, `SpotQuestion`, `Span`, `DailyQuestion`, `DailyVerb`, `Best`, `DailyResult`, `DailyAnswer`, `SpotItem`, `SpotCount` đúng như `specs/021-learning-games/data-model.md`
- [ ] T002 [P] Tạo `frontend/src/app/core/models/games.ts`: kiểu TypeScript cho mọi request/response trong `specs/021-learning-games/contracts/api.md` (`GameId`, `GamesList`, `PictureRound`, `LettersRound`, `MemoryRound`, `SpotQuestion`, `GrammarRound`, `GameResultBody`, `GameResultReply`, `DailyView`, `DailyQuestion`, `DailyResultBody`, `SpotItem`, `SpotCounts`)

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: phần mọi trò cần: interface phụ thuộc, kỷ lục, nhận kết quả, đưa mục sai về hàng ôn, trang danh sách, khung chơi

**⚠️ CRITICAL**: xong phase này mới làm các user story

### Backend

- [ ] T003 Viết `backend/internal/games/deps.go`: interface `BestRepository`, `DailyRepository`, `SpotRepository` (đúng chữ ký ở data-model.md) và các cổng ra ngoài: `Cards` (lemma, text, meaningVi, lessonId, due của thẻ người học), `Pictures` (`ImageURLs(ctx, lemmas) (map[string]string, error)`), `WordBankMeanings` (`MeaningsOf(ctx, lemmas) (map[string]string, error)`), `LessonWords` (`Vocabulary(ctx, lessonID)` → lemma, text, meaningVi), `StudiedLessons` (`StudiedLessons(ctx, userID) ([]string, error)`), `StarterLesson` (`StarterLesson(ctx, userID) (lessonID string, ok bool, err error)`), `LessonAccess` (`CanOpen(ctx, userID string, isAdmin bool, lessonID string) (bool, error)`), `WordMisses` (`GameMisses(ctx, userID string, lemmas []string) (int, error)`), `VerbMisses` (`GameMisses(ctx, userID string, misses []VerbMiss) error`), `VerbProgress` (`LearnedBases(ctx, userID) (due, other []string, err error)`), `GrammarSource`, `GrammarProgress`, `SpotWriter` (AI), `StudyDays`, `Timezones`
- [ ] T004 Viết `backend/internal/games/service.go`: `type Service struct` giữ các cổng ở T003 + `now func() time.Time` + `rand` gieo được (để test), `NewService(Deps) *Service`
- [ ] T005 Thêm vào `backend/internal/vocab/misses.go` hàm `func (s *Service) GameMisses(ctx context.Context, userID string, lemmas []string) (int, error)` theo research R1: tối đa 60 lemma (`ValidationError` khi vượt), chuẩn hoá chữ thường + `collapse`, mỗi lemma một lần, chỉ kéo `Due = now` cho thẻ đã có và đang hẹn sau `now` (giữ trạng thái FSRS, dùng `effective` + `UpdateSchedule` như nhánh thẻ có sẵn của `PracticeMisses`), lemma chưa có thẻ thì bỏ qua; trả số thẻ đã kéo
- [ ] T006 [P] Test `GameMisses` trong `backend/internal/vocab/misses_test.go`: thẻ hẹn ngày mai → về `now`, thẻ đã đến hạn → giữ, lemma không có thẻ → không tạo, lặp lemma → một lần, 61 lemma → lỗi, trạng thái FSRS (Stability, Reps, State) không đổi
- [ ] T007 Viết `backend/internal/games/results.go`: `func (s *Service) Finish(ctx, userID string, isGuest bool, game Game, r Result) (FinishReply, error)` với `Result{Score int; Completed bool; WrongWords []string; WrongVerbs []VerbMiss}`: kiểm tra `0 ≤ Score ≤ MaxScore[game]`, `len(WrongWords) ≤ 60`, `len(WrongVerbs) ≤ 50`, `Form ∈ {past, participle}`, `game ≠ daily`, `game == grammar && isGuest` → lỗi quyền; nếu `Completed` thì `bests.Raise`; luôn gọi `WordMisses.GameMisses` và `VerbMisses.GameMisses` khi có; trả `{Best, NewBest, Rescheduled}`
- [ ] T008 [P] Test `results.go` trong `backend/internal/games/results_test.go` (fake repo ở `backend/internal/games/fake_test.go`): điểm âm/vượt trần → `ValidationError`; `Completed=false` → không đổi kỷ lục nhưng vẫn kéo hạn; điểm thấp hơn kỷ lục → `NewBest=false`; khách + grammar → lỗi quyền
- [ ] T009 Viết `backend/internal/games/handler.go`: `NewHandler(svc, log)`, `Register(mux, requireAuth)` với route `GET /api/games` và `POST /api/games/{game}/results` (đọc `httpx.PrincipalFrom`, quyền khách lấy từ principal như `httpx.RequireMember` đang làm), map lỗi: `ValidationError` → `httpx.WriteFieldErrors`, lỗi quyền → `403 members_only`, `ErrNotEnough` → `409 not_enough_words|not_enough_sentences`, `ErrAlreadyDone` → `409 already_done`, còn lại `500` + log như `verbs.Handler.fail`
- [ ] T010 Viết `func (s *Service) List(ctx, userID string, isGuest bool) (ListView, error)` trong `backend/internal/games/list.go`: danh sách trò đã phát hành (biến `released []Game` trong code, thêm dần theo phase), kỷ lục từng trò, `locked` cho grammar khi là khách, `daily{day, done, streak}` (để trống cho tới US6)
- [ ] T011 [P] Test handler trong `backend/internal/games/handler_test.go`: `GET /api/games` 200 + 401 khi chưa đăng nhập; `POST /api/games/picture/results` 200, 400 (điểm vượt trần, game lạ), 403 (khách + grammar)
- [ ] T012 Thêm repository kỷ lục: interface vào `backend/internal/storage/storage.go` (`GameBests() games.BestRepository`); Mongo `backend/internal/storage/mongo/games.go` (collection `game_bests`, `Raise` bằng `$max` + upsert, `at` chỉ đổi khi điểm tăng — đọc trước rồi `UpdateOne` có điều kiện `score < new`) + chỉ mục duy nhất `(userId, game)` trong `backend/internal/storage/mongo/indexes.go`; nối trong `backend/internal/storage/mongo/store.go`
- [ ] T013 [P] MySQL: `backend/internal/storage/mysql/games.go` (bảng `game_bests(user_id, game, score, at, PRIMARY KEY(user_id, game))`, `INSERT … ON DUPLICATE KEY UPDATE` chỉ khi điểm lớn hơn), migration `{ID: "045_games", Stmts: gamesSchema()}` trong `backend/internal/storage/mysql/migrate.go` (tạo cả 3 bảng `game_bests`, `daily_challenges`, `grammar_spot_items` một lần), nối trong `backend/internal/storage/mysql/mysql.go`; test `backend/internal/storage/mysql/games_test.go` theo kiểu `verb_progress_test.go`
- [ ] T014 Đưa `game_bests` vào file xuất và xoá tài khoản: hằng `CollGameBests = "game_bests"` + trường `GameBests []Doc \`json:"gameBests"\`` trong `backend/internal/export/model.go`, thêm vào danh sách trong `backend/internal/export/service.go`, khoá MySQL trong `backend/internal/storage/mysql/export.go`, danh sách xoá trong `backend/internal/storage/{mongo,mysql}/users.go`; sửa test export liên quan
- [ ] T015 Viết `backend/internal/service/games.go`: `initGames()` tạo `games.NewService` với adapter: `Cards` qua `c.store.Cards()`/vocab, `Pictures` qua `c.wordBank.ImageURLs`, `WordBankMeanings` qua `c.wordBank.Words`, `LessonWords` qua lesson reader (giống adapter `LessonVocabulary` của vocab), `StudiedLessons`/`StarterLesson`/`LessonAccess` qua `c.study`, `WordMisses` qua `c.vocab`, `StudyDays` qua `c.studyDays`, `Timezones` như vocab; gọi `initGames()` trong `backend/internal/service/service.go` **sau** `initVocab`, `initVerbs`, `initGrammar`, `initProgress`; đăng ký route trong `backend/internal/service/routes.go`
- [ ] T016 [P] Viết `backend/internal/games/pool.go`: `func (s *Service) wordPool(ctx, userID string) ([]PoolWord, error)` theo R2 — thẻ của người học (đến hạn trước, rồi theo `LessonID`), thiếu thì từ vựng các bài đã học, chưa có bài thì bài khởi đầu; gắn `ImageURL` và `MeaningVi` (thẻ trước, Kho sau); hàm thuần `spellable(lemma)` (một từ, a–z, 3–8 ký tự) và `sameForm(a, b)` (cùng lemma hoặc khác nhau bởi đuôi s/es/ed/ing)
- [ ] T017 [P] Test `pool.go` trong `backend/internal/games/pool_test.go`: thứ tự ưu tiên, fallback 2 tầng, `spellable`, `sameForm`, bỏ từ trùng nghĩa

### Frontend

- [ ] T018 Viết `frontend/src/app/core/services/games-api.service.ts` (`providedIn: 'root'`, `inject(HttpClient)`): các hàm theo contracts/api.md; hàng đợi kết quả chưa gửi trong `localStorage` khoá `luna.games.pending` (mọi đọc/ghi bọc try/catch), `flushPending()` gửi lại lần lượt, bỏ mục bị 400/403/409
- [ ] T019 [P] Viết `frontend/src/app/features/games/shared/game-clock.logic.ts` (hàm thuần: `start`, `pause`, `resume`, `remaining(state, nowMs)` dựa trên `performance.now()`, không cộng dồn `setInterval`) và test `game-clock.logic.spec.ts`
- [ ] T020 [P] Viết `frontend/src/app/features/games/shared/game-score.logic.ts` (`nextCombo`, `pictureScore(combo)` = 100 × combo, combo tối đa 5, `isNewBest`) và test `game-score.logic.spec.ts`
- [ ] T021 Viết `frontend/src/app/features/games/shared/game-shell.ts|html|css`: khung màn chơi toàn màn hình — hàng pill trên cùng (slot nội dung), thanh thời gian, nút đóng (hỏi "Thoát lượt này?"), màn **Tạm dừng** ("Tiếp tục" / "Thoát") khi `document.visibilitychange` ẩn trang (`visibility-pause.ts`), vùng nội dung chính ở nửa dưới màn hình, nút ≥ 44 px
- [ ] T022 Viết `frontend/src/app/features/games/shared/game-result.ts|html|css` (FR-002): điểm, số đúng/sai, kỷ lục ("Kỷ lục mới" khi vượt), danh sách mục sai kèm nghĩa/đáp án, nút "Chơi lại" và "Về danh sách trò"; báo nhẹ khi gửi kết quả thất bại (đã lưu để gửi sau)
- [ ] T023 Viết trang `frontend/src/app/features/games/games.ts|html|css` (`/games`): gọi `flushPending()` rồi `GET /api/games`, mỗi trò một thẻ (tên, mô tả một dòng, "1–3 phút", kỷ lục), trò khoá có nhãn "Chỉ dành cho thành viên"; chỗ cho thẻ Thử thách ở đầu (US6)
- [ ] T024 Thêm route lazy `games` và `games/:game` trong `frontend/src/app/app.routes.ts` (title tiếng Việt `… · Luna`)
- [ ] T025 Trong `frontend/src/app/layouts/learner-layout/learner-layout.ts`: thêm `'/games'` vào `isPracticePath`, thêm `/^\/games\/[^/]+$/` vào `isFocusPage`; sửa test `learner-layout.spec.ts` tương ứng
- [ ] T026 Thêm ô `{ label: 'Trò chơi', icon: …, link: '/games', note: 'Vừa chơi vừa học', badge: 0 }` vào danh sách ô trong `frontend/src/app/layouts/learner-layout/practice-menu/practice-menu.ts` (dùng icon có sẵn của `lu-icon`; chưa có icon phù hợp thì thêm một icon nét mảnh cùng kiểu trong component icon); sửa `practice-menu.spec.ts`

**Checkpoint**: trang `/games` mở được từ Luyện, chưa có trò nào chơi được

---

## Phase 3: User Story 1 - Ảnh nhanh tay (Priority: P1) 🎯 MVP

**Goal**: lượt 60 giây nhìn ảnh chọn từ, combo, kết quả, kỷ lục, từ sai về hàng ôn

**Independent Test**: quickstart.md bước 1–4

### Tests

- [ ] T027 [P] [US1] Test `backend/internal/games/picture_test.go`: 40 câu, mỗi câu 4 đáp án khác nhau, đúng 1 đáp án khớp ảnh, không có đáp án nhiễu cùng dạng/cùng nghĩa, không lặp lemma khi bể đủ lớn, bể < 4 từ có ảnh → `ErrNotEnough`
- [ ] T028 [P] [US1] Test handler `GET /api/games/picture/round` trong `backend/internal/games/handler_test.go`: 200, 409 `not_enough_words`, 401
- [ ] T029 [P] [US1] Test `frontend/src/app/features/games/picture-quiz/picture-quiz.logic.spec.ts`: đúng → combo +1 (tối đa x5) và điểm = 100 × combo; sai → combo x1, ghi từ sai một lần; hết giờ → kết thúc

### Implementation

- [ ] T030 [US1] Viết `backend/internal/games/picture.go`: `func (s *Service) PictureRound(ctx, userID) ([]PictureQuestion, error)` theo FR-010, FR-012, R2 (đáp án nhiễu ưu tiên cùng `LessonID`, trộn vị trí đáp án)
- [ ] T031 [US1] Thêm route `GET /api/games/picture/round` vào `backend/internal/games/handler.go`; thêm `picture` vào `released`
- [ ] T032 [P] [US1] Viết `frontend/src/app/features/games/picture-quiz/picture-quiz.logic.ts` (trạng thái lượt thuần: câu hiện tại, combo, điểm, đúng/sai, từ sai)
- [ ] T033 [US1] Viết component `frontend/src/app/features/games/picture-quiz/picture-quiz.ts|html|css` theo `design/de-xuat-tro-choi.gif` phần 1: pill ⏱/🔥 combo/điểm, ảnh trong thẻ, "Đây là gì?", lưới 2×2 đáp án ở nửa dưới, đúng → xanh + ✓ + điểm bay lên, sai → đỏ + ✗ và tô đáp án đúng ~1 giây; 60 giây dùng `game-clock`; hết giờ → `POST /api/games/picture/results` (`completed: true`) rồi `game-result`; thoát giữa chừng → gửi `completed: false` với từ đã sai
- [ ] T034 [US1] Viết `frontend/src/app/features/games/game-host.ts` (route `games/:game`, chọn component theo tham số, trò chưa phát hành → về `/games`) và gắn `picture`; ảnh câu hỏi dùng `alt="Ảnh câu hỏi"` (không ghi từ hay nghĩa vào `alt` để khỏi lộ đáp án)
- [ ] T035 [US1] Kiểm tra thủ công quickstart.md bước 1–4 ở 360 px, sáng và tối

**Checkpoint**: MVP — phát hành được riêng

---

## Phase 4: User Story 2 - Đấu động từ (Priority: P2)

**Goal**: gõ V2, V3 cho V1, 3 mạng, chuỗi, kỷ lục; động từ sai đến hạn ôn hôm nay, gõ đúng không đổi lịch

**Independent Test**: quickstart.md bước 5

### Tests

- [ ] T036 [P] [US2] Test `backend/internal/verbs/verbs_test.go`: `Result{Game: true, Right: false}` cho động từ đã học hẹn sau `now` → `DueAt = now`, `Level` giữ nguyên, `Wrong+1`, lỗi gõ được giữ; `Game` + đúng → chỉ `Right+1`, `DueAt` và `Level` giữ nguyên; động từ chưa học + sai → chỉ đếm
- [ ] T037 [P] [US2] Test `frontend/src/app/features/games/verb-duel/verb-duel.logic.spec.ts`: chấp nhận mọi dạng trong `past`/`participle` có `/` (learned/learnt), không phân biệt hoa thường, bỏ khoảng trắng thừa; sai một ô → mất 1 mạng; hết 3 mạng → kết thúc; thứ tự động từ: đến hạn và hay sai → đã học → thông dụng chưa học; < 10 đã học thì bổ sung thông dụng
- [ ] T038 [P] [US2] Test `backend/internal/games/results_test.go`: `wrongVerbs` được chuyển cho `VerbMisses` đúng base/form/typed

### Implementation

- [ ] T039 [US2] Thêm `Game bool` vào `verbs.Result` và nhánh `case r.Game:` trong `ApplyResult` (`backend/internal/verbs/verbs.go`) theo R1; nhận trường `game` trong `resultsRequest` của `backend/internal/verbs/handler.go`
- [ ] T040 [US2] Thêm `func (s *Service) GameMisses(ctx, userID string, misses []games.VerbMiss)` — hoặc adapter trong `backend/internal/service/games.go` — gọi `verbs.Service.Record` với `Result{Base, Right: false, Game: true, Wrong: []Mistake{{Form, Typed}}}` (gộp theo base); nối vào `games.Deps.VerbMisses`
- [ ] T041 [P] [US2] Viết `frontend/src/app/features/games/verb-duel/verb-duel.logic.ts`: xếp thứ tự động từ từ `GET /api/verbs/progress` + `irregular-verbs.data.ts` (dùng lại hàm ở `verb-mistakes.logic.ts` cho "hay sai"), chấm câu, mạng, chuỗi
- [ ] T042 [US2] Viết component `frontend/src/app/features/games/verb-duel/verb-duel.ts|html|css` theo gif phần 3: pill mạng/chuỗi/🏆 kỷ lục, thẻ ảnh động từ (dùng `verb-pictures.service.ts`), V1 chữ lớn + nghĩa, 2 ô gõ V2/V3 (`autocapitalize="off"`, `autocorrect="off"`, `spellcheck="false"`, Enter sang ô sau rồi kiểm tra), sai → ô đỏ + ✗ + khối "Đáp án: V2 … · V3 …"; ô gõ không bị bàn phím che (cuộn vào vùng nhìn); hết mạng → gửi `POST /api/games/verbs/results` + `game-result`
- [ ] T043 [US2] Thêm `verbs` vào `released` (`backend/internal/games/list.go`) và vào `game-host.ts`
- [ ] T044 [US2] Kiểm tra thủ công quickstart.md bước 5

---

## Phase 5: User Story 3 - Chữ rơi (Priority: P3)

**Goal**: chạm chữ rơi theo thứ tự để ghép từ đúng nghĩa, màn tăng tốc, 3 mạng

**Independent Test**: quickstart.md bước 6

### Tests

- [ ] T045 [P] [US3] Test `backend/internal/games/letters_test.go`: 25 từ, chỉ từ `spellable`, không lặp, < 5 từ → `ErrNotEnough`
- [ ] T046 [P] [US3] Test `frontend/src/app/features/games/letter-drop/letter-drop.logic.spec.ts`: chữ lặp (apple) tính đúng thứ tự; chạm sai → mất mạng, không điền; chữ cần chạm rơi hết → mất mạng và chữ đó rơi lại; 5 từ = 1 màn; màn sau tốc độ và số chữ nhiễu tăng; chữ nhiễu không lấy từ chữ còn thiếu của từ

### Implementation

- [ ] T047 [US3] Viết `backend/internal/games/letters.go` (`LettersRound`) và route `GET /api/games/letters/round`; thêm `letters` vào `released`
- [ ] T048 [P] [US3] Viết `frontend/src/app/features/games/letter-drop/letter-drop.logic.ts`: sinh chữ rơi (vị trí x, thời lượng rơi theo màn, chữ nhiễu), xử lý chạm, mạng, điểm theo từ và màn
- [ ] T049 [US3] Viết component `frontend/src/app/features/games/letter-drop/letter-drop.ts|html|css` theo gif phần 2: nghĩa trong thẻ trên cùng, vùng rơi, mỗi chữ là `button` chạy `@keyframes` `translateY` (chỉ `transform`), `animationend` = rơi hết; `@media (prefers-reduced-motion: reduce)` dùng `steps()`; các ô chữ ở đáy; ghép xong → thẻ ảnh + "house ✓" ~1 giây; tạm dừng thì `animation-play-state: paused`
- [ ] T050 [US3] Thêm `letters` vào `game-host.ts`; kiểm tra thủ công quickstart.md bước 6 (cả khi bật giảm chuyển động)

---

## Phase 6: User Story 4 - Lật thẻ ghi nhớ (Priority: P4)

**Goal**: lưới 12 thẻ ghép ảnh/nghĩa ↔ từ, tính lượt và thời gian; lối vào từ màn xong bài

**Independent Test**: quickstart.md bước 7

### Tests

- [ ] T051 [P] [US4] Test `backend/internal/games/memory_test.go`: đúng 6 cặp; có `lessonId` → từ của bài, người học không mở được bài → lỗi quyền; không `lessonId` → từ đến hạn trước; từ không có ảnh → cặp dùng `meaningVi`; < 6 từ → `ErrNotEnough`
- [ ] T052 [P] [US4] Test handler `GET /api/games/memory/round?lessonId=` 200/403/409 trong `backend/internal/games/handler_test.go`
- [ ] T053 [P] [US4] Test `frontend/src/app/features/games/memory-match/memory.logic.spec.ts`: lật 2 thẻ khớp → giữ; lệch → úp lại, chạm thẻ thứ ba thì úp ngay cặp lệch và lật thẻ mới; đếm lượt; điểm giảm theo lượt và thời gian; xong 6 cặp → kết thúc

### Implementation

- [ ] T054 [US4] Viết `backend/internal/games/memory.go` (`MemoryRound(ctx, userID, isAdmin, lessonID)`) và route; thêm `memory` vào `released`
- [ ] T055 [P] [US4] Viết `frontend/src/app/features/games/memory-match/memory.logic.ts`
- [ ] T056 [US4] Viết component `frontend/src/app/features/games/memory-match/memory-match.ts|html|css` theo gif phần 4: lưới 3×4, mặt sau màu primary, lật bằng `rotateY` (giảm chuyển động: đổi mặt ngay), khớp → viền xanh + ✓ rồi mờ đi, lệch → viền đỏ + ✗ + rung nhẹ; đọc `lessonId` từ query
- [ ] T057 [US4] Thêm lối vào "Chơi Lật thẻ với từ của bài này" (`/games/memory?lessonId=…`) ở màn xong bài trong `frontend/src/app/features/lesson/lesson-detail/lesson-detail.html|ts`; thêm `memory` vào `game-host.ts`
- [ ] T058 [US4] Kiểm tra thủ công quickstart.md bước 7

---

## Phase 7: User Story 5 - Bắn lỗi ngữ pháp (Priority: P5)

**Goal**: chạm chữ sai, chọn cách sửa hoặc "Không có lỗi", quy tắc + nhãn chủ điểm; admin nhờ AI soạn thêm câu và duyệt

**Independent Test**: quickstart.md bước 8–9

### Tests

- [ ] T059 [P] [US5] Test `backend/internal/games/spot_test.go`: (a) `choice` → câu có lỗi dùng phương án sai, 3 phương án sửa gồm đáp án, câu đúng khi điền đáp án, bỏ bài tập không có đúng một `___`; (b) `fix` và `Mistakes` → diff đoạn khác 1–3 từ được nhận, 4 từ bị bỏ, chữ nhiễu thứ ba lấy từ câu khác cùng điểm, không có thì bỏ câu; (c) câu soạn thêm đã duyệt dùng được, nháp không dùng; lượt 10 câu, ~3 câu không lỗi, ưu tiên điểm người học có tiến độ, thiếu thì cùng trình độ rồi trình độ thấp hơn
- [ ] T060 [P] [US5] Test `backend/internal/games/spot_admin_test.go`: `ValidateSpot` (chỗ sai xuất hiện đúng một lần, chữ đúng ≠ chữ sai, 2 chữ nhiễu khác nhau và khác chữ đúng, câu 3–20 từ); sinh bằng AI giả: câu hỏng bị bỏ và đếm `dropped`, câu tốt lưu `draft`; AI lỗi `ErrNotConfigured` → trả lỗi, không lưu; đếm theo nguồn (a)/(b)/(c)/nháp
- [ ] T061 [P] [US5] Test handler: `GET /api/games/grammar/round` 200 (thành viên), 403 `members_only` (khách); route admin spot-items 200 (admin), 403 (thành viên), 400 (sửa không hợp lệ)
- [ ] T062 [P] [US5] Test `frontend/src/app/features/games/grammar-spot/grammar-spot.logic.spec.ts`: chạm đúng chỗ sai (bất kỳ token trong `wrong`) → hiện phương án; chạm sai chỗ hoặc bấm "Không có lỗi" khi có lỗi → mất mạng và lộ đáp án; câu đúng + "Không có lỗi" → đúng; hết giờ câu → sai

### Implementation — nguồn câu

- [ ] T063 [US5] Thêm `func (s *Service) PublishedSources(ctx) ([]PointSource, error)` vào `backend/internal/grammar/` (file mới `sources.go`): mỗi điểm có bài `StatusPublished` → `{PointID, Level, TitleVi, Exercises: Practice+Mastery, Mistakes: Content.Mistakes + MistakesOf(pointID)}`; test trong `sources_test.go`
- [ ] T064 [US5] Viết `backend/internal/games/spot.go`: tách token (giữ dấu câu là token riêng), `diffSpan(wrong, right []string)`, dựng câu từ nguồn (a)/(b)/(c) theo R4, `GrammarRound(ctx, userID)`; adapter `GrammarSource` và `GrammarProgress` (đọc `grammar.ProgressRepository.List`) trong `backend/internal/service/games.go`
- [ ] T065 [US5] Route `GET /api/games/grammar/round` bọc `httpx.RequireMember`; thêm `grammar` vào `released`

### Implementation — câu soạn thêm (admin)

- [ ] T066 [P] [US5] Repository `grammar_spot_items`: thêm `GrammarSpotItems() games.SpotRepository` vào `backend/internal/storage/storage.go`; Mongo trong `backend/internal/storage/mongo/games.go` (+ chỉ mục `(pointId, status)`); MySQL trong `backend/internal/storage/mysql/games.go` (bảng đã tạo ở migration 045, `distractors` lưu JSON); test MySQL
- [ ] T067 [P] [US5] AI: `backend/internal/ai/spot.go` (`GrammarSpotRequest{Level, TitleEn, Pattern, Examples, Existing []string, Count int}`, `GrammarSpotItem{Sentence, Wrong, Right, Distractors, RuleVi}`, `Disabled.GrammarSpotItems` trả `ErrNotConfigured`), thêm `GrammarSpotItems` vào `ai.Provider` trong `backend/internal/ai/ai.go`; `backend/internal/ai/llm/spot.go` gọi `c.generate(ctx, "grammar_spot", prompt, schema, 0.3, …)`, prompt **tiếng Anh**: viết tối đa N câu ở đúng trình độ, mỗi câu có đúng một lỗi của điểm ngữ pháp, chỗ sai là 1–3 từ liền nhau, 2 chữ nhiễu cũng sai trong câu đó, quy tắc ngắn tiếng Việt, không trùng các câu đã có; sửa mọi fake `ai.Provider` trong test cho biên dịch được
- [ ] T068 [US5] Viết `backend/internal/games/spot_admin.go`: `SpotCounts(ctx)`, `SpotItems(ctx, pointID)`, `GenerateSpots(ctx, pointID)` (1 request, timeout như `grammar.Generate`, `ValidateSpot`, lưu `draft`), `UpdateSpot`, `ApproveSpot`, `DeleteSpot`; route admin `/api/admin/games/spot-counts`, `/api/admin/games/spot-items…` bọc `httpx.RequireAdmin`; lỗi AI trả cùng mã như handler `generate` của grammar
- [ ] T069 [US5] Thêm op `grammar_spot` (tên "Soạn câu Bắn lỗi", 1 request mỗi lần bấm) vào `frontend/src/app/features/admin/ai-usage/ai-usage-catalog.ts`
- [ ] T070 [US5] Admin UI: cột "Câu Bắn lỗi" (tổng, đánh dấu khi < 10) trong `frontend/src/app/features/admin/grammar-list/grammar-list.html|ts`; khối "Câu Bắn lỗi" ở trang bài ngữ pháp của admin (số câu theo nguồn, nút "Nhờ AI soạn thêm" có trạng thái đang chạy, danh sách câu với Sửa / Duyệt / Xoá, nhãn Nháp/Đã duyệt); gọi API qua `frontend/src/app/features/admin/admin-api.service.ts`; sửa spec của các component này

### Implementation — màn chơi

- [ ] T071 [P] [US5] Viết `frontend/src/app/features/games/grammar-spot/grammar-spot.logic.ts`
- [ ] T072 [US5] Viết component `frontend/src/app/features/games/grammar-spot/grammar-spot.ts|html|css` theo `design/de-xuat-tro-choi-2.gif` phần 1: pill câu n/10, mạng, điểm; thanh giờ từng câu; thẻ câu với nhãn chủ điểm và các token là `button`; chạm chỗ sai → token đỏ gạch ngang + bảng trượt từ dưới lên "Sửa “go” thành:" 3 phương án; đúng → token xanh, thẻ "💡 quy tắc"; nút "Không có lỗi" ở đáy; 10 câu → `POST /api/games/grammar/results` + `game-result`
- [ ] T073 [US5] Khách: thẻ Bắn lỗi ở `/games` khoá, mở `/games/grammar` thì hiện "Chỉ dành cho thành viên"; thêm `grammar` vào `game-host.ts`
- [ ] T074 [US5] Kiểm tra thủ công quickstart.md bước 8–9

---

## Phase 8: User Story 6 - Thử thách hằng ngày (Priority: P6)

**Goal**: 10 câu rút gọn mỗi ngày, 20 giây/câu, chỉ lượt đầu tính, chuỗi ngày, so sánh với chính mình, chia sẻ, thẻ trang chủ

**Independent Test**: quickstart.md bước 10–11

### Tests

- [ ] T075 [P] [US6] Test `backend/internal/games/daily_test.go`: cùng người + cùng ngày → cùng bộ câu, khác ngày → khác; tỉ lệ 3/3/2/2 (khách 4/3/3, không có grammar); trò thiếu nội dung → phần thiếu chia cho trò khác; `streakOf` (liên tiếp tới hôm nay, tới hôm qua khi hôm nay chưa chơi, đứt chuỗi, dài nhất); tuần Thứ Hai → Chủ Nhật; so sánh hôm qua / TB 7 ngày (chỉ ngày đã chơi) / kỷ lục; nhận `day` hôm nay và hôm qua, từ chối ngày khác; lưu lần hai → `ErrAlreadyDone`; lưu xong gọi `MarkStudied`
- [ ] T076 [P] [US6] Test handler: `GET /api/games/daily` 200 (chưa chơi / đã chơi có `result`); `POST /api/games/daily/result` 200, 400 (`answers` ≠ 10, `kind` lạ, `day` sai, điểm vượt trần), 409 `already_done`; `POST /api/games/daily/practice` 200
- [ ] T077 [P] [US6] Test `frontend/src/app/features/games/daily-challenge/daily.logic.spec.ts` và `share.logic.spec.ts`: 20 giây/câu, hết giờ → sai; điểm cộng theo giây còn lại; câu động từ `base` rỗng → chọn động từ thông dụng bằng `seed` (cùng seed → cùng động từ); Ghép chữ xáo trộn có chữ nhiễu; chuỗi chia sẻ `Luna 13/11 · 9/10 🔥13` + 10 ô 🟩/🟥, không có đáp án

### Implementation

- [ ] T078 [P] [US6] Repository `daily_challenges`: `DailyChallenges() games.DailyRepository` trong `backend/internal/storage/storage.go`; Mongo (chỉ mục duy nhất `(userId, day)`, `Save` gặp trùng khoá → `games.ErrExists`), MySQL (bảng từ migration 045, `answers` lưu JSON); test MySQL
- [ ] T079 [US6] Đưa `daily_challenges` vào file xuất (`CollDailyChallenges`, `DailyChallenges []Doc \`json:"dailyChallenges"\``) và danh sách xoá tài khoản như T014
- [ ] T080 [US6] Viết `backend/internal/games/daily.go`: `Daily(ctx, userID, isGuest)` (ngày theo `Timezones`, `seed = fnv(userID+day)`, trộn câu từ `PictureRound`/`LettersRound`/`GrammarRound` và động từ đã học qua `VerbProgress.LearnedBases`, `streakOf`, tuần, so sánh), `SaveDaily(ctx, userID, body)` (kiểm tra, `ErrAlreadyDone`, lưu, `MarkStudied`, đưa mục sai về hàng ôn), `DailyPractice`; adapter `VerbProgress` trong `backend/internal/service/games.go`
- [ ] T081 [US6] Route `GET /api/games/daily`, `POST /api/games/daily/result`, `POST /api/games/daily/practice`; điền `daily{day, done, streak}` trong `List` (`backend/internal/games/list.go`)
- [ ] T082 [P] [US6] Viết `frontend/src/app/features/games/daily-challenge/daily.logic.ts` và `share.logic.ts`
- [ ] T083 [US6] Viết component `frontend/src/app/features/games/daily-challenge/daily-challenge.ts|html|css` theo gif đợt 2 phần 2: màn mở đầu (ngày, 🔥 chuỗi, hàng 7 ngày trong tuần, "Bắt đầu"); màn chơi (hàng 10 chấm tiến độ xanh/đỏ, mỗi câu một dạng rút gọn: ảnh 4 đáp án, gõ V2+V3, Ghép chữ xáo trộn không rơi, Bắn lỗi 1 câu; thanh 20 giây); màn kết quả (chuỗi tăng, 10 ô, so với hôm qua / TB 7 ngày / kỷ lục, "Câu sai đã thêm vào lịch ôn", "↗ Chia sẻ kết quả" dùng `navigator.clipboard.writeText` và báo "Đã sao chép", không có clipboard thì hiện ô chữ để chép); đã xong hôm nay → hiện kết quả đã tính + nút "Chơi lại để luyện" (gửi `/daily/practice`)
- [ ] T084 [US6] Thẻ Thử thách đầu trang `/games` (`frontend/src/app/features/games/games.html`) và thẻ Thử thách trên trang chủ `frontend/src/app/features/home/home.ts|html` (chưa chơi → "Bắt đầu", đã xong → "Đã xong" + 🔥 chuỗi), 1 chạm vào `/games/daily`; sửa `home.spec.ts`
- [ ] T085 [US6] Thêm `daily` vào `game-host.ts` (hoặc route riêng `/games/daily`) và vào `isFocusPage`; kiểm tra thủ công quickstart.md bước 10–11

---

## Phase 9: Polish & Cross-Cutting

- [ ] T086 [P] Cập nhật `docs/architecture.md` (thêm `games/` vào cây thư mục backend và `features/games` frontend) và đánh dấu tiêu chí F26 trong `docs/phases/giai-doan-3.md` khi đạt
- [ ] T087 [P] Rà khả năng tiếp cận các màn chơi: mọi nút có nhãn đọc được, đúng/sai có ✓/✗ và chữ ngoài màu, có `aria-live="polite"` báo đúng/sai và thời gian còn lại (không đọc mỗi giây), tương phản WCAG AA ở cả sáng và tối, chơi được bằng bàn phím
- [ ] T088 Chạy `go test ./...`, `go vet ./...` trong `backend/`, `npm test` và `npm run lint` trong `frontend/`; sửa hết lỗi
- [ ] T089 Kiểm tra thủ công toàn bộ `specs/021-learning-games/quickstart.md` (bước 1–12) ở 360 px, sáng và tối; kiểm tra bước 12 (xuất dữ liệu, xoá tài khoản)

---

## Dependencies & Execution Order

- **Setup (T001–T002)** → **Foundational (T003–T026)** → các user story.
- **US1 (P1)** chỉ cần Foundational. Là MVP.
- **US2** cần Foundational (T007 nhận `wrongVerbs`); độc lập với US1 về code nhưng dùng chung khung.
- **US3, US4** chỉ cần Foundational (dùng `pool.go`).
- **US5** cần Foundational; phần admin (T066–T070) độc lập với màn chơi (T071–T073).
- **US6** cần các trò mà nó trộn: dựng câu từ US1 (`PictureRound`), US3 (`LettersRound`), US5 (`GrammarRound`) và logic chấm động từ của US2. Làm sau cùng, như spec.
- Trong mỗi story: test → backend → frontend logic → component → kiểm tra thủ công.

## Parallel Opportunities

- Phase 1: T001 ∥ T002.
- Phase 2: T005/T006 (vocab) ∥ T012/T013 (storage) ∥ T016/T017 (pool) ∥ T018–T020 (frontend thuần); T021–T026 sau T018.
- US1: T027 ∥ T028 ∥ T029, rồi T030 ∥ T032.
- US5: T066 ∥ T067 ∥ T063; T071 ∥ T068.
- US6: T078 ∥ T082.
- Sau Foundational, US1–US5 có thể giao cho nhiều người/agent cùng lúc (khác thư mục); US6 chờ.

### Ví dụ chạy song song (US1)

```text
T027 backend/internal/games/picture_test.go
T028 backend/internal/games/handler_test.go (picture)
T029 frontend/src/app/features/games/picture-quiz/picture-quiz.logic.spec.ts
```

## Implementation Strategy

1. **MVP**: Phase 1 + Phase 2 + US1 → kiểm tra quickstart 1–4 → dừng lại cho người dùng thử.
2. Thêm lần lượt US2 → US3 → US4 → US5 → US6; mỗi story xong thì thêm vào `released` và chạy test + lint.
3. Phase 9 sau cùng.
