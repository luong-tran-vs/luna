# Implementation Plan: Trò chơi — vừa chơi vừa học (F26)

**Branch**: `021-learning-games` (làm trên `main`) | **Date**: 2026-11-13 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/021-learning-games/spec.md`

## Summary

Thêm mục **Trò chơi** cho người học (vào từ nút giữa Luyện) gồm 6 trò làm theo thứ tự P1 → P6: Ảnh nhanh tay, Đấu động
từ, Chữ rơi, Lật thẻ, Bắn lỗi ngữ pháp, Thử thách hằng ngày. Backend có package mới `internal/games`: dựng lượt chơi từ dữ
liệu đã có (thẻ ôn, ảnh Kho từ, từ vựng bài học, bài tập ngữ pháp đã đăng), nhận kết quả (kỷ lục, đưa mục sai về hàng ôn
bằng cơ chế "kéo hạn về hôm nay" có sẵn), lưu kết quả thử thách mỗi ngày và tính chuỗi khi đọc. Bắn lỗi dựng câu từ bài
tập `choice` (điền phương án sai), `fix` và danh sách lỗi hay gặp, thêm câu do AI soạn khi admin bấm (op `grammar_spot`,
admin duyệt). Frontend có feature `games` với phần dùng chung (đồng hồ, tạm dừng, điểm, màn kết quả, gửi lại khi mất mạng)
và một component cho mỗi trò; Đấu động từ dựng lượt ở frontend vì danh sách động từ chỉ nằm ở đó.

## Technical Context

**Language/Version**: Go 1.25 (backend), TypeScript strict + Angular 21 standalone/signals (frontend)

**Primary Dependencies**: thư viện chuẩn Go + driver Mongo/MySQL đang dùng; Angular. **Không thêm thư viện mới.**

**Storage**: MongoDB và MySQL (cả hai đều được hỗ trợ, chọn bằng cấu hình, `internal/storage/factory`). Mới:
`game_bests`, `daily_challenges`, `grammar_spot_items` (MySQL migration `045_games`)

**Testing**: `go test` (fake repository như các package khác, MySQL test qua `testdb_test.go`); Vitest qua `ng test`;
`ng lint`

**Target Platform**: web responsive, chủ yếu điện thoại (từ 360 px), sáng/tối

**Project Type**: web application (backend Go REST + frontend Angular)

**Performance Goals**: phản hồi đúng/sai ngay trên máy (không chờ mạng); dựng một lượt < 300 ms với vài nghìn thẻ; chuyển
động 60 fps trên điện thoại tầm trung (chỉ animate `transform`/`opacity`)

**Constraints**: không gọi AI khi người học chơi; dev và production dùng chung một DB (đổi schema phải tương thích ngược,
chỉ thêm); prompt AI viết bằng tiếng Anh; chữ trên giao diện bằng tiếng Việt

**Scale/Scope**: vài người dùng; mỗi người một bản ghi kỷ lục/trò và một bản ghi/ngày cho thử thách; 6 màn chơi + 1 trang
danh sách + 1 thẻ trang chủ + 1 phần trong trang quản trị ngữ pháp

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- [x] **I. docs/ là nguồn sự thật**: đã thêm **F26** vào `docs/phases/giai-doan-3.md` (chức năng + tiêu chí nghiệm thu)
  trước khi lập kế hoạch; spec ghi mã F26; mỗi tiêu chí có bước kiểm tra ở `quickstart.md`.
- [x] **II. Chi phí 0 đồng**: không thêm dịch vụ hay thư viện. Soạn câu bằng AI dùng nhà cung cấp AI đang cấu hình, chỉ
  chạy khi admin bấm.
- [x] **III. AI là phần bổ sung**: mọi trò chạy không cần AI; AI chỉ ở phần admin, qua `ai.Provider`
  (`GrammarSpotItems`, có `Disabled`). *Ngoại lệ*: chạy đồng bộ trong request thay vì job nền — xem Complexity Tracking.
- [x] **IV. Mobile-first**: thiết kế cho 360 px, một tay (vùng chạm nửa dưới, ≥ 44 px), chỉ dùng token màu/chữ của
  `docs/design-system.md`, đúng/sai có ✓/✗ ngoài màu, tôn trọng `prefers-reduced-motion`.
- [x] **V. Dữ liệu người học**: mọi route kiểm tra quyền ở API (`requireAuth`, `RequireMember` cho Bắn lỗi,
  `RequireAdmin` cho câu soạn thêm); dữ liệu theo `userID`; truy cập DB qua repository; dữ liệu mới có trong file xuất và
  bị xoá cùng tài khoản.
- [x] **VI. Kiểm thử**: unit test cho dựng lượt, đáp án nhiễu, tách câu Bắn lỗi, kiểm tra câu AI, chuỗi ngày, chấp nhận
  ngày, giới hạn điểm, `GameMisses`, `verbs` Game; test handler (thành công, đầu vào sai, sai quyền); AI giả lập; test
  logic frontend (đồng hồ, combo, chữ rơi, lật thẻ, chấm Bắn lỗi, chuỗi chia sẻ).
- [x] **VII. Đơn giản trước**: không lưu lịch sử lượt, không lưu thực thể chuỗi (tính khi đọc), không lưu bộ câu thử thách
  (gieo ngẫu nhiên theo ngày), chuyển động bằng CSS, không thư viện game.

*Re-check sau Phase 1*: thiết kế ở `data-model.md` và `contracts/api.md` giữ nguyên các điểm trên. **Đạt.**

## Project Structure

### Documentation (this feature)

```text
specs/021-learning-games/
├── spec.md
├── plan.md              # file này
├── research.md          # R1–R8: quyết định và lý do
├── data-model.md        # thực thể, repository, quyền
├── quickstart.md        # cách chạy và các bước kiểm tra thủ công
├── contracts/api.md     # REST API + đường dẫn frontend
├── checklists/requirements.md
└── tasks.md             # tạo bởi /speckit-tasks
```

### Source Code (repository root)

```text
backend/internal/
├── games/                         # MỚI
│   ├── games.go                   # Game, MaxScore, lỗi, kiểu câu hỏi (PictureQuestion, LetterWord, MemoryPair, SpotQuestion, DailyQuestion)
│   ├── deps.go                    # interface: Cards, Pictures, LessonWords, StarterLessons, Members, Timezones,
│   │                              #   WordMisses, VerbMisses, VerbProgress, GrammarSource, StudyDays, BestRepository,
│   │                              #   DailyRepository, SpotRepository
│   ├── pool.go                    # bể từ của người học (R2), lọc từ cho ghép chữ
│   ├── picture.go / letters.go / memory.go   # dựng lượt P1, P3, P4
│   ├── spot.go                    # tách câu (a)/(b)/(c), diff đoạn khác, dựng lượt P5 (R4)
│   ├── spot_admin.go              # câu soạn thêm: đếm theo nguồn, sinh bằng AI, ValidateSpot, sửa/duyệt/xoá (R5)
│   ├── daily.go                   # gieo theo ngày, trộn câu, chuỗi, tuần, so sánh, chấp nhận ngày (R6)
│   ├── results.go                 # nhận kết quả: giới hạn điểm, kỷ lục, đưa mục sai về hàng ôn
│   ├── service.go                 # Service + NewService
│   ├── handler.go                 # route người học + admin
│   └── *_test.go, fake_test.go
├── vocab/misses.go                # + GameMisses (R1)
├── verbs/verbs.go                 # + Result.Game, nhánh trong ApplyResult (R1); handler nhận "game"
├── grammar/                       # + PublishedSources (đọc bài đã đăng cho games)
├── ai/spot.go, ai/ai.go           # + GrammarSpotRequest/Item, Provider.GrammarSpotItems, Disabled
├── ai/llm/spot.go                 # + op "grammar_spot", schema, prompt tiếng Anh
├── storage/storage.go             # + GameBests(), DailyChallenges(), GrammarSpotItems()
├── storage/mongo/games.go         # MỚI: 3 repository + index (indexes.go)
├── storage/mysql/games.go         # MỚI: 3 repository; migrate.go + 045_games
├── storage/{mongo,mysql}/users.go # xoá game_bests, daily_challenges theo tài khoản
├── export/                        # + CollGameBests, CollDailyChallenges vào file xuất
└── service/games.go, routes.go, service.go  # initGames (sau initVocab, initVerbs, initGrammar, initProgress), adapter, đăng ký route

