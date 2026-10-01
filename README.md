# Luna

Web app học tiếng Anh cá nhân: luyện 4 kỹ năng Nghe, Nói, Đọc, Viết quanh một bài học mỗi ngày.

Tài liệu sản phẩm nằm trong [`docs/`](docs/); spec từng tính năng nằm trong [`specs/`](specs/).

## Chạy nhanh

Chỉ cần cài [Docker](https://docs.docker.com/get-docker/) (kèm Docker Compose v2). Từ thư mục gốc repo:

```bash
docker compose -f deploy/docker-compose.yml up --build
```

Mở <http://localhost:8000>. Lần đầu, bấm **Đăng ký** để tạo tài khoản: **tài khoản đầu tiên là quản trị viên**,
các tài khoản sau là người học. Sau khi đăng nhập là màn hình chính; nếu máy chủ hoặc cơ sở dữ liệu không chạy, chân trang
báo lỗi kết nối (bình thường không hiện gì).

Dừng app: `Ctrl+C`, hoặc chạy nền bằng `up -d` rồi dừng bằng:

```bash
docker compose -f deploy/docker-compose.yml down        # giữ dữ liệu
docker compose -f deploy/docker-compose.yml down -v     # xoá cả dữ liệu MongoDB (kể cả tài khoản)
```

Muốn tạo lại quản trị viên từ đầu: chạy `down -v` rồi `up`, tài khoản đăng ký đầu tiên sẽ là quản trị viên.
Xem danh sách tài khoản:

```bash
docker compose -f deploy/docker-compose.yml exec mongo mongosh luna --quiet --eval 'db.users.find({}, {email: 1, role: 1, timezone: 1}).toArray()'
```

## Soạn bài học (quản trị viên)

Đăng nhập bằng tài khoản quản trị viên rồi mở **Quản trị** (`/admin`):

1. **Chủ đề** (`/admin/topics`): tạo chủ đề theo trình độ, ví dụ "A1 · Gia đình". Tên không trùng trong cùng trình độ;
   chủ đề còn bài thì không xoá được.
2. **Thêm bài**: dán văn bản tiếng Anh và chọn chủ đề; trình độ của bài lấy theo chủ đề.
3. **Lộ trình** (`/admin/roadmap`): chọn chủ đề, thêm bài của chủ đề, kéo thả hoặc nút Lên/Xuống để xếp thứ tự. Mỗi chủ đề
   một lộ trình; chủ đề còn dưới 3 bài chưa học được cảnh báo. Đổi chủ đề của một bài đang ở lộ trình thì bài chuyển sang
   cuối lộ trình của chủ đề mới.

Dữ liệu từ trước F14 (chủ đề gõ tự do, một lộ trình chung) được chuyển tự động khi backend khởi động lần đầu: chủ đề tạo từ
các cặp trình độ và chủ đề của bài, bài chưa có chủ đề vào "Chung" cùng trình độ, lộ trình chung chia theo chủ đề giữ thứ
tự. Chỉ chạy một lần (đánh dấu trong collection `migrations`).

Sau khi lưu bài, hệ thống tự làm nền:

- **Audio từng câu** bằng [Kokoro](https://github.com/remsky/Kokoro-FastAPI) chạy trên CPU (service `kokoro`,
  image khoảng 5 GB, tải một lần). Audio lưu ở volume `audio-data` và không tạo lại.
- **Chú thích từ vựng** bằng Gemini (gói miễn phí). Cần khoá API: tạo tại
  <https://aistudio.google.com/apikey>, rồi ghi vào `deploy/.env`:

  ```bash
  GEMINI_API_KEY=khoá-của-bạn
  ```

  và chạy lại `docker compose -f deploy/docker-compose.yml up -d`. Không có khoá thì bài vẫn lưu và có audio,
  chú thích báo "AI chưa được cấu hình"; thêm khoá xong bấm **Chạy lại chú thích**. Tắt hẳn AI: `AI_PROVIDER=none`.

### Sinh bài bằng AI

Ở **Lộ trình**, chọn một chủ đề rồi bấm **Sinh bài bằng AI**. Trình độ và chủ đề lấy theo lộ trình đang mở; chọn:

- **Số bài**: 1–5 (mặc định 3).
- **Độ dài** mỗi bài: 50–800 từ, mặc định theo trình độ (A1 120, A2 160, B1 220, B2 300, C1 và C2 400). Bài lệch quá 20%
  bị loại.
- **Dạng bài**: bài đọc, hoặc hội thoại (mỗi lượt nói một dòng `Tên: câu nói`).
- **Ý chính**: không bắt buộc, tối đa 500 ký tự.

Mỗi lượt tốn **một** request AI cho cả lượt và có thể mất đến một phút. Các tiêu đề đã có trong chủ đề được gửi kèm để
AI tránh lặp.

Kết quả là các **bản nháp**, chỉ có trên trang đang mở và không được lưu vào cơ sở dữ liệu:

- Sửa tiêu đề và nội dung từng bản, bấm **Lưu** hoặc **Bỏ**, hoặc **Lưu tất cả**.
- Bài được lưu có nguồn "AI sinh", giấy phép "Nội dung do AI tạo". Bài được thêm vào cuối lộ trình rồi chạy audio và chú
  thích như bài tạo tay.
- Rời trang, tải lại hoặc đổi chủ đề khi còn bản nháp thì được hỏi lại, vì bản nháp sẽ mất.

AI chưa cấu hình, khoá sai, hết lượt hoặc trả về nội dung không dùng được thì trang báo rõ lỗi. Bản nháp và các lựa chọn
đã nhập vẫn giữ nguyên để thử lại.

Request sinh bài dài hơn mọi request khác: nginx (`frontend/nginx.conf`) và backend chỉ cho riêng route
`/api/admin/topics/{id}/generate` chờ tới 75 giây, các route còn lại vẫn 15 giây.

## Bước Đọc và từ điển

Bước Đọc tra nghĩa bằng chú thích AI của bài, rồi bằng từ điển Anh–Việt offline
([minhqnd/dictionary](https://github.com/minhqnd/dictionary), MIT, ~180 MB). Tải từ điển một lần trước khi chạy:

```bash
./deploy/fetch-dictionary.sh        # hoặc PowerShell: .\deploy\fetch-dictionary.ps1
```

Script lưu vào `deploy/data/dictionary/dictionary.db` (git bỏ qua) và kiểm tra SHA-256. Thiếu file thì app vẫn chạy,
tra từ chỉ dùng chú thích của bài.

Mở bước Đọc: trang chi tiết bài trong **Quản trị** → **Mở bước Đọc** (hoặc `/lessons/<id>/read`). Chạm một từ để tra,
bôi đen nhiều từ để tra cụm, bấm ▶ để nghe, **Lưu vào sổ từ** để lưu kèm câu.

### Câu hỏi hiểu bài và ngữ pháp

Khi chú thích một bài, AI viết luôn trong **cùng một request**:

- 3–5 câu hỏi trắc nghiệm hiểu bài: 4 lựa chọn, 1 đáp án, giải thích bằng tiếng Việt.
- Một ghi chú ngữ pháp tiếng Việt, kèm ví dụ chép từ bài.
- Một đề viết, dùng ở bước Viết (F8).

Câu hỏi hỏng và ví dụ không có trong bài bị bỏ; thiếu phần nào thì phần đó để trống, chú thích từ vẫn được lưu.

Ở bước Đọc:

- Mục **Ngữ pháp** mở sẵn, thu gọn được.
- Bài có câu hỏi thì không còn nút "Đã đọc xong": người học trả lời lần lượt từng câu.
  - Chọn xong là thấy ngay "✓ Đúng" hoặc "✗ Sai", đáp án đúng và lời giải thích. Câu đã trả lời không đổi được.
  - Thoát ra rồi vào lại vẫn tiếp tục đúng câu đang dở.
  - Trả lời hết (kể cả khi sai) thì bước Đọc hoàn thành. Máy chủ kiểm tra việc này: `POST /api/today/steps/read/complete` trả 409
    `read_incomplete` khi còn câu chưa trả lời.
- Bài chưa có câu hỏi (AI lỗi hoặc bài cũ) vẫn dùng nút "Đã đọc xong".
- Xem lại bài từ trang **Bài học** thì thấy các câu trả lời cũ.

Câu trả lời được lưu trong collection `reading_answers`, mỗi câu một lần cho mỗi phiên bản bộ câu hỏi (`quizVersion`). Khi
quản trị viên sửa câu hỏi, chạy lại chú thích hoặc sửa nội dung bài, bộ câu hỏi mới bắt đầu lại từ đầu; câu trả lời cũ vẫn
tính vào thống kê.

Quản trị viên sửa câu hỏi (thêm, xoá, đổi đáp án), ngữ pháp và đề viết ở trang chi tiết bài; ví dụ ngữ pháp phải chép đúng
từ bài. **Chạy lại chú thích** dùng được cả khi chú thích đã xong, nên bài có từ trước F15 bấm nút này là có thêm câu hỏi.
Nếu bài có chú thích từ hoặc câu hỏi đã sửa tay, trang hỏi xác nhận trước khi thay.

## Bước Nghe (chép chính tả)

Mở từ trang chi tiết bài → **Mở bước Nghe** (hoặc `/lessons/<id>/listen`); bài cần có audio "Xong". Nghe từng câu
(trước/sau, nghe lại, tốc độ 0.5x–1.25x, chữ ẩn mặc định), gõ lại câu rồi **Kiểm tra** (hoặc Enter):

- từ sai bị gạch ngang, kèm từ đúng; từ thừa bị gạch ngang; từ thiếu được gạch chân chấm;
- so sánh không phân biệt hoa thường và dấu câu; `don't` khác `dont`, `9.30` khác `9 30`.

Kết quả mới nhất của mỗi câu được lưu theo người học (`POST /api/lessons/{id}/dictation`); tỷ lệ đúng = số từ đúng / tổng
số từ (`GET /api/lessons/{id}/dictation/summary`). Kiểm tra hết các câu là hoàn thành bước Nghe. Sửa nội dung bài thì kết
quả cũ không còn được tính.

## Một ngày học

- **Mục tiêu** (`/goal`): chọn trình độ rồi một chủ đề của trình độ đó; mục tiêu là học hết lộ trình của chủ đề. Tiến độ từng
  chủ đề lưu riêng; đổi chủ đề khi bài hôm nay đã bắt đầu thì có hiệu lực từ ngày mai.
- **Bài hôm nay** (trang chủ → **Bắt đầu / Tiếp tục: <bước>**, `/today`): mỗi ngày một bài mới (sang ngày lúc 0 giờ theo múi giờ trong
  Cài đặt), học theo thứ tự **Ôn** (thẻ đến hạn, tối đa 30 thẻ/ngày, đổi trong Cài đặt; không có thẻ thì tự xong) → **Đọc** → **Nghe**. Thoát ra
  vào lại thì tiếp tục đúng bước và đúng câu. Xong bài: mục tiêu +1, chuỗi ngày học +1; nghỉ một ngày trọn thì chuỗi về 0.
- **Bài học** (header → **Bài học**, `/lessons`): bài hôm nay, các bài đã học (đọc/nghe lại, không đổi tiến độ), các bài sắp tới
  bị khoá — người học không mở được bài chưa tới lượt, quản trị viên thì được.

Thử sang ngày mới mà không phải chờ (lùi ngày học của tài khoản `hoc@example.com` về hôm qua):

```bash
docker compose -f deploy/docker-compose.yml exec mongo mongosh luna --quiet --eval '
  const u = db.users.findOne({email: "hoc@example.com"})._id;
  const t = new Date(Date.now() + 7 * 3600e3); t.setUTCDate(t.getUTCDate() - 1);
  const y = t.toISOString().slice(0, 10);
  db.study_days.updateMany({userId: u}, {$set: {dayKey: y}});
  db.lesson_progress.updateMany({userId: u}, {$set: {dayKey: y}});
  db.goals.updateMany({userId: u}, {$set: {effectiveFrom: y}})'
```

(`+ 7 * 3600e3` là múi giờ Việt Nam; đổi theo múi giờ của tài khoản.)

## Màn hình chính và thống kê

- **Màn hình chính** (`/`): chuỗi ngày học, thanh mục tiêu ("A1 · Gia đình", số bài đã xong / tổng số bài) kèm thanh Đọc và
  Nghe (số bài của lộ trình hiện tại đã xong bước đó), bài hôm nay với thanh bước, nút **Bắt đầu: <bước>** (chưa xong bước nào) hoặc
  **Tiếp tục: <bước>** tới đúng bước đang dở, và số thẻ cần ôn tới hết ngày mai (gồm cả thẻ còn nợ). Chưa có mục tiêu → **Chọn
  chủ đề**; xong bài hôm nay → **Ôn tự do**; hết bài → **Ôn tự do** / **Chọn chủ đề khác**. Số liệu tải lại mỗi lần quay về trang;
  mở trang không làm bài hôm nay "bắt đầu".
- **Thống kê** (màn hình chính → **Xem thống kê**, `/stats`): số từ đã học, số câu đã chép chính tả và tỷ lệ đúng, số bài đã
  xong bước Đọc / Nghe và số bài hoàn thành, tính trên mọi chủ đề. Thẻ **Hiểu bài** (F15) là tỷ lệ trả lời đúng câu hỏi hiểu bài
  trên mọi câu đã trả lời (hiện "—" khi chưa trả lời câu nào).
- **Trạng thái kết nối** (F0) nằm ở chân trang và chỉ hiện khi không kết nối được máy chủ hoặc cơ sở dữ liệu (kiểm tra lại sau mỗi
  lần chuyển trang).

## Cài đặt

Thanh trên → **Cài đặt** (`/settings`); cài đặt lưu theo tài khoản, đăng nhập máy khác vẫn giữ.

- **Giao diện**: Sáng, Tối, Theo hệ thống (mặc định); đổi là có hiệu lực ngay. Nút giao diện nhanh ở thanh trên cũng lưu vào tài
  khoản. Trước khi đăng nhập app dùng lựa chọn lưu trên trình duyệt; sau khi đăng nhập lựa chọn của tài khoản được dùng.
- **Số thẻ ôn mỗi ngày** (bước Ôn của bài hôm nay): 5–200, mặc định 30; tính cả số thẻ đã ôn trong ngày. Ôn tự do không giới hạn.
- **Múi giờ** tính ngày học: mặc định là múi giờ lúc đăng ký. Đổi múi giờ không làm mất tiến độ: bài đang dở vẫn là bài hôm nay,
  bước đã xong và vị trí đang học giữ nguyên; ngày học không bao giờ lùi, nên đổi sang múi giờ còn "hôm qua" không có thêm bài mới.
- Cài đặt nằm trong `users.settings {theme, dailyReviewLimit, timezone}`; trường `timezone` cũ của tài khoản được chuyển vào đó tự
  động khi khởi động.
- **Xuất dữ liệu** (nhóm Dữ liệu cuối trang, `GET /api/export`): tải file `luna-export-YYYYMMDD.json` gồm tài khoản (email, vai trò,
  ngày tạo), cài đặt, sổ từ và lịch ôn, lịch sử ôn, mục tiêu, tiến độ từng bài, ngày học, kết quả chép chính tả, câu trả lời câu hỏi hiểu bài và nội dung các bài
  đã học (bài đã bị xoá ghi `deleted`). Chỉ có dữ liệu của tài khoản đang đăng nhập; không có mật khẩu hay phiên đăng nhập. Chưa
  nhập lại được từ file này.

## Sổ từ và ôn tập

- **Sổ từ** (header → Sổ từ, `/vocabulary`): thẻ nhóm theo ngày lưu ("Hôm nay", "Hôm qua", ngày cụ thể) theo múi giờ của
  tài khoản; tìm theo từ, lọc theo bài hoặc "Thẻ tự thêm"; thêm thẻ, sửa nghĩa/IPA/câu ví dụ (lịch ôn giữ nguyên), xoá thẻ
  (xoá cả lịch sử ôn).
- **Ôn tập** (`/vocabulary/review`): các thẻ đến hạn, kiểu *Xem từ đoán nghĩa* (Space lật thẻ) hoặc *Nghe rồi gõ* (không phân
  biệt hoa thường); chọn Again / Hard / Good / Easy (phím 1–4), ngày ôn tiếp theo tính bằng FSRS
  ([go-fsrs](https://github.com/open-spaced-repetition/go-fsrs), tham số mặc định). Thẻ mới đến hạn lần đầu lúc 0 giờ hôm sau.
- **Mục Từ vựng** ở cuối bước Đọc: các từ đã chú thích của bài, nút Lưu / Lưu tất cả (không tạo thẻ trùng), không gọi AI.

Thử ôn ngay một thẻ vừa lưu (mặc định phải chờ tới hôm sau):

```bash
docker compose -f deploy/docker-compose.yml exec mongo mongosh luna --quiet \
  --eval 'db.cards.updateMany({}, {$set: {due: new Date(Date.now() - 60000)}})'
```

## Sao lưu và khôi phục

Service `backup` (chạy cùng `docker compose … up`) tự sao lưu toàn bộ database mỗi ngày lúc **03:00** (giờ `BACKUP_TZ`, mặc định
Việt Nam) vào `deploy/backups/luna-YYYYMMDD.archive.gz` và chỉ giữ **7 bản** mới nhất (bản thứ 8 làm bản cũ nhất bị xoá). Máy chủ
tắt lúc 03:00 thì khi bật lại sẽ sao lưu bù. Sao lưu lỗi chỉ ghi log `ERROR`, không xoá bản nào, app vẫn chạy bình thường. File âm
thanh (volume `audio-data`, không bị ảnh hưởng khi khôi phục) và từ điển không được sao lưu: từ điển tải lại bằng
`deploy/fetch-dictionary.sh`, audio tạo lại bằng nút **Chạy lại audio** ở trang chi tiết bài. Nên chép `deploy/backups/` sang máy/ổ
khác định kỳ.

```bash
docker compose -f deploy/docker-compose.yml logs backup           # lần sao lưu gần nhất, lần tới, lỗi nếu có
ls deploy/backups                                                 # danh sách bản sao lưu
docker compose -f deploy/docker-compose.yml run --rm -e BACKUP_NOW=1 -e BACKUP_ONCE=1 backup   # sao lưu ngay một lần
```

**Khôi phục** (thay toàn bộ dữ liệu hiện tại bằng dữ liệu của bản sao lưu, kể cả tài khoản và phiên đăng nhập — mọi thay đổi sau thời
điểm sao lưu sẽ mất):

```bash
docker compose -f deploy/docker-compose.yml stop backend
docker compose -f deploy/docker-compose.yml run --rm --entrypoint bash backup /backup/restore.sh luna-20260930.archive.gz
docker compose -f deploy/docker-compose.yml start backend
```

- Không truyền tên file hoặc tên sai: lệnh in danh sách bản hiện có và dừng. File hỏng: lệnh báo lỗi và **không** đụng vào database.
- Xong, lệnh in số document của từng collection; đăng nhập lại để kiểm tra sổ từ và tiến độ.
- Chạy trong Git Bash trên Windows: thêm `MSYS_NO_PATHCONV=1` trước lệnh (nếu không, `/backup/restore.sh` bị đổi thành đường dẫn
  Windows).

## Cấu hình

Không cần tạo file cấu hình để chạy: mọi biến đều có giá trị mặc định trong `deploy/docker-compose.yml`.
Muốn đổi thì chép file mẫu rồi sửa:

```bash
cp deploy/.env.example deploy/.env
```

| Biến | Dùng ở | Bắt buộc | Mặc định | Ý nghĩa |
|---|---|---|---|---|
| `MONGO_URI` | backend | Có | `mongodb://mongo:27017` (compose) | Chuỗi kết nối MongoDB |
| `HTTP_ADDR` | backend | Không | `:8080` | Địa chỉ backend lắng nghe |
| `MONGO_DATABASE` | backend | Không | `luna` | Tên database |
| `LOG_LEVEL` | backend | Không | `info` | `debug`, `info`, `warn`, `error` |
| `COOKIE_SECURE` | backend | Không | `false` | `true` khi chạy qua HTTPS (thêm cờ `Secure` cho cookie đăng nhập) |
| `WEB_PORT` | compose | Không | `8000` | Cổng mở app trên máy |
| `TTS_URL` | backend | Không | `http://localhost:8880` (compose: `http://kokoro:8880`) | Địa chỉ Kokoro |
| `TTS_VOICE` | backend | Không | `af_heart` | Giọng đọc |
| `AUDIO_DIR` | backend | Không | `./data/audio` (compose: `/data/audio`) | Thư mục lưu mp3 |
| `AI_PROVIDER` | backend | Không | `gemini` | `gemini` hoặc `none` |
| `GEMINI_API_KEY` | backend | Không | rỗng | Khoá Gemini (bí mật, chỉ để trong `.env`) |
| `GEMINI_MODEL` | backend | Không | `gemini-3.5-flash-lite` | Model dùng để chú thích và sinh bài |
| `DICTIONARY_PATH` | backend | Không | `./data/dictionary/dictionary.db` (compose: `/data/dictionary/dictionary.db`) | File từ điển SQLite |
| `BACKUP_TZ` | backup | Không | `Asia/Ho_Chi_Minh` | Múi giờ của giờ sao lưu và ngày trong tên bản sao lưu |
| `BACKUP_TIME` | backup | Không | `03:00` | Giờ sao lưu mỗi ngày (HH:MM) |
| `BACKUP_KEEP` | backup | Không | `7` | Số bản sao lưu giữ lại |

Thiếu `MONGO_URI`, hoặc `LOG_LEVEL` / `COOKIE_SECURE` / `AI_PROVIDER` / `TTS_URL` sai thì backend in lỗi nêu tên biến và dừng với mã 1.
Mọi file `.env` đều bị git bỏ qua; không commit bí mật vào repo.

## Phát triển

Cần thêm Go 1.25 và Node 22. Chạy từng phần riêng:

**1. Cơ sở dữ liệu và TTS** (MongoDB, Kokoro trong Docker, chỉ mở cổng trên localhost):

```bash
docker compose -f deploy/docker-compose.yml up -d mongo kokoro
```

**2. Backend** (`http://localhost:8080`). Lần đầu, chép file cấu hình mẫu cho môi trường phát triển:

```bash
cd backend
cp .env.example .env      # PowerShell: Copy-Item .env.example .env
go run ./cmd/api
```

Khi chạy local, backend tự đọc `backend/.env` trong thư mục đang đứng (nên chạy lệnh từ `backend/`).
Biến môi trường đặt thật (terminal, Docker) luôn được ưu tiên hơn giá trị trong file.
`backend/.env` bị git bỏ qua; Docker không dùng file này (xem `deploy/.env.example`).

**3. Frontend** (`http://localhost:4200`, tự tải lại khi sửa code; `/api` được chuyển tới backend qua
`frontend/proxy.conf.json`):

```bash
cd frontend
npm ci
npm start
```

> Nếu `npm ci`/`npm install` báo `Cannot read properties of null (reading 'edgesOut')` (lỗi của npm 10.9),
> chạy bằng npm 11: `npx -y npm@11 install`.

## Test và lint

Chạy được không cần mạng sau khi đã tải phụ thuộc một lần (`npm ci`, `go mod download`, image golangci-lint).

**Frontend**

```bash
cd frontend
npm test -- --watch=false
npm run lint
```

**Backend** (golangci-lint chạy trong Docker, không cần cài):

| Việc | `make` (Linux/macOS/WSL) | Lệnh tương đương (PowerShell) |
|---|---|---|
| Chạy | `make run` | `go run ./cmd/api` |
| Test | `make test` | `go test ./...` |
| Lint | `make lint` | xem bên dưới |
| Build | `make build` | `$env:CGO_ENABLED=0; go build -trimpath -o bin/api.exe ./cmd/api` |

```powershell
cd backend
docker run --rm -v "${PWD}:/app" -w /app -v luna-go-mod:/go/pkg/mod -v luna-golangci-cache:/root/.cache golangci/golangci-lint:v2.14.0 golangci-lint run ./...
```

## Cấu trúc thư mục

```text
frontend/   Angular 21 (standalone, signals, zoneless): core/, shared/, features/
backend/    Go 1.25, net/http: cmd/api, internal/{auth,lesson,topic,job,tts,ai,dictionary,vocab,progress,health,platform,storage}
deploy/     docker-compose.yml, .env.example, fetch-dictionary.sh/.ps1, data/ (dữ liệu tải về)
docs/       Tài liệu sản phẩm (nguồn sự thật)
specs/      Spec, plan, tasks của từng tính năng
```

Chi tiết: [`docs/architecture.md`](docs/architecture.md), [`docs/design-system.md`](docs/design-system.md).
#   l u n a  
 