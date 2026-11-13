# Research: Trò chơi — vừa chơi vừa học (F26)

Mọi quyết định dưới đây dựa trên code hiện có (đọc ngày 2026-11-13). Không còn mục NEEDS CLARIFICATION.

## R1. Đưa mục chơi sai về hàng ôn

- **Decision**: thêm `vocab.Service.GameMisses(ctx, userID, lemmas []string) (int, error)`: với mỗi lemma đã có thẻ,
  nếu hạn ôn còn ở sau `now` thì đặt `Due = now` (giữ nguyên trạng thái FSRS), giống nhánh "thẻ đã có" của
  `PracticeMisses` (`backend/internal/vocab/misses.go`). Lemma chưa có thẻ thì bỏ qua (không tạo thẻ). Tối đa 60 lemma
  mỗi lần, mỗi lemma xử lý một lần.
- **Động từ**: thêm cờ `Game bool` vào `verbs.Result`. Khi `Game` và sai: đếm `Wrong++`, giữ lỗi gõ (`recordTyping`),
  và nếu động từ đã học mà `DueAt` sau `now` thì đặt `DueAt = now` (không đổi hộp). Khi `Game` và đúng: chỉ `Right++`
  như `applyPractice`. Trò gửi chỉ các động từ đã sai, nên lịch ôn không bị đẩy xa (FR-004, US2 #6).
- **Rationale**: dùng lại đúng cơ chế "kéo hạn ôn về hôm nay mà không đổi độ thuộc" đã được chấp nhận ở luyện tập bài
  học và ở chế độ `practice` của động từ.
- **Alternatives**: chấm thẻ "Quên" (rating 1) → làm hỏng FSRS vì trò chơi không phải lượt ôn thật; tạo thẻ cho từ chưa
  có → spec không cho (Assumptions).

## R2. Nguồn từ cho các trò từ vựng (Ảnh nhanh tay, Chữ rơi, Lật thẻ, câu từ vựng của thử thách)

- **Decision**: "từ đã học" = thẻ ôn của người học (`vocab.Repository.Words`, `Schedules`). Thứ tự ưu tiên: thẻ đến hạn →
  thẻ khác. Ảnh lấy bằng `wordbank.Service.ImageURLs(lemmas)` (ảnh Kho, URL `/api/words/{lemma}/image`), nghĩa lấy từ
  thẻ (`MeaningVi`), thiếu thì từ Kho (`wordbank.Service.Words`).
- **Thiếu từ** (người mới, khách mới): dùng từ vựng của các bài đã học (`progress.StudyService.StudiedLessons` +
  `vocab.LessonVocabulary.Vocabulary(lessonID)`). Nếu vẫn chưa có bài nào thì dùng bài đầu tiên của chủ đề đang học
  (`StudyService.Goals` → bài đầu của lộ trình hiện tại). Khách chỉ có bài khách được mở nên tự đúng quyền (FR-033).
- **Đáp án nhiễu (Ảnh nhanh tay)**: lấy từ các lemma khác trong cùng bể từ (ưu tiên cùng bài/`LessonID`), loại từ trùng
  nghĩa (`MeaningVi` giống nhau sau khi chuẩn hoá) và dạng khác của nhau (cùng lemma, hoặc một từ là từ kia thêm
  s/es/ed/ing). Bể có dưới 4 lemma thì trò báo "Học thêm vài bài để mở trò này" (`409 not_enough_words`).
- **Chữ rơi / Ghép chữ**: chỉ lemma một từ, toàn chữ cái a–z, dài 3–8.
- **Rationale**: không cần thêm truy vấn mới vào Kho; mọi thứ đều đã có interface sẵn.

## R3. Dựng lượt chơi ở đâu

- **Decision**: server dựng lượt cho các trò cần dữ liệu người học (ảnh, chữ, lật thẻ, ngữ pháp, thử thách) qua
  `GET /api/games/{game}/round`; frontend chấm điểm, đếm giờ và gửi kết quả. **Đấu động từ** dựng hoàn toàn ở frontend
  vì danh sách 237 động từ chỉ nằm ở `frontend/src/app/features/irregular-verbs/irregular-verbs.data.ts`; frontend đã có
  `GET /api/verbs/progress` để xếp thứ tự (đến hạn, hay sai, đã học, thông dụng — dùng lại `verb-mistakes.logic.ts`).
- **Rationale**: không có bảng xếp hạng nên không cần chống gian lận chặt; điểm do client tính, server chỉ chặn giá trị
  vô lý (giới hạn trên mỗi trò, xem data-model).
- **Alternatives**: chuyển danh sách động từ sang backend → việc lớn, không cần cho spec này.

## R4. Bắn lỗi ngữ pháp: dựng câu từ dữ liệu có sẵn

Bài ngữ pháp lưu `Content` gồm `Practice`, `Mastery` (các `Exercise`) và `Mistakes`; ngoài ra có danh sách lỗi người
Việt hay gặp nhúng sẵn (`grammar.MistakesOf(pointID)`, `l1_mistakes.json`). Chỉ dùng bài **đã đăng**.

- **(a) `choice`**: `Text` có đúng một `___`, `Options` 3–4 phương án, `AnswerIndex`. Câu có lỗi = thay `___` bằng một
  phương án sai (chọn ngẫu nhiên); chỗ sai = vị trí phương án đó (có thể nhiều chữ). 3 phương án sửa = đáp án đúng + 2
  phương án sai khác (khi chỉ có 3 phương án thì là cả 3, trong đó có chữ sai đang hiện). Câu "Không có lỗi" = thay
  `___` bằng đáp án. Bỏ bài tập có `Text` không có đúng một `___`.
- **(b) `fix` và `Mistakes`/`MistakesOf`**: tách câu sai và câu đúng thành từ (giữ dấu câu riêng), tìm đoạn khác nhau
  ngắn nhất ở giữa (bỏ phần đầu và phần cuối giống nhau). Chỉ nhận khi đoạn khác ở mỗi bên dài 1–3 từ. Phương án sửa: chữ
  đúng, chữ sai, và một chữ nhiễu thứ ba lấy từ chỗ sai của một câu khác cùng điểm ngữ pháp (khác cả chữ đúng và chữ
  sai); không có thì bỏ câu.
- **(c) câu soạn thêm** (R5): đã có sẵn chỗ sai, chữ đúng và 2 chữ nhiễu.
- **Quy tắc** hiện sau mỗi câu: `ExplanationVi` của bài tập, `NoteVi` của lỗi, hoặc `RuleVi` của câu soạn thêm. **Nhãn**
  = `TitleVi` của điểm ngữ pháp (syllabus).
- **Chọn câu cho lượt**: điểm người học đã có tiến độ (`grammar.ProgressRepository.List`) trước, rồi điểm khác cùng trình
  độ, rồi trình độ thấp hơn; khoảng 3/10 câu "Không có lỗi".
- **Rationale**: lựa chọn C của người dùng (clarify 2026-11-13); câu và chữ nhiễu ở (a) đã được admin duyệt nên không
  thể ra câu đúng khác.

## R5. Admin nhờ AI soạn câu Bắn lỗi

- **Decision**: thêm vào `ai.Provider` phương thức `GrammarSpotItems(ctx, GrammarSpotRequest) ([]GrammarSpotItem,
  error)`, có bản `Disabled` trả `ErrNotConfigured`, và bản `llm` (`backend/internal/ai/llm/spot.go`) gọi `c.generate`
  với op **`grammar_spot`**, schema JSON, nhiệt độ thấp. Prompt viết bằng **tiếng Anh** (quy ước dự án), gửi kèm
  `TitleEn`, `Pattern`, trình độ, ví dụ của điểm và các câu đã có để tránh trùng. Một lần bấm = 1 request, tối đa 10
  câu.
- **Kiểm tra sau khi nhận** (FR-026b): chỗ sai phải nằm trong câu sai đúng một lần; chữ đúng ≠ chữ sai; 2 chữ nhiễu khác
  nhau và khác chữ đúng; câu đúng (thay chỗ sai bằng chữ đúng) không trùng câu đã có; độ dài câu 3–20 từ. Câu không đạt bị
  bỏ, trả về số câu bị bỏ.
- **Chạy đồng bộ** trong request của admin, có timeout, giống `grammar.Service.Generate`. Lỗi AI không ảnh hưởng trò
  chơi (trò vẫn dùng nguồn a, b).
- **Ghi lượt dùng**: `c.generate` đã ghi `ai_usage`; thêm op `grammar_spot` vào
  `frontend/src/app/features/admin/ai-usage/ai-usage-catalog.ts`.
- **Alternatives**: đưa vào hàng đợi job → hàng đợi dùng chung giữa máy local và production (xem memory "Dev và
  production chung 1 DB"), và các thao tác AI một request khác của admin (sinh bài ngữ pháp, điền nghĩa) đều chạy đồng
  bộ. Ghi ở Complexity Tracking.

## R6. Thử thách hằng ngày

- **Ngày**: theo múi giờ của người học (`Timezones.Location`, mặc định `Asia/Ho_Chi_Minh`), khoá `YYYY-MM-DD`.
- **Bộ 10 câu cố định trong ngày**: dựng bằng số ngẫu nhiên gieo từ `hash(userID + day)`, nên mở lại trong ngày luôn ra
  cùng bộ mà không phải lưu bộ câu. Tỉ lệ mặc định: 3 ảnh, 3 động từ, 2 ghép chữ, 2 bắn lỗi. Khách: 4 ảnh, 3 động từ, 3
  ghép chữ. Trò nào thiếu nội dung thì phần thiếu chia cho trò khác. Câu động từ: server chọn `base` từ động từ đã học
  (đến hạn trước, đọc `verbs.Repository.List`); người học chưa học động từ nào thì trả `base` rỗng và frontend chọn một
  động từ thông dụng bằng cùng hạt giống `seed`.
- **Chỉ lượt đầu được tính**: `POST /api/games/daily/result` lưu khi chưa có kết quả của ngày đó; có rồi thì trả
  `409 already_done` kèm kết quả cũ. Nhận `day` là hôm nay hoặc hôm qua (lượt bắt đầu trước nửa đêm, Edge Cases).
- **Chuỗi**: tính khi đọc từ danh sách ngày đã có kết quả (một bản ghi mỗi ngày, rất ít) — không lưu thực thể chuỗi
  riêng. Hiện tại = số ngày liên tiếp kết thúc ở hôm nay (hoặc hôm qua nếu hôm nay chưa chơi); dài nhất = đoạn liên tiếp
  dài nhất.
- **So sánh** (FR-031): điểm hôm qua, trung bình các ngày đã chơi trong 7 ngày trước hôm nay, kỷ lục thử thách.
- **Ngày học**: lưu kết quả xong thì gọi `StudyDays.MarkStudied` (như vocab, grammar, verbs).
- **Rationale**: tính chuỗi khi đọc giữ một nguồn sự thật, không lệch khi xoá hay sửa dữ liệu.

## R7. Giao diện và chuyển động

- **Không thêm thư viện**. Chữ rơi dùng CSS `@keyframes` cho từng chữ (`translateY`, thời lượng theo màn), bắt
  `animationend` để biết chữ rơi khỏi màn; chạm là sự kiện click thường. `prefers-reduced-motion`: dùng
  `animation-timing-function: steps(n)` để chữ đi từng bước. Lật thẻ dùng `transform: rotateY` (giảm chuyển động thì chỉ
  đổi mặt).
- **Đồng hồ**: một hàm thuần đếm thời gian còn lại từ `performance.now()` (không đếm bằng `setInterval` cộng dồn), tạm
  dừng khi `document.visibilitychange` báo trang ẩn (FR-007).
- **Gửi kết quả khi mất mạng**: kết quả chưa gửi được lưu `localStorage` (khoá `luna.games.pending`), gửi lại khi mở
  trang Trò chơi; bọc try/catch như các chỗ khác dùng localStorage.
- **Toàn màn hình**: thêm `/games/:game` vào `isFocusPage` và `/games` vào `isPracticePath` trong
  `frontend/src/app/layouts/learner-layout/learner-layout.ts`; thêm ô "Trò chơi" vào danh sách ô của
  `practice-menu.ts`.
- **Chia sẻ**: `navigator.clipboard.writeText`, không có thì hiện ô chữ để người học tự chép.

## R8. Lưu trữ

- Hai collection/bảng mới cho dữ liệu người học: `game_bests`, `daily_challenges`; một cho nội dung admin:
  `grammar_spot_items`. Làm cả Mongo (`internal/storage/mongo`) và MySQL (`internal/storage/mysql`, migration
  **045_games**), khai báo trong `storage.Store`.
- `game_bests` và `daily_challenges` thuộc dữ liệu người học: thêm vào file xuất dữ liệu (`internal/export`) và danh sách
  xoá theo tài khoản (`storage/{mongo,mysql}/users.go`).
