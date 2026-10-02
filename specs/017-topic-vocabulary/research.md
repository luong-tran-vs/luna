# Research: Từ vựng theo chủ đề (F18)

Mọi mục NEEDS CLARIFICATION đã được giải quyết. Mỗi mục gồm quyết định, lý do, phương án đã loại. Mục nào lệch khối 2 của
`docs/spec-inputs/f18-tu-vung-chu-de.md` thì ghi **Lệch đầu vào**.

## R1. Lưu danh sách từ và kiểm tra

**Quyết định**

- `topic.Topic` thêm `Words []string`, `WordsSeeded bool`; Mongo `words`, `wordsSeeded`. Chủ đề cũ chưa có trường → `words` rỗng,
  `wordsSeeded=false`.
- `topic.CleanWords(in []string) ([]string, *ValidationError)` (hàm thuần):
  - trim, gộp khoảng trắng; bỏ dòng rỗng;
  - mỗi từ khớp `^[A-Za-z][A-Za-z '\-/]*$` và ≤ 40 ký tự, nếu không → `words.{i}`: "Chỉ dùng chữ cái tiếng Anh, khoảng trắng, - ' /"
    hoặc "Tối đa 40 ký tự";
  - trùng (so chữ thường) với từ đứng trước → `words.{i}`: "Từ bị trùng";
  - quá 100 từ → `words`: "Tối đa 100 từ".
  - Lỗi nào cũng không lưu gì (FR-004). Chỉ số `i` là vị trí trong mảng gửi lên, nên frontend gắn lỗi đúng dòng.
- `Repository.SetWords(id, words)` đặt `words` và `wordsSeeded=true` (sửa tay, kể cả xoá hết, không bao giờ bị nạp đè).

## R2. Dữ liệu ban đầu và so tên chủ đề

**Quyết định**

- Chép `docs/spec-inputs/f18-topic-words.json` vào `backend/internal/topic/seed/topic-words.json`, nhúng bằng `//go:embed`. Sửa 1 từ sai
  quy tắc ở cả hai file: "Intensive care unit (ICU)" → "Intensive care unit". Test kiểm tra mọi từ của file nhúng qua `CleanWords`
  (42 chủ đề, 1.257 từ, không lỗi).
- `topic.SeedKey(name)`: chữ thường, gộp khoảng trắng, đưa dấu thanh kiểu cũ về kiểu mới (`oà oá oả oã oạ oè oé oẻ oẽ oẹ uỳ uý uỷ uỹ uỵ`
  → `òa óa ỏa õa ọa òe óe ỏe õe ọe ùy úy ủy ũy ụy`, cả chữ hoa), nên "Sức khoẻ" = "Sức khỏe". Không thêm thư viện chuẩn hoá Unicode:
  tên gõ trên trình duyệt và file dữ liệu đều ở dạng dựng sẵn (NFC).
- `topic.PlanSeed(topics []SeedTarget, seed Seed) []SeedWrite` (hàm thuần): với mỗi chủ đề `!WordsSeeded`, tên khớp → ghi danh sách
  đã `CleanWords`; không khớp → ghi danh sách rỗng. File không có trình độ nên tên khớp ở nhiều trình độ thì mọi chủ đề đó đều nhận.
- `mongo.SeedTopicWords(ctx, db, seed, log)` chạy trong `PrepareInBackground` sau `MigrateTopics`: đọc chủ đề `wordsSeeded != true`,
  áp `PlanSeed`, mỗi chủ đề một `UpdateOne` có điều kiện `{_id, wordsSeeded: {$ne: true}}` (chạy lại hay chạy song song đều không ghi
  đè). Log `topic words seeded` với số chủ đề nhận từ và số chủ đề không khớp.
- Không dùng marker trong `migrations` như `MigrateTopics`: cờ theo từng chủ đề cho phép chủ đề tạo sau được xét ở lần khởi động kế tiếp
  (giả định của spec).

**Phương án đã loại**: nạp khi tạo chủ đề (không đúng khối 1 "khi khởi động"); `golang.org/x/text/unicode/norm` (thêm phụ thuộc trực
tiếp mà không cần, nguyên tắc VII).

## R3. Khớp từ trong bài (`internal/wordmatch`)

**Quyết định** — gói mới, thuần, dùng chung cho `topic` (độ phủ) và `lesson` (bản nháp, chú thích) vì hai gói không import nhau:

- `wordmatch.New(text string, lemmas map[string]string) *Text`: tách `text` thành token (chữ cái, `'`/`’` và `-` ở giữa từ; `/` và mọi
  ký tự khác là dấu ngắt). Mỗi token có tập dạng gốc: chính nó (chữ thường, `’`→`'`), `dictionary.Candidates(token)` (hậu tố s/es/ies/
  ed/ing và bảng bất quy tắc có sẵn), và `lemmas[token]` (lemma của chú thích đơn từ).
