# Research: Trang chi tiết bài và luyện tập từ vựng (F17)

Mọi mục NEEDS CLARIFICATION đã được giải quyết. Mỗi mục gồm: quyết định, lý do, phương án đã loại. Mục nào lệch khối 2 của
`docs/spec-inputs/f17-luyen-tap-tu-vung.md` thì ghi rõ **Lệch đầu vào**.

## R1. `ai.Provider.Practice`

**Quyết định**

- Thêm `Practice(ctx, PracticeRequest) (Practice, error)` vào `ai.Provider`. `PracticeRequest{Level, Title string; Sentences []string;
  Words []PracticeWord{Lemma, Text, MeaningVi}}`.
- `Practice{ObjectiveVi; Examples []Example{Lemma, Sentence}; Dialogue Dialogue{Speakers []string; Turns []Turn{Speaker int, Text,
  MeaningVi}}; GrammarTipVi; Translations []Translation{Vi, En string; Distractors []string}}`, JSON camelCase như khối 2.
- `ai.Disabled` trả `ErrNotConfigured`. Cập nhật cả `fakeAI` (`internal/lesson/fake_test.go`) và `fakeGrader`
  (`internal/writing/fake_test.go`).
- Gemini: 1 request, `temperature 0.5`, `practiceSchema` (OBJECT, mọi trường cấp 1 đều `required`; `speaker` là INTEGER). Không đổi
  `Annotate`.
- Prompt (tiếng Anh, raw string như `annotatePrompt`): học viên Việt Nam trình độ CEFR `{level}`, tiêu đề, các câu bài đọc có số thứ tự,
  danh sách từ `lemma | text | meaningVi`. Yêu cầu:
  - `objectiveVi`: 1 câu tiếng Việt "Bạn có thể …";
  - `examples`: mỗi lemma 1 câu ngắn (≤ 12 từ), đúng trình độ, chứa nguyên từ;
  - `dialogue`: 2 tên (một tên Việt, một tên nước ngoài), 6–10 lượt luân phiên, dùng nhiều từ trong danh sách, mỗi lượt có `meaningVi`;
  - `grammarTipVi`: 1–2 câu về một cách nói trong hội thoại;
  - `translations`: 3–5 câu tiếng Việt đơn giản, `en` dùng ít nhất 1 từ trong danh sách, 3–4 `distractors` là từ đơn tiếng Anh không có
    trong `en`;
  - không markdown.
- Tối đa 30 từ gửi cho AI (theo thứ tự xuất hiện trong bài).
- Log `op=practice` kèm số từ, số câu; không ghi nội dung.

**Lý do**: nguyên tắc III (qua interface, đổi nhà cung cấp bằng cấu hình); một request tách riêng để AI lỗi ở đây không ảnh hưởng chú
thích (FR-001, FR-008). Temperature cao hơn annotate một chút để hội thoại tự nhiên, vẫn đủ ổn định cho schema.

**Phương án đã loại**: gộp vào request `Annotate` (trái khối 1: "1 request riêng"; schema chú thích đã lớn, lỗi một phần kéo theo cả hai).

## R2. Lưu trong tài liệu bài và trạng thái

**Quyết định**

- `lesson.Lesson` thêm `Practice *Practice`, `PracticeStatus Status`, `PracticeError string`, `PracticeVersion int`.
- Trạng thái: `""` (chưa có) | `running` | `done` | `failed`; API trả `"none"` cho `""`. **Lệch đầu vào** (khối 2 ghi
  `none|pending|done|failed`): dùng `running` cho khớp `Status` hiện có của audio và chú thích, để frontend dùng lại `JobStatus` và
  `isRunning()`/`pollWhile` ở trang quản trị.
- `PracticeVersion` tăng mỗi lần lưu phần luyện tập; dùng cho đường dẫn audio (R7) và để ghi có điều kiện.
- `Lesson.StatusOf` và mongo `SetStatus` đổi sang `switch` rõ ràng: `TypeTTS` → audio, `TypeAnnotate` → chú thích, `TypePractice` →
  luyện tập, loại khác → lỗi (thay vì "còn lại là chú thích").