frontend/src/app/
├── core/models/games.ts                       # MỚI: kiểu API
├── core/services/games-api.service.ts         # MỚI: gọi API + hàng đợi kết quả chưa gửi (localStorage)
├── features/games/                            # MỚI
│   ├── games.ts|html|css                      # trang /games
│   ├── shared/
│   │   ├── game-clock.logic.ts                # đồng hồ thuần (performance.now, tạm dừng)
│   │   ├── game-score.logic.ts                # combo, điểm từng trò, kỷ lục
│   │   ├── game-shell.ts                      # khung màn chơi: thanh trên (pill), nút đóng, màn Tạm dừng
│   │   ├── game-result.ts                     # màn kết quả chung (FR-002)
│   │   └── visibility-pause.ts                # tạm dừng khi trang ẩn
│   ├── picture-quiz/                          # P1
│   ├── verb-duel/ (+ verb-duel.logic.ts)      # P2, dùng irregular-verbs.data.ts, verb-mistakes.logic.ts
│   ├── letter-drop/ (+ letter-drop.logic.ts)  # P3, CSS keyframes, animationend
│   ├── memory-match/ (+ memory.logic.ts)      # P4
│   ├── grammar-spot/ (+ grammar-spot.logic.ts)# P5
│   └── daily-challenge/ (+ daily.logic.ts, share.logic.ts)  # P6, câu rút gọn + Ghép chữ không rơi
├── features/home/                             # + thẻ Thử thách hằng ngày (FR-032)
├── features/lesson/lesson-detail/             # + lối vào "Chơi Lật thẻ với từ của bài này" ở màn xong bài (FR-022)
├── features/admin/grammar-list/, grammar lesson page   # + cột/khối "Câu Bắn lỗi" (FR-026a, FR-026b)
├── features/admin/ai-usage/ai-usage-catalog.ts          # + op grammar_spot
├── layouts/learner-layout/learner-layout.ts             # /games vào isPracticePath, /games/:game vào isFocusPage
├── layouts/learner-layout/practice-menu/practice-menu.ts # + ô "Trò chơi"
└── app.routes.ts                                         # + /games, /games/:game (lazy)
```

**Structure Decision**: web application có sẵn (backend Go chia theo domain + ports, frontend Angular chia
core/shared/features theo `docs/architecture.md`). Toàn bộ logic trò chơi nằm ở một package `games` mới; các package khác
chỉ thêm đúng phần cần (một hàm, một cờ, một phương thức AI, một bộ đọc ngữ pháp). `games` phụ thuộc các package khác qua
interface nhỏ, nối bằng adapter trong `internal/service` như các domain hiện có.

## Thứ tự làm (map với user story)

1. **Nền chung + P1 Ảnh nhanh tay**: storage `game_bests`, `vocab.GameMisses`, `games` (pool, picture, results,
   handler `GET /api/games`, `/picture/round`, `/{game}/results`), frontend trang `/games`, lối vào Luyện, khung chơi, màn
   kết quả, hàng đợi gửi lại, Ảnh nhanh tay. → phát hành được riêng.
2. **P2 Đấu động từ**: `verbs.Result.Game`, nhận `wrongVerbs` ở `/results`; frontend verb-duel.
3. **P3 Chữ rơi**: `/letters/round`; frontend letter-drop.
4. **P4 Lật thẻ**: `/memory/round` (+ `lessonId`); frontend memory-match + lối vào ở màn xong bài.
5. **P5 Bắn lỗi**: `grammar.PublishedSources`, spot.go, `/grammar/round`; storage `grammar_spot_items`, AI
   `GrammarSpotItems`, route admin, phần admin trong trang ngữ pháp, catalog AI; frontend grammar-spot.
6. **P6 Thử thách**: storage `daily_challenges`, daily.go, route daily, `MarkStudied`; frontend daily-challenge (câu rút
   gọn, Ghép chữ, chia sẻ), thẻ Trang chủ.
7. Xuất dữ liệu + xoá tài khoản cho 2 collection người học (làm cùng bước tạo collection đó).

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| III: soạn câu Bắn lỗi bằng AI chạy **đồng bộ** trong request admin, không qua job nền | Một request AI ngắn, admin bấm và chờ kết quả để duyệt ngay; giống `POST /api/admin/grammar-lessons/{pointId}/generate` và "Điền nghĩa bằng AI" đang chạy đồng bộ | Hàng đợi job dùng chung giữa máy local và production (cùng một DB), production cũ có thể lấy job; thêm job cho một thao tác admin vài giây là phức tạp hơn mà không lợi gì. Kết quả vẫn được lưu (câu nháp) nên không gọi lặp |
| Constitution "mỗi tính năng một nhánh riêng" | Người dùng làm các spec 018–020 thẳng trên `main` và đang có thay đổi chưa commit | Tạo nhánh sẽ mang theo các thay đổi chưa commit không liên quan |
