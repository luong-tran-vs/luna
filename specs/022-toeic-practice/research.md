# Research: Luyện thi TOEIC Nghe–Đọc (F27)

Mọi quyết định dựa trên code hiện có (đọc ngày 2026-11-18) và spec đã clarify. Không còn mục NEEDS CLARIFICATION.

## R1. Đơn vị nội dung và mã ổn định

- **Decision**: đơn vị lưu là **cụm câu** (`toeic_groups`). Mỗi cụm có `key` do người soạn đặt trong JSON (ví dụ
  `p7-double-12`, chữ thường, số, gạch ngang, ≤ 48 ký tự), duy nhất toàn hệ thống. Mã câu hỏi = `key#n` (n = thứ tự câu
  trong cụm, bắt đầu từ 1), ổn định khi nhập lại. Đề, bài làm, sổ câu sai đều trỏ tới `key` / mã câu, không trỏ tới ObjectID.
- **Rationale**: FR-040a yêu cầu nhập lại giữ mã cụm; dùng `key` làm khoá nghiệp vụ thì JSON trong `content/toeic/` là
  nguồn duy nhất, nhập lại không cần tra ID nội bộ. Câu nằm trong cụm (cụm Part 3 luôn có 3 câu) nên lưu lồng trong cụm.
- **Alternatives**: mỗi câu một bản ghi riêng → phải ghép lại cho Part 3/4/6/7 mỗi lần đọc, nhập lại phức tạp hơn.

## R2. Không lộ đáp án trước khi nộp

- **Decision**: API người học trả "đề để làm" (`PaperView`): ngữ liệu và câu hỏi **không có** `answer`, `evidence`,
  `explainVi`, `translateVi`, `vocab`. Phần Nghe cần chữ để giọng trình duyệt đọc, nên `script` và các phương án nói của Part
  1/2 được gửi kèm nhưng **không hiện** trên giao diện cho đến khi nộp. Đáp án và giải thích chỉ có trong `GET .../review`
  sau khi bài ở trạng thái đã nộp.
- **Rationale**: SC-004 nói về đáp án/giải thích; lời đọc không phải khoá đáp án (người nghe vốn nghe được lời đó). Không có
  bảng xếp hạng nên không cần mã hoá lời đọc.
- **Alternatives**: tạo file âm thanh trên server để giấu chữ → trái quyết định "nghe bằng giọng trình duyệt, không lưu
  file" và nguyên tắc chi phí.

## R3. Giọng đọc theo kịch bản (phần Nghe)

- **Decision**: thêm vào frontend một **trình phát kịch bản** thuần (`listening-script.logic.ts`) dựng danh sách đoạn
  `{text, accent, speaker, pauseAfterMs, groupIndex, questionNo?}` từ đề: Directions (lời Luna tự viết, không chép lời ETS)
  → câu dẫn "Questions 41 through 43 refer to the following conversation." → các lượt nói → đọc từng câu hỏi (Part 3/4) với
  nghỉ 8 giây; Part 1: "Number 3. Look at the picture marked number 3 in your test book." + (A)…(D) + nghỉ 5 giây; Part 2:
  câu hỏi + (A)…(C) + nghỉ 5 giây. Component `listening-player` chạy lần lượt các đoạn qua `SpeechService`.
- **Mở rộng `SpeechService`** (chỉ thêm, không đổi hành vi cũ):
  - tham số `opts?: { lang?: 'en-US'|'en-GB'|'en-AU'|'en-CA'; natural?: boolean }`: chọn giọng có `lang` khớp giọng yêu cầu
    (hết thì quay về giọng tiếng Anh xếp hạng cao), `natural: false` để bỏ `NaturalVoiceService` (chỉ có 1 giọng) khi cần
    nhiều người nói;
  - hàm `voiceCount()` để màn kiểm tra trước khi thi báo "máy chỉ có 1 giọng tiếng Anh".
- **Tạm dừng**: Web Speech `pause()` không ổn định trên Android → tạm dừng = `stop()` và ghi nhớ đoạn hiện tại; tiếp tục =
  đọc lại **từ đầu đoạn đó**. Nghe lại cụm = nhảy về đoạn đầu của cụm. Tốc độ = `rate` 0.75/1/1.25.
- **Độ dài phần Nghe** (SC-006): ước tính từ số từ (150 từ/phút ở rate 1) + các khoảng nghỉ. Chạy thử khi nhập đề báo tổng
  thời lượng ước tính của phần Nghe; ngoài 40–50 phút thì báo cảnh báo (không chặn).
- **Chặn tự phát**: màn kiểm tra trước khi thi và màn Directions đầu tiên có nút bấm để "mở khoá" âm thanh.

## R4. Bài làm dở, đồng hồ và làm tiếp trên máy khác