**Lý do**: phần luyện tập thuộc về một revision của bài như chú thích; lưu cùng tài liệu bài thì ghi có điều kiện `{_id, revision}` sẵn
có (`updateAtRevision`) đã chống ghi đè từ job cũ. Không cần collection mới (nguyên tắc VII).

**Phương án đã loại**: collection `lesson_practice` riêng (thêm repository, index, dọn dữ liệu khi xoá bài mà không lợi gì).

## R3. Luồng job sinh phần luyện tập

**Quyết định**

- `job.TypePractice = "practice"`. Xếp hàng ở 2 nơi:
  1. `ProcessAnnotate` lưu chú thích thành công (`SaveAnnotations` trả `ok`): `SaveAnnotations` đồng thời đặt `practice = nil`,
     `practiceStatus = running`, `practiceError = ""` trong cùng lệnh ghi; sau đó enqueue `TypePractice` cùng revision.
  2. Quản trị viên bấm Tạo lại: `Service.RegeneratePractice` — từ chối nếu `AnnotationStatus != done` (`ErrAnnotationNotDone`) hoặc
     `PracticeStatus == running` (`ErrPracticeRunning`); đặt `practiceStatus = running` (giữ nội dung cũ đến khi có bản mới), enqueue.
- `ProcessPractice(ctx, j)`:
  1. `s.current(ctx, j)`: bài không còn → permanent; revision khác → bỏ qua (job xong âm thầm).
  2. Không có chú thích → `job.Permanent` ("chưa có chú thích").
  3. `practiceWords(l)`: từ chú thích, gộp theo lemma đã chuẩn hoá, giữ lần xuất hiện đầu, tối đa 30.
  4. `AI.Practice`. `ErrNotConfigured`/`ErrInvalidKey` → permanent; lỗi khác (kể cả hết lượt, mạng) → retry theo worker.
  5. `CleanPractice` (R4). Không còn phần nào dùng được → `job.Permanent(ErrNoValidPractice)`: **không retry** để mỗi lần sinh tốn tối
     đa 1 request có kết quả (SC-003).
  6. `SavePractice(id, revision, l.PracticeVersion, p)`: ghi có điều kiện `{_id, revision, practiceVersion}`, `$set practice,
     practiceStatus=done, practiceError=""`, `$inc practiceVersion`. Không khớp → bỏ qua.
  7. Enqueue `TypePracticeAudio` (R7). Đã lưu xong thì lỗi xếp hàng chỉ ghi log, job vẫn xong: retry sẽ gọi lại AI.
- `JobFailed` với `TypePractice` → `SetStatus(..., failed, failureMessage)` (thông báo tiếng Việt); không đụng chú thích, câu hỏi, đề viết.
- Sửa nội dung bài: `ReplaceContent` đặt thêm `practice = nil`, `practiceStatus = ""`, `practiceError = ""`; chuỗi annotate → practice tự
  chạy lại theo revision mới. Quản trị viên sửa tay chú thích (`ReplaceAnnotations`) không tự sinh lại (bấm Tạo lại nếu muốn).
- Chạy lại chú thích (`Retry` annotate): phần luyện tập cũ giữ nguyên cho tới khi chú thích mới lưu thành công (bước 1 ở trên xoá và sinh
  lại). Chú thích chạy lại mà lỗi thì chú thích và phần luyện tập cũ đều còn, vẫn khớp nhau.
- Thứ tự an toàn nhờ chỉ có **một worker** xử lý tuần tự: job practice luôn đọc chú thích mới nhất tại lúc chạy; job cũ đến sau gặp
  revision/version khác thì bỏ qua.
- Bài cũ (chú thích xong trước F17, chưa có `practiceStatus`): khi backend khởi động và đã cấu hình AI, `QueueMissingPractice` đặt
  `running` và xếp job cho từng bài, mỗi bài đúng một lần (đổi 2026-10-02 theo yêu cầu người dùng). Bài sinh lỗi thành `failed`, không tự
  xếp lại; quản trị viên bấm Tạo lại.