- `(*Text).Find(term string) (span string, ok bool)`: tách `term` cùng cách; khớp khi có một dãy token liên tiếp mà token thứ k có tập
  dạng gốc chứa `term[k]` (hoặc `Candidates(term[k])` giao khác rỗng — để "grandmothers" khớp "Grandmother", "took a shower" khớp
  "take a shower"). Trả đoạn chữ gốc trong `text` để dùng làm `Annotation.Text`.
- Chú thích cụm ("gave up" → "give up") được xử lý tự nhiên vì từng token đã có dạng gốc.
- Không khớp một phần từ ("son" ≠ "season").

**Lý do**: một cách khớp duy nhất cho độ phủ, "còn thiếu" của bản nháp và từ chú thích (FR-006, FR-013, FR-016). **Lệch đầu vào**: khối 2
đặt việc khớp trong `topic.Service`; tách thành gói riêng để `lesson` dùng lại mà không import `topic`.

**Phương án đã loại**: đặt trong `lesson` (topic phải import lesson); dùng `strings.Contains` (sai với "son"/"season").

## R4. Độ phủ và số bài dùng mỗi từ

**Quyết định**

- Port `topic.Lessons` thêm `Texts(ctx, topicIDs []string) (map[string][]LessonText, error)`; `LessonText{Content string; Lemmas
  map[string]string}` (lemma của chú thích đơn từ, chữ thường). Mongo `Lessons.TopicTexts`: `find({topicId: {$in}})` chiếu `topicId,
  content, annotations.text, annotations.lemma`.
- `topic.Coverage(words []string, lessons []LessonText) []WordUse{Text string; Used bool; LessonCount int}` (thuần, dùng `wordmatch`).
- `Service.List` tính `WordCount`, `UsedWordCount` cho mọi chủ đề bằng **một** lần gọi `Texts` (vài trăm bài, chấp nhận được).
  `Service.Words(id)` trả `[]WordUse` của một chủ đề.
- Bài thuộc chủ đề = mọi bài có `topicId` đó, kể cả chưa vào lộ trình (giả định của spec). Tính khi xem, không lưu.

## R5. Chia từ mục tiêu (`PlanWords`)

**Quyết định** — `topic.PlanWords(uses []WordUse, count, perLesson int) [][]string` (thuần):

1. Sắp thứ tự: chưa dùng trước, rồi `LessonCount` tăng dần, rồi thứ tự trong danh sách.
2. `k = min(perLesson, len(words))`. Duyệt vòng tròn theo thứ tự trên, mỗi bài lấy `k` từ chưa có trong nhóm của nó.
3. Khi `count*k ≤ len(words)` các nhóm không trùng nhau; thiếu thì vòng lại từ đầu thứ tự (từ dùng ít nhất được lặp trước).
4. `perLesson = 0` hoặc danh sách rỗng → `count` nhóm rỗng.

API `GET /api/admin/topics/{id}/word-plan?count=&perLesson=` (count 1–5, perLesson 0–15) → `{groups: [[…]]}`.

## R6. Sinh bài với từ mục tiêu

**Quyết định**

- `lesson.GenerateInput` thêm `TargetWords [][]string`. `TopicRef` thêm `Words []string` (port `lessonTopicsPort.Get` điền).
- `ValidateGenerate` + kiểm tra trong `Generate`: `targetWords` rỗng hoặc đúng `count` phần tử; mỗi nhóm ≤ 15 từ, không trùng (chữ thường);
  mỗi từ thuộc danh sách chủ đề (chữ thường) → nếu không: `targetWords.{i}.{j}`: "Từ không có trong danh sách của chủ đề". Từ gửi lên
  được thay bằng cách viết trong danh sách.
- `ai.GenerateRequest` thêm `TargetWords [][]string`. Prompt Gemini: với mỗi bài có nhóm không rỗng thêm "Lesson {i+1} must use every
  one of these words or phrases (any natural form): …". Schema không đổi; bản nháp theo thứ tự bài. Vẫn 1 request.
- `filterDrafts` giữ chỉ số gốc của bản nháp để gắn đúng nhóm. `Draft` thêm `TargetWords`, `MissingWords` (từ không `Find` được trong
  nội dung bản nháp, R3). Bản nháp không bị loại vì thiếu từ (FR-013).
- `count` bản nháp trả về ít hơn số nhóm: nhóm của bản nháp bị loại không hiện.

## R7. Chú thích với từ của chủ đề

**Quyết định**

- Đổi chữ ký `Annotate(ctx, sentences, level)` thành `Annotate(ctx, ai.AnnotateRequest{Sentences, Level, FocusWords})`; cập nhật
  `ai.Disabled`, Gemini, các fake.
