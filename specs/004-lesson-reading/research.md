# Research: Đọc (F3)

Không còn mục NEEDS CLARIFICATION.

## R1. Từ điển minhqnd/dictionary

- **Dữ liệu**: release `v2.0.0`, file `dictionary.db` 178 987 008 byte,
  SHA-256 `9259403f0675b2991a1bd0ef6d0dbc5933afdb135632af095a60662f09bbf1d3`, giấy phép MIT.
- **Schema thật** (đã mở file kiểm tra): `words(id, word, lang_code, source_id)` với unique index
  `(word, lang_code)`; `word_definitions(word_id, definition_id, example)`; `definitions(id, definition, pos,
  definition_lang)`; `pronunciations(word_id, ipa, region)`. 118 356 từ `lang_code='en'`, 118 183 có nghĩa
  `definition_lang='vi'`. Từ lưu **chữ thường** (`monday`).
- **Truy vấn** (một lần cho mỗi ứng viên, dùng index):

  ```sql
  SELECT id FROM words WHERE word = ? AND lang_code = 'en';
  SELECT d.pos, d.definition FROM word_definitions wd JOIN definitions d ON d.id = wd.definition_id
   WHERE wd.word_id = ? AND d.definition_lang = 'vi' ORDER BY wd.id LIMIT 3;
  SELECT ipa FROM pronunciations WHERE word_id = ? ORDER BY (ipa LIKE '/%') DESC, id LIMIT 1;
  ```

  Mỗi nghĩa trả kèm `pos` (N, V, A…); IPA ưu tiên dạng có `/…/`.
- **Đính chính khi triển khai**: từ điển **có** mục cho nhiều dạng biến đổi nhưng nghĩa đầu tiên chỉ là ghi chú,
  ví dụ *went* "động từ quá khứ của go.", *studies* "Động từ chia ở ngôi thứ ba số ít của study", *children* "Số nhiều
  của child", *better* "cấp so sánh của good" (lần truy vấn đầu bị `LIMIT 40` che mất). → `dictionary.Resolve`: tra
  chính từ; nếu **nghĩa đầu tiên** là ghi chú dạng biến đổi thì chuyển sang từ gốc được nêu; không có mục thì thử ứng
  viên của R2. Từ có nghĩa riêng trước (*saw* "Cái cưa.", *left*) giữ nghĩa riêng. Đo với file thật: ~0,5 ms mỗi lần.
- **Không có** cụm (*give up*, *look forward to*) trong từ điển → cụm chỉ tra được qua chú thích của bài. Spec kịch bản US3-2 (*look forward to* từ từ điển) vì vậy thường rơi xuống "Chưa có nghĩa" — ghi
  nhận trong quickstart; tiêu chí nghiệm thu không phụ thuộc kịch bản này.
- **Driver**: `modernc.org/sqlite` **v1.59.0** (go 1.25.0; v1.60.0 yêu cầu Go 1.26). Mở
  `file:<path>?mode=ro&immutable=1` (chỉ đọc, bỏ khoá file); `SetMaxOpenConns(4)`.
- **Thiếu file**: log `warn` và dùng `NoDictionary` (luôn "không có"); app không lỗi (edge case trong spec).
- **Phân phối**: file lớn không vào repo/image. Script `deploy/fetch-dictionary.sh` và `.ps1` tải về
  `deploy/data/dictionary/dictionary.db` và kiểm tra SHA-256; compose bind mount `./data/dictionary:/data/dictionary:ro`.
  Chạy backend local: `DICTIONARY_PATH=../deploy/data/dictionary/dictionary.db` trong `backend/.env`.

## R2. Đưa về dạng gốc (FR-009)

- **Decision**: `dictionary.Candidates(word) []string` trả danh sách ứng viên theo thứ tự, bắt đầu bằng chính từ
  (chữ thường, bỏ `’`→`'`):
  1. Bảng bất quy tắc (~180 động từ thông dụng: went/gone → go, saw/seen → see…; danh từ: children → child, men → man,
     women, people → person, mice, feet, teeth, geese).
  2. Quy tắc: `-ies → -y` (studies → study), `-es → ""` (watches), `-s → ""` (parks), `-ied → -y`, `-ed → ""`,
     `-ed → -e` (liked → like), phụ âm đôi + `-ed` (stopped → stop), `-ing → ""`, `-ing → -e` (making → make), phụ âm
     đôi + `-ing` (running → run), `-'s` sở hữu.
  3. Bỏ trùng, giữ thứ tự.