**Lý do**: đúng nguyên tắc III (chạy nền, có trạng thái, chạy lại được, lưu kết quả) và FR-001, FR-008, FR-009.

**Phương án đã loại**

- Chống trùng bằng index unique trên `jobs`: kiểm tra `PracticeStatus == running` như `Retry` annotate là đủ cho một quản trị viên.
- Tự sinh lại cả bài lỗi mỗi lần khởi động: tốn hạn mức AI không kiểm soát.

## R4. `CleanPractice`

**Quyết định** — hàm thuần trong `internal/lesson/practice.go`, nhận `ai.Practice` và danh sách từ, trả `Practice` đã làm sạch hoặc
`ErrNoValidPractice` khi không còn ví dụ, hội thoại và câu dịch nào:

- Mọi chuỗi: trim, gộp khoảng trắng.
- `objectiveVi` ≤ 200 ký tự, `grammarTipVi` ≤ 400 ký tự (dài hơn thì bỏ, không cắt giữa câu).
- `examples`: lemma phải thuộc danh sách từ; câu ≤ 200 ký tự và chứa **nguyên từ** (`containsWord`: so không phân biệt hoa thường, theo
  ranh giới từ, khớp `text` hoặc `lemma`, hỗ trợ cụm nhiều từ); trùng lemma giữ câu đầu.
- `dialogue`: đúng 2 tên không rỗng (thiếu thì bỏ cả đoạn); bỏ lượt có `speaker` ngoài {0,1}, `text` rỗng hoặc > 200 ký tự, `meaningVi`
  rỗng; còn < 4 lượt → bỏ cả đoạn; > 10 → giữ 10 lượt đầu.
- `translations`: `vi`, `en` không rỗng; `en` tách được 2–15 ô (R6) và chứa ít nhất 1 từ vựng (`containsWord`); `distractors` trim, bỏ
  rỗng, bỏ trùng (không phân biệt hoa thường), bỏ từ trùng với một ô của đáp án, tối đa 4; tối đa 5 câu.

**Lý do**: khối 1 "câu nào hỏng thì bỏ câu đó"; giới hạn số lượng theo khối 2.

## R5. Ô trống bước 3 (`BuildFill`)

**Quyết định** — tính ở backend lúc trả API (không lưu, không cần AI), hàm thuần:

1. Tách mỗi lượt thành token giữ nguyên vị trí (từ và phần không phải chữ).
2. Ứng viên: mọi lần xuất hiện nguyên từ của một từ vựng (khớp `text` hoặc `lemma`, không phân biệt hoa thường, cụm nhiều từ khớp liên
   tiếp, không chồng nhau).
3. Chọn tối đa 5 ô, trải đều: lặp vòng qua các lượt theo thứ tự, mỗi vòng lấy ở mỗi lượt 1 ứng viên đầu tiên chưa chọn, ưu tiên từ vựng
   chưa được chọn; dừng khi đủ 5 hoặc hết ứng viên. Sắp lại các ô theo vị trí trong hội thoại để đánh số.
4. Kết quả: `turns[{speaker, meaningVi, audioUrl, parts: [{text} | {blank: i}]}]`, `blanks[{answer}]` (`answer` = chữ đúng như trong câu,
   bỏ dấu câu bao quanh), `wordBank` = các `answer` + tối đa 2 `text` của từ vựng khác (chưa là đáp án, theo thứ tự trong bài).
5. Không có ứng viên nào → `fill = null` (bước 3 báo chưa có phần luyện tập).

`wordBank` trả theo thứ tự xác định; **frontend xáo** mỗi lần mở và mỗi lần Làm lại (test backend xác định được).

**Lý do**: đáp án luôn là từ vựng của bài (tiêu chí nghiệm thu), không tốn request AI.

## R6. Ô từ câu dịch và cách chấm

**Quyết định**

- `answer = strings.Fields(en)`: dấu câu dính theo từ ("you,", "Vietnam."), giữ chữ hoa.
- `tiles = answer + distractors` (thứ tự xác định; frontend xáo).
- Chấm ở trình duyệt:
  - điền từ: `normalize(ô) === normalize(answer)` với `normalize` = trim + chữ thường;
  - dịch câu: mảng chữ của các ô đã ghép === `answer` (so đúng từng phần tử, phân biệt hoa thường vì ô giữ nguyên chữ). Hai ô cùng chữ thay
    được cho nhau vì so theo chữ, không theo id ô.