- `ProcessAnnotate`: lấy `Topics.Get(l.TopicID).Words`; `focus` = các từ `Find` được trong nội dung bài (R3, không có lemma chú thích vì
  chưa có). Prompt: "Always include each of these words or phrases as annotations, with text copied exactly as it appears: …; they count
  toward the 25 items." — giữ nguyên khoảng 8–25 (không có giới hạn trong code, FR-017 đạt bằng thứ tự ưu tiên trong prompt).
- Sau `CleanAnnotations`: mỗi từ focus chưa có chú thích nào khớp (so `Find` trên text/lemma chú thích) → tìm câu đầu tiên chứa nó, lấy
  `span` làm `Text`, `Lemma` = từ của chủ đề (chữ thường), `MeaningVi` = nghĩa đầu tiên của `Dict.Resolve(lemma)`; không có nghĩa → bỏ
  qua (FR-018). `lesson.Deps` thêm `Dict Dictionary` (main đã có `dict`).
- Thêm vào đều xảy ra trước `SaveAnnotations`, vẫn 1 request AI. Chủ đề không có từ → như hiện nay.
- **Lệch đầu vào**: khối 2 chỉ yêu cầu prompt; bước bổ sung bằng từ điển là FR-018 của spec, để đạt tiêu chí "bài chú thích xong có
  mọi từ của chủ đề".

## R8. Frontend

**Quyết định**

- `core/models/topic.ts`: `Topic` thêm `wordCount`, `usedWordCount`; `TopicWord {text, used, lessonCount}`.
- `features/admin/topics`: mỗi dòng thêm "Từ vựng: đã dùng X/Y" (hoặc "Chưa có từ vựng") và link **Từ vựng** tới trang mới
  `admin/topics/:id/words`.
- `features/admin/topic-words/` (trang mới): tiêu đề "Từ vựng · {chủ đề}", danh sách từ với nhãn chữ "Đã dùng · n bài"/"Chưa dùng",
  nút ✕ từng từ (xoá tạm), ô `<textarea>` "Thêm từ" (mỗi dòng hoặc dấu phẩy), nút **Lưu** gửi cả danh sách (cũ chưa xoá + mới), lỗi
  `words.{i}` hiện dưới đúng từ, lỗi `words` ở đầu. Không cảnh báo khi rời trang chưa lưu (đơn giản); nút Huỷ đặt lại danh sách như lúc tải.
- `generate-dialog`: thêm input `topicId`, `topicWords: string[]`. Nếu `topicWords` rỗng: như cũ. Nếu có: trường số "Từ mục tiêu mỗi
  bài" (0–15, mặc định theo trình độ), gọi `wordPlan` khi mở và khi đổi số bài/số từ (debounce 300 ms, bỏ kết quả cũ); mỗi bài một nhóm:
  từ có nút ✕ `aria-label="Bỏ {từ} khỏi bài {i}"`, ô thêm `<input list>` gợi ý từ chủ đề chưa có trong nhóm (Enter để thêm; từ ngoài
  danh sách báo "Từ không có trong danh sách"). Gửi `targetWords`. Lỗi AI: dialog đang giữ form và nhóm nên giữ nguyên.
- `draft-list`: khi `targetWords.length > 0` hiện "Dùng a/b từ mục tiêu" và "Còn thiếu: …" (không chỉ màu). Không tính lại khi sửa bản
  nháp (thông tin của lần sinh).
- `roadmap.ts`: lấy `GET …/words` khi mở hộp thoại để có `topicWords`.
- `AdminApiService`: `topicWords(id)`, `setTopicWords(id, words)`, `wordPlan(id, count, perLesson)`; `generateLessons` gửi `targetWords`.

## R9. Kiểm thử

- Go: `topic` (`CleanWords`, `SeedKey`, `PlanSeed`, file nhúng hợp lệ, `Coverage`, `PlanWords`, handler words/word-plan 200/400/404/403),
  `wordmatch` (cụm, hoa thường, số nhiều, -ed/-ing, bất quy tắc, lemma chú thích, "son"/"season", `/`), `storage/mongo` (doc words,
  `PlanSeed` không ghi đè đã nạp), `lesson` (generate: validate targetWords, missingWords, chỉ số sau lọc; annotate: focus words gửi AI,
  từ bỏ sót thêm bằng từ điển, không có nghĩa thì bỏ), `ai/gemini` (prompt chứa TargetWords, FocusWords).
- Vitest: `topic-words` (thêm nhiều, trùng, xoá, lỗi theo từ, bàn phím), `topics` (nhãn độ phủ), `generate-dialog` (nhóm từ, bỏ/thêm,
  0 từ, chủ đề không có từ, giữ nhóm khi lỗi), `draft-list` (thiếu từ), `admin-api`.
- Thủ công: quickstart (khởi động lần đầu với DB thật, AI thật, 360px sáng/tối).