- **Decision**: bài làm (`toeic_attempts`) lưu trên server: `answers`, `flags`, `position`, `remainingMs`, `phase`,
  `listeningSegment`. Frontend gửi `PUT /api/toeic/attempts/{id}` khi chọn đáp án (gộp trong 1 giây), mỗi 5 giây khi đồng
  hồ chạy, và khi trang bị ẩn (`visibilitychange`, `fetch` với `keepalive`). Đồng hồ chạy ở client bằng `game-clock.logic.ts`
  (dùng lại); server nhận `remainingMs` mới nhưng **không cho tăng** so với giá trị đã lưu.
- **Rời phòng thi**: thời gian dừng (spec Assumptions). Mở lại lấy `remainingMs` đã lưu → sai lệch ≤ 5 giây (SC-003).
- **Phần Nghe của full test**: lưu `listeningSegment` = đoạn đầu của cụm đang phát; mở lại thì phát từ đó (FR-020). Thời
  gian phần Nghe đi theo kịch bản, không có đồng hồ riêng; đồng hồ 75 phút chỉ bắt đầu khi sang phần Đọc.
- **Mất mạng**: các `PUT` thất bại được giữ trong hàng đợi trên máy (localStorage, giống hàng đợi kết quả của Trò chơi) và
  gửi lại; nộp bài khi mất mạng hiện "Chưa gửi được, thử lại" và không xoá bài.
- **Hết giờ**: client tự gọi submit; nếu client không gọi (đóng máy), lần mở sau thấy `remainingMs = 0` thì nộp luôn.
- **Alternatives**: đồng hồ tính ở server theo `startedAt` → không làm được "dừng khi rời phòng thi"; lưu chỉ ở
  localStorage → không làm tiếp được trên máy khác.

## R5. Quy đổi điểm ước tính

- **Decision**: hàm thuần `scaled(section, correct, total)`: quy về 100 câu (`c100 = round(correct * 100 / total)`), rồi nội
  suy tuyến tính giữa các mốc và làm tròn bước 5, kẹp trong 5–495:
  - Listening: 0→5, 10→20, 25→105, 50→245, 65→330, 75→385, 85→435, 92→470, 96→495, 100→495.
  - Reading: 0→5, 10→15, 25→85, 50→215, 65→300, 75→360, 85→420, 92→460, 97→495, 100→495.
  Có điểm thang 990 chỉ khi lượt có **đủ 7 Part** (full test, Mini Test, đề ngẫu nhiên). Luyện tập vài Part chỉ có số câu đúng
  và độ chính xác (FR-025).
- **Rationale**: ETS không công bố bảng cố định; các mốc trên bám theo các bảng ước tính phổ biến (Reading thường khó lấy
  điểm hơn Listening ở cùng số câu đúng). Luôn ghi "ước tính".
- **Alternatives**: chia đều 495/100 → lệch nhiều ở hai đầu.

## R6. Dạng câu (tag) và liên kết ngữ pháp

- **Decision**: danh sách tag cố định trong code (`internal/toeic/tags.go`), mỗi tag có `Code`, `LabelVi`, `Parts`, và tuỳ
  chọn `GrammarPoints []string` (mã điểm trong `grammar/syllabus.json`, ví dụ tag `verb-tense` → các điểm thì). Chạy thử từ
  chối tag lạ hoặc tag không thuộc Part của câu. Một test kiểm tra mọi `GrammarPoints` có trong syllabus.
- Danh sách khởi đầu:
  - P1: `photo-people`, `photo-objects`, `photo-scene`.
  - P2: `q-wh`, `q-yesno`, `q-choice`, `q-request`, `q-statement`, `indirect-answer`.
  - P3/4: `main-idea`, `detail`, `inference`, `speaker-intent`, `next-action`, `graphic`, `who-where`.
  - P5/6: `word-form`, `verb-tense`, `verb-form`, `preposition`, `conjunction`, `pronoun`, `relative`, `comparison`,
    `vocabulary`, `collocation`, `sentence-insert` (chỉ P6).
  - P7: `main-idea`, `detail`, `inference`, `not-true`, `vocabulary-in-context`, `sentence-insert`, `cross-reference`,
    `speaker-intent`.

## R7. Ảnh trong đề

- **Ảnh Part 1** lưu ở collection mới `toeic_images` (khoá `(groupKey, slot)`, `slot = "photo"`; ảnh minh hoạ văn bản Part 7
  dùng `slot = "doc-<n>"`), JPEG thu nhỏ bằng `imageutil.Shrink(data, 800)`, chất lượng 70 như ảnh cảnh của bài. Không dùng
  chung `word_images` (khoá theo bài) hay Kho từ (khoá theo lemma).
- **Vẽ bằng AI**: thêm `ai.PhotoBrief{Scene, Statement string}` vào `ai.ImageRequest` (trường mới `Photo *PhotoBrief`); prompt
  tiếng Anh trong `ai/llm/image.go` và `ai/codexcli`: "realistic photograph, natural light, workplace or everyday scene, no
  text, no captions, no watermark"; tỉ lệ 4:3 (`ImageAspect` mặc định). Prompt chỉ gồm `scene` và câu mô tả đúng (FR-037).