- Đáp án gửi kèm API (FR-019, không có điểm số nên chấp nhận được).

## R7. Audio phần luyện tập

**Quyết định**

- Job riêng `job.TypePracticeAudio = "practice_audio"`, không mang version (`jobs.targetId` chỉ nhận ObjectID): job luôn tạo audio cho
  version hiện tại của bài. **Lệch đầu vào**: khối 2 tạo audio "sau khi lưu practice"
  ngay trong job; tách job để TTS lỗi thì retry mà **không gọi lại AI**.
- `ProcessPracticeAudio`: bài/revision không khớp hoặc bài không còn phần luyện tập → bỏ qua. Lần lượt tổng hợp `example-{i}`, `turn-{i}`, `answer-{i}` (i theo
  mảng đã làm sạch; đáp án đọc `en`), bỏ qua file đã có và khác rỗng (chạy tiếp được), ghi bằng `writeFileAtomic`. Xong thì xoá các thư
  mục version khác trong `{audioDir}/{id}/{rev}/practice/`. Lỗi → retry theo worker; hết lượt thử thì `JobFailed` chỉ log, **không** đổi
  `practiceStatus` (nội dung vẫn dùng được, nút nghe của câu thiếu audio bị ẩn).
- Đường dẫn: `{audioDir}/{id}/{rev}/practice/{ver}/{kind}-{i}.mp3`. **Lệch đầu vào**: thêm `{ver}` vì audio phục vụ với
  `Cache-Control: immutable`; Tạo lại trong cùng revision mà dùng chung URL thì trình duyệt phát audio cũ.
- Route `GET /api/audio/{lessonId}/{revision}/practice/{version}/{kind}/{index}` (requireAuth như route audio hiện có); kiểm tra id 24 hex,
  revision/version/index là số không âm, `kind ∈ {example, turn, answer}`, sai → 404. Cùng header với `AudioHandler`.
- Nằm trong thư mục revision nên `removeOtherRevisions` (F4) và xoá bài tự dọn.
- `audioUrl` trong API: `Reader` kiểm tra file tồn tại (`os.Stat`, ≤ ~40 file mỗi bài) → URL, không có → `null`. Không ghi đường dẫn
  vào DB.
- Từ đơn ở bước 1 dùng lại `GET /api/tts/word?text=` (F5), không tạo audio mới.

**Phương án đã loại**: lưu đường dẫn audio vào DB như `SaveAudio` — thêm một lệnh ghi có điều kiện cho mỗi lần tạo audio mà không lợi gì
so với `os.Stat`.

## R8. API người học

**Quyết định**

- `GET /api/lessons/{id}/practice`, đăng ký trong `ReadingHandler.Register` → `requireAuth(guard(...))` (cùng guard L với
  `GET /api/lessons/{id}`: 403 `lesson_locked` cho bài sắp tới, 404 bài không tồn tại).
- `Reader.Practice(ctx, id) (PracticeView, error)`. `NewReader` nhận thêm `audioDir`.
- `lessonNumber`: port `Topics` thêm `Position(ctx, topicID, lessonID) (int, error)` = vị trí (bắt đầu từ 1) trong `LessonIDs` của chủ
  đề, `0` khi bài không thuộc lộ trình → frontend ẩn "Bài N". Cài trong `lessonTopicsPort` (`cmd/api/main.go`) bằng `topicSvc.Get`.
- Không có endpoint nộp kết quả (FR-019).

## R9. Trang chi tiết bài (frontend)

**Quyết định**

- Viết lại `features/lesson/lesson-detail/` (route `/lessons/:id` giữ nguyên). Trang cha giữ state bằng signal: `tab`, `step` (1–4),
  `translateIndex`, kết quả bước 3/4, `seed` xáo. Các bước là component con trong cùng thư mục: `vocab-step/`, `dialogue-step/`,
  `fill-step/`, `translate-step/`, `practice-summary/`; logic thuần (xáo, chấm, đếm tổng kết) trong `lesson-detail/practice-logic.ts`
  (có spec), không đưa vào `shared/utils` vì chỉ trang này dùng.