- Lookup thử lần lượt từng ứng viên ở từ điển; ứng viên đầu tiên có mục là dạng gốc. Không có → "Chưa có nghĩa".
- **Rationale**: đơn giản (nguyên tắc VII), đủ cho *went, studies, running, stopped*; kiểm chứng bằng từ điển nên quy tắc
  thừa không gây sai.
- **Alternatives considered**: Porter stemmer (ra gốc không phải từ thật: *studi*); thư viện lemmatizer (phụ thuộc lớn).

## R3. Thứ tự tra cứu (FR-007)

Với truy vấn `q` (từ hoặc cụm như người học chọn) và `sentence = N`:

1. **Chú thích**: so khớp không phân biệt hoa thường `q` với `annotation.text`, rồi với `annotation.lemma`, rồi từng
   `Candidates(q)` (với từ đơn) với `lemma`. Ưu tiên mục có `sentenceIndex == N`, sau đó mục ở câu khác. → `source: "ai"`,
   `lemma` = lemma của chú thích, nghĩa = `meaningVi`; IPA lấy từ từ điển theo lemma (nếu có).
2. **Từ điển**: thử `q` rồi `Candidates(q)` (chỉ với từ đơn). → `source: "dictionary"`.
3. Không có → 404 `not_found`. Frontend hiện "Chưa có nghĩa" + ô tự nhập.

Không có bước nào gọi AI (FR-008, kiểm bằng test: `Lookup` không nhận `ai.Provider`).

## R4. Bản đồ dạng gốc cho bài (tô màu, FR-003)

- `GET /api/lessons/{id}` trả `lemmas: {"went":"go","studies":"study",…}` cho mọi token khác nhau trong bài (khoá chữ
  thường), tính bằng đúng logic R3 (chú thích trước, từ điển sau; không tìm thấy thì không có khoá) và `phrases:
  [{text, lemma}]` các chú thích nhiều từ.
- Frontend tô token nếu `lemmas[token] ?? token` ∈ tập lemma trong sổ; tô cụm nếu lemma của cụm trong sổ và chuỗi
  `phrase.text` xuất hiện trong câu (hoặc lemma cụm xuất hiện nguyên văn).
- Chi phí: bài 10 000 ký tự ~ 700 token khác nhau × ≤ 6 ứng viên truy vấn SQLite có index → vài chục ms; không cache.

## R5. Tách token (dùng chung quy tắc ở backend và frontend)