- **Engine bắt buộc**: route TOEIC trả `400 validation_failed {engine}` khi thiếu `engine` (chặt hơn lesson/wordbank vốn mặc
  định Gemini), đúng quy tắc người dùng "phải chọn mỗi lần". Frontend dùng `lu-engine-dialog` với `requireChoice`.
- **Ghi sử dụng AI**: op mới `toeic_photo` (thêm vào `OP_LABELS` và bảng chi phí trong `ai-usage-catalog.ts`).
- **Vẽ hàng loạt** "Vẽ ảnh còn thiếu" của một đề chạy ngầm trong process như `wordbank/batch.go` (tối đa 1 lượt mỗi lúc), không
  dùng hàng đợi job (dev và production chung DB).
- **Biểu đồ Part 3/4 và văn bản Part 6/7** dựng bằng component từ dữ liệu JSON (không ảnh).

## R8. Nhập lại cụm đã đăng và chấm lại (FR-040a)

- **Decision**: nhập với `replace: true` → cụm có `key` trùng được thay nội dung, giữ `key`, `status`, `createdAt`, tăng
  `version`. Nếu cụm nằm trong một đề đã đăng thì số câu phải giữ nguyên (chạy thử báo lỗi `questions`).
- **Chấm lại**: sau khi lưu, service tìm các bài đã nộp có `groupKeys` chứa cụm đó (chỉ số trên `groupKeys`), chấm lại toàn
  bài (hàm thuần `grade(paper, answers)`), cập nhật `result`. Sổ câu sai: với mỗi người học và mỗi câu của cụm, lấy câu trả lời
  ở **bài nộp gần nhất** có câu đó: đúng theo khoá mới → xoá mục sổ (nếu có); sai mà chưa có mục → thêm mục `streak = 0`.
- Chạy đồng bộ trong request nhập (vài người học, vài chục bài) — đủ nhanh; nếu lỗi giữa chừng thì báo và có thể nhập lại.

## R9. Dựng đề cho lượt làm

- **Đề cố định** (full / Mini Test): `paper` = danh sách `groupKey` của đề theo Part; lượt luyện tập chỉ lấy các Part đã tick.
- **Đề ngẫu nhiên, luyện theo Part, theo tag, làm lại câu sai**: server dựng `paper` khi tạo bài làm và **lưu vào bài làm**
  (mở lại vẫn đúng đề). "Câu đã làm" = mã câu có trong các bài đã nộp của người học (đọc từ `toeic_attempts`, chỉ trường
  `questionIds`); ưu tiên cụm có ít câu đã làm nhất, rồi ngẫu nhiên.
- **Tỉ lệ đề ngẫu nhiên** (theo cụm, làm tròn theo câu): 50 câu = P1 2, P2 6, P3 9 (3 cụm), P4 9 (3 cụm), P5 8, P6 4 (1 văn
  bản), P7 12; 100 câu = P1 3, P2 12, P3 21, P4 15, P5 15, P6 8, P7 26 (gần đúng theo cụm). Mini Test cố định do admin ghép
  theo cùng tỉ lệ (kiểm tra ±1 cụm).

## R10. Quyền khách (FR-045)

- **Decision**: đề được khách làm = Mini Test **đã đăng có `order` nhỏ nhất**. Luyện theo Part chỉ Part 1, 2, 5. Kiểm tra ở
  service (`isGuest`) như `games.List`, và route dùng `auth` (không `member`) cho các API dùng chung, trả `403 members_only`
  khi khách vượt quyền; các route chỉ thành viên (sổ câu sai, theo tag, thống kê, đề ngẫu nhiên) dùng `httpx.RequireMember`.
- Thư viện đề vẫn liệt kê mọi đề cho khách với cờ `locked`; xem đề bị khoá chỉ có tên, không có nội dung.

## R11. Bố cục phòng thi

- **Decision**: một component `exam-room` cho mọi chế độ; dưới 1024 px xếp dọc (thanh dưới, phiếu trả lời), từ 1024 px trở
  lên chia cột (văn bản | câu hỏi | cột bên). Chỉ dùng token của `docs/design-system.md`. Tô vàng văn bản lưu theo bài làm
  trong localStorage (`toeic-highlights:<attemptId>`), không lên server (FR-009 chỉ cần giữ suốt bài làm).

## R12. Nội dung do Claude soạn

- **Decision**: file JSON trong `content/toeic/<part>/…` và `content/toeic/tests/<test>.json` (danh sách `groupKey` của đề).
  Định dạng ở `contracts/content-format.md`. Nhập qua `POST /api/admin/toeic/import?dryRun=true` như các luồng nhập khác.
  Directions là lời Luna tự viết bằng tiếng Anh (cố định trong frontend), không chép lời trong đề ETS.

## R13. Chuỗi ngày học

- **Decision**: khi nộp bất kỳ lượt nào (FR-035) gọi `progress.StudyService.MarkStudied` qua adapter `studyDays` có sẵn
  (`internal/service/progress.go`), như `games`.