- Dữ liệu (song song, `forkJoin`, mỗi cái `catchError` riêng trừ bài):
  - `reading.getLesson(id)` — tiêu đề, mức độ, câu, đoạn, ghi chú ngữ pháp (tab Bài đọc). 403 `lesson_locked` → hiện "Bài này sẽ mở
    khi tới lượt" và nút về danh sách; 404 → "Không tìm thấy bài".
  - `reading.vocabulary(id)` — từ, phiên âm, nghĩa cho bước 1.
  - `reading.practice(id)` — lỗi hoặc chưa có → coi như chưa có phần luyện tập.
  - `study.myLessons()` — bài hôm nay hay đã học (nút ở tổng kết, như bản cũ).
  - `dashboard.dashboard()` — **streak**. Không dùng `GET /api/today` vì có tác dụng phụ (tự hoàn thành bước Ôn khi không có thẻ). Lỗi
    → ẩn streak.
- Nhãn mức độ: A1–A2 Cơ bản, B1–B2 Trung cấp, C1–C2 Nâng cao (hàm thuần trong `practice-logic.ts`).
- Thanh tab đáy vẫn hiện (trang không phải focus page). Nút Tiếp theo `position: fixed; bottom: var(--tabbar-h, 0px)`, nền
  `--color-bg`; `.screen` thêm `padding-bottom` bằng chiều cao nút để không che nội dung. Từ 768px `--tabbar-h = 0` nên nút nằm sát đáy.
- Tab Bài đọc chỉ hiển thị (đoạn văn + ghi chú ngữ pháp), không có tra từ; muốn tra từ thì dùng Đọc lại.
- Phần luyện tập sinh xong khi đang mở trang: không poll, không đổi nội dung (edge case).

## R10. Phát hội thoại cả đoạn và tốc độ

**Quyết định**

- Dùng `lu-audio-player` sẵn có: phát lần lượt `audioUrl` của các lượt (bỏ lượt `null`), sự kiện `finished` → lượt kế, `failed` → bỏ
  qua lượt đó. Lượt đang phát được đánh dấu (`aria-current`).
- `lu-audio-player` thêm output `timeupdate` (giây hiện tại) để hiện thời gian đã phát `mm:ss` = tổng thời lượng các lượt đã xong +
  `currentTime`; kèm "Lượt x/y". Không hiện tổng thời lượng (không biết trước khi tải hết file).
- Tốc độ `0.75× / 1× / 1.25×`: radiogroup như bước Nghe (`listening.html:64-79`), chép markup nhỏ vào `dialogue-step` thay vì tách
  component dùng chung (chỉ 2 chỗ dùng, khác danh sách tốc độ).
- Nghe từng lượt, câu ví dụ, đáp án câu dịch, "Nghe câu" ở bước 3: cùng một `lu-audio-player` của trang (phát cái mới thì dừng cái cũ).
- Icon: thêm `play`, `pause`, `volume` vào `IconName` (SVG nét tự vẽ cùng phong cách bộ icon hiện có).

## R11. Tương tác điền từ, ghép câu và trợ năng

**Quyết định**

- Ô trống là `<button>` (`aria-pressed` khi đang chọn, `aria-label` "Ô trống 2: trống" / "Ô trống 2: hello"); từ trong ngân hàng là
  `<button>`, đã dùng thì `disabled` + mờ; ✕ là button riêng `aria-label="Gỡ từ hello"`. Mọi thứ dùng Tab/Enter/Space.
- Sau khi Kiểm tra: mỗi ô có biểu tượng ✓/✗ và chữ "Đúng"/"Sai" (ô sai kèm "Đáp án: …"); dòng kết quả `role="status"` ("Chính xác!" hoặc
  "Đúng 3/5 ô"). Đã kiểm tra thì khoá ô (muốn làm lại dùng Làm lại ở tổng kết).