- Regex từ: `[\p{L}]+(?:['’\-][\p{L}]+)*` (chữ cái Unicode; nháy/gạch nối **ở giữa** thuộc từ: *don't*, *well-known*).
  Phần còn lại (khoảng trắng, dấu câu, số) là token không bấm được.
- Frontend `tokenize(sentence) → [{text, isWord, index}]`; backend `lesson.Words(text) []string` để tính `lemmas`. Hai
  bên có bộ test mẫu giống nhau.

## R6. Chọn cụm (FR-005, US3)

- Mỗi từ là `<span class="w" data-s="{câu}" data-t="{token}">`. Trên `pointerup` / `keyup` và `selectionchange`
  (debounce 150ms, chỉ khi selection không rỗng): lấy `Range`, tìm các span từ giao với range, lấy câu của span đầu tiên,
  bỏ span thuộc câu khác, mở rộng thành từ trọn vẹn. 1 từ → như chạm; 2–6 từ → tra cụm, text = nối nguyên văn đoạn từ
  đầu tới cuối trong câu; > 6 → gợi ý "Chọn tối đa 6 từ".
- Sau khi mở popup, xoá selection để menu hệ thống không che popup trên điện thoại.

## R7. Popup (FR-005, FR-006, FR-010)

- `@angular/cdk/overlay` `cdkConnectedOverlay` gắn vào phần tử từ (hoặc span đầu cụm): vị trí ưu tiên **dưới** từ, dự
  phòng **trên**; `cdkConnectedOverlayPush` để nằm trong màn hình; bề rộng `min(320px, 100vw - 16px)`; `offsetY` 8px để
  không che từ.
- Đóng: `overlayOutsideClick`, phím Esc (`overlayKeydown`), nút ✕; focus vào popup khi mở, trả focus về từ khi đóng.
  `role="dialog"`, `aria-labelledby` = từ.
- Nội dung: từ, `→ lemma` nếu khác, IPA (`--font-ipa`), nhãn nguồn ("AI · theo ngữ cảnh" / "Từ điển"), tối đa 3 nghĩa
  (kèm từ loại N/V/A viết tiếng Việt: danh từ, động từ, tính từ…), nút ▶ nghe, nút "Lưu vào sổ từ" hoặc "✓ Đã có trong
  sổ". Không có nghĩa: "Chưa có nghĩa" + ô nhập (≤ 200 ký tự) + "Lưu".

## R8. Phát âm từ (FR-011, FR-012)

- `GET /api/tts/word?text=…` (RequireAuth). `text` chuẩn hoá (trim, gộp khoảng trắng, chữ thường), 1–60 ký tự, chỉ chữ,
  khoảng trắng, `'`, `-` → khác thì 400. File `{AUDIO_DIR}/words/{sha256(text)}.mp3`; có sẵn thì phục vụ ngay, chưa có
  thì gọi `tts.Synthesizer` (Kokoro) rồi ghi atomic. `singleflight` theo hash để nhiều yêu cầu cùng lúc chỉ tạo một lần.
  `Cache-Control: private, max-age=31536000, immutable`. Kokoro lỗi → 503 `tts_unavailable`.
- Frontend phát bằng một `Audio` dùng chung; phát lại thì `currentTime = 0` (không chồng tiếng).

## R9. Sổ từ tối thiểu

- `cards {userId, text, lemma, ipa, meaningVi, contextSentence, lessonId, source, createdAt}`, unique index
  `(userId, lemma)`; `lemma` lưu chữ thường, trim, gộp khoảng trắng.
- `POST /api/vocab/cards`: userId lấy từ phiên. Trùng → 409 `card_exists` kèm thẻ đang có (FR-015). Kiểm tra: `text`,
  `lemma` 1–100 ký tự, `meaningVi` 1–200 (nhiều nghĩa nối bằng "; " ở client), `contextSentence` ≤ 1000, `lessonId` hợp
  lệ và bài tồn tại, `source` ∈ ai|dictionary|manual.
- `GET /api/vocab/words` → `{words: [{lemma, text}]}` của người gọi (tô màu). F5 mở rộng.

## R10. Nút Đã đọc xong (FR-019, FR-020)

- Phần tử "cuối bài" sau câu cuối; `IntersectionObserver` (threshold 0) bật cờ `reachedEnd` lần đầu nó hiện và ngắt quan
  sát. Bài ngắn thì phần tử hiện ngay khi render → nút bật ngay.
- Không có `IntersectionObserver` (trình duyệt cũ, jsdom): bật ngay để không khoá người học.
- Bấm → `output completed` + banner "Đã hoàn thành bước Đọc". Lưu tiến độ ở L.

## R11. Route và truy cập tạm

- Route `lessons/:id/read` (`authGuard`), lazy `features/lesson/lesson.routes.ts`. Trang chi tiết bài admin có nút "Mở
  bước Đọc". Người học mở bằng đường dẫn trực tiếp cho tới khi có L.
- `GET /api/lessons/{id}` (RequireAuth, mọi vai trò) trả bài cho người học: không có `annotations`, `revision`, lỗi việc
  nền; có `sentences`, `paragraphs`, `lemmas`, `phrases`.

## R12. Font IPA

- Import `@fontsource/noto-sans/latin-400.css` và `latin-ext-400.css` (dải latin-ext của Fontsource gồm IPA Extensions
  U+0250–02AF và dấu ˈ ˌ ː) — kiểm tra lại dải `unicode-range` khi triển khai. Chỉ popup dùng `--font-ipa`.