- Ghép câu: ô đã chọn hiện trong vùng "Câu dịch của bạn" theo thứ tự chạm, mỗi ô có ✕; "Làm lại" xoá hết; Kiểm tra khoá khi câu trống.
  Sai → "Chưa đúng" + "Câu đúng: …"; sau khi kiểm tra nút loa đọc đáp án dùng được.
- Tiếp theo luôn bấm được; bước 4 chuyển câu, câu cuối thành "Hoàn thành". Ô/câu chưa kiểm tra tính sai khi tổng kết.
- Tab Bài học/Bài đọc: `role="tablist"`, phím mũi tên trái/phải.
- Vùng chạm ≥ 44px; ô từ xuống dòng (`flex-wrap`) để không cuộn ngang ở 360px; màu chỉ dùng token (`--color-ok`, `--color-bad`, …).

## R12. Trang quản trị

**Quyết định**

- `features/admin/lesson-detail/practice-section/` (theo mẫu `lesson-extras`): `lesson = input.required<Lesson>()`,
  `regenerated = output<Lesson>()`. Hiện `lu-status-chip label="Luyện tập"` + lỗi, nội dung (mục tiêu, câu ví dụ, hội thoại, mẹo ngữ
  pháp, câu dịch + từ nhiễu) chỉ đọc, nút **Tạo lại phần luyện tập** (khoá khi chú thích chưa xong hoặc đang sinh, kèm lý do).
- `AdminApiService.regeneratePractice(id)` → `POST /api/admin/lessons/{id}/practice/regenerate` (202 + lesson).
- `isRunning()` tính cả `practiceStatus` để trang quản trị tự poll khi đang sinh.

## R13. Kiểm thử

- Go (không mạng, theo khối 2):
  - `lesson/practice_test.go`: `CleanPractice` (từng quy tắc R4), `containsWord`, `BuildFill` (khớp text/lemma, cụm, nguyên từ, trải
    đều, tối đa 5, wordBank), tách ô (R6).
  - `lesson/practice_job_test.go`: annotate xong → practice xếp hàng và phần cũ bị xoá; ProcessPractice thành công (1 lần gọi AI, lưu,
    enqueue audio); AI lỗi → failed, chú thích/câu hỏi giữ nguyên; nội dung hỏng hết → failed không retry; revision cũ bị bỏ; Regenerate
    (từ chối khi đang chạy / chưa có chú thích); sửa nội dung xoá practice; ProcessPracticeAudio (bỏ file đã có, version cũ bị bỏ, dọn
    version khác).
  - `lesson/practice_handler_test.go`: GET practice 200 (có/không có practice, audioUrl null/khác null), 403 `lesson_locked`, 404; admin
    regenerate 202/409/404/403 không phải admin; route audio practice (kind sai → 404).
  - `ai/gemini/gemini_test.go`: `TestPractice` (schema, prompt chứa từ và trình độ), `TestPracticeErrors`,
    `TestPracticeWithoutKeyMakesNoRequest`.
  - `storage/mongo`: chuyển đổi `practiceDoc` ↔ `lesson.Practice`.
- Vitest: `practice-logic.spec.ts` (xáo có seed, chấm, nhãn mức độ, tổng kết); `lesson-detail.spec.ts` (đầu trang, tab, chuyển bước và
  tiến độ, chưa có phần luyện tập, bài bị khoá, tổng kết và Làm lại, nút về bài theo loại bài); `fill-step.spec.ts` (chọn ô, điền, gỡ,
  thay, Kiểm tra đúng/sai có chữ, bàn phím); `translate-step.spec.ts` (ghép đúng/sai thứ tự, Làm lại, ô trùng chữ); `dialogue-step.spec.ts`
  (phát lần lượt, đổi tốc độ, bật/tắt nghĩa); `vocab-step.spec.ts`; `practice-section.spec.ts` (admin); `reading-api`/`admin-api`
  (URL mới); `audio-player.spec.ts` (`timeupdate`).
- Thủ công: `quickstart.md` (360px, sáng/tối, bàn phím, AI thật).
