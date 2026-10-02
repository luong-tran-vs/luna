# Luna

Web app học tiếng Anh cá nhân: luyện 4 kỹ năng Nghe, Nói, Đọc, Viết quanh một bài học mỗi ngày.

Tài liệu sản phẩm nằm trong [`docs/`](docs/); spec từng tính năng nằm trong [`specs/`](specs/).

## Chạy nhanh

Chỉ cần cài [Docker](https://docs.docker.com/get-docker/) (kèm Docker Compose v2). Từ thư mục gốc repo:

```bash
docker compose -f deploy/docker-compose.yml up --build
```

Mở <http://localhost:8000>. Lần đầu, bấm **Đăng ký** để tạo tài khoản: **tài khoản đầu tiên là quản trị viên**,
các tài khoản sau là người học. App chia hai khu riêng, mỗi khu một menu:

- **Khu học** (`/`): 4 tab Trang chủ, Khóa học, Ôn tập, Tài khoản (ở đáy màn hình điện thoại, trên đầu trang khi màn hình
  rộng). Bài viết, Sổ từ, Thống kê, Cài đặt, chế độ tối và Đăng xuất nằm trong **Tài khoản**. Chỉ dành cho người học.
- **Khu quản trị** (`/admin`): Bài học, Chủ đề, Lộ trình. Chỉ dành cho quản trị viên; quản trị viên đăng nhập vào thẳng
  đây và mở trang học nào cũng được chuyển về `/admin`. Muốn tự học thì dùng một tài khoản người học riêng.

Nếu máy chủ hoặc cơ sở dữ liệu không chạy, chân trang báo lỗi kết nối (bình thường không hiện gì).

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

Đăng nhập bằng tài khoản quản trị viên (vào thẳng khu **Quản trị**, `/admin`):

1. **Chủ đề** (`/admin/topics`): tạo chủ đề theo trình độ, ví dụ "A1 · Gia đình". Tên không trùng trong cùng trình độ;
   chủ đề còn bài thì không xoá được.
2. **Thêm bài**: dán văn bản tiếng Anh và chọn chủ đề; trình độ của bài lấy theo chủ đề.
3. **Lộ trình** (`/admin/roadmap`): chọn chủ đề, thêm bài của chủ đề, kéo thả hoặc nút Lên/Xuống để xếp thứ tự. Mỗi chủ đề
   một lộ trình; chủ đề còn dưới 3 bài chưa học được cảnh báo. Đổi chủ đề của một bài đang ở lộ trình thì bài chuyển sang
   cuối lộ trình của chủ đề mới.

Dữ liệu từ trước F14 (chủ đề gõ tự do, một lộ trình chung) được chuyển tự động khi backend khởi động lần đầu: chủ đề tạo từ
các cặp trình độ và chủ đề của bài, bài chưa có chủ đề vào "Chung" cùng trình độ, lộ trình chung chia theo chủ đề giữ thứ
tự. Chỉ chạy một lần (đánh dấu trong collection `migrations`).

Sau khi lưu bài, hệ thống tự làm nền **chú thích từ vựng** (phần nghe không cần tạo trước: app đọc bằng giọng
đọc của trình duyệt):

- **Chú thích từ vựng** bằng Gemini (gói miễn phí). Cần khoá API: tạo tại
  <https://aistudio.google.com/apikey>, rồi ghi vào `deploy/.env`:

  ```bash
  GEMINI_API_KEY=khoá-của-bạn
  ```

  và chạy lại `docker compose -f deploy/docker-compose.yml up -d`. Không có khoá thì bài vẫn lưu và nghe được,
  chú thích báo "AI chưa được cấu hình"; thêm khoá xong bấm **Chạy lại chú thích**. Tắt hẳn AI: `AI_PROVIDER=none`.

### Sinh bài bằng AI

Ở **Lộ trình**, chọn một chủ đề rồi bấm **Sinh bài bằng AI**. Trình độ và chủ đề lấy theo lộ trình đang mở; chọn:

- **Số bài**: 1–10 (mặc định 3).
- **Độ dài** mỗi bài: 50–800 từ, mặc định theo trình độ (A1 120, A2 160, B1 220, B2 300, C1 và C2 400). Bài lệch quá 20%
  vẫn được giữ, kèm cảnh báo "ngắn hơn/dài hơn yêu cầu" (từ 2026-10-02; trước đó bị loại).
- **Dạng bài**: bài đọc, hoặc hội thoại (mỗi lượt nói một dòng `Tên: câu nói`).
- **Ý chính**: không bắt buộc, tối đa 500 ký tự.

Mỗi lượt tốn **một** request AI cho cả lượt và có thể mất đến hai phút (10 bài dài). Các tiêu đề đã có trong chủ đề được gửi kèm để
AI tránh lặp.

Kết quả là các **bản nháp**, chỉ có trên trang đang mở và không được lưu vào cơ sở dữ liệu:

- Sửa tiêu đề và nội dung từng bản, bấm **Lưu** hoặc **Bỏ**, hoặc **Lưu tất cả**.
- Bài được lưu có nguồn "AI sinh", giấy phép "Nội dung do AI tạo". Bài được thêm vào cuối lộ trình rồi được chú
  thích như bài tạo tay.
- Rời trang, tải lại hoặc đổi chủ đề khi còn bản nháp thì được hỏi lại, vì bản nháp sẽ mất.
- Bản AI trả về bị loại khi trùng tiêu đề (trong lượt hoặc với bài đã có), thiếu tiêu đề/nội dung, hoặc quá dài so với
  giới hạn bài; thông báo sau khi sinh ghi số bản bị loại theo từng lý do.

AI chưa cấu hình, khoá sai, hết lượt hoặc trả về nội dung không dùng được thì trang báo rõ lỗi. Bản nháp và các lựa chọn
đã nhập vẫn giữ nguyên để thử lại.

Request sinh bài dài hơn mọi request khác: nginx (`frontend/nginx.conf`) và backend chỉ cho riêng route
`/api/admin/topics/{id}/generate` chờ tới 135 giây (AI tối đa 120 giây), các route còn lại vẫn 15 giây.

### Từ vựng theo chủ đề

Mỗi chủ đề có một danh sách từ vựng tiếng Anh cốt lõi, tối đa 100 từ hoặc cụm (ví dụ A1 · Gia đình: family, parents…).

- **Dữ liệu ban đầu:** khi khởi động, chủ đề chưa từng được xét mà tên trùng một chủ đề trong
  `backend/internal/topic/seed/topic-words.json` (42 chủ đề, 1.257 từ; chỉ lấy từ tiếng Anh, tham khảo danh sách của Langmaster)
  nhận danh sách đó; chủ đề khác nhận danh sách rỗng. Mỗi chủ đề chỉ được xét một lần (cờ `wordsSeeded`), nên danh sách đã sửa,
  kể cả xoá hết, không bao giờ bị nạp đè. Log: `topic words seeded`.
- **Trang Chủ đề** hiện "Từ vựng: đã dùng X/Y"; **Từ vựng** mở trang sửa danh sách: thêm nhiều từ một lần (mỗi dòng hoặc dấu
  phẩy), xoá từ, lỗi hiện dưới từng từ (trùng, quá 40 ký tự, ký tự ngoài chữ cái tiếng Anh, khoảng trắng, `-`, `'`, `/`).
- **Đã dùng:** từ có trong nội dung ít nhất một bài của chủ đề (kể cả bài ngoài lộ trình), nguyên từ hoặc nguyên cụm, không phân
  biệt hoa/thường, tính cả dạng số nhiều, -ed, -ing, bất quy tắc thường gặp và dạng gốc trong chú thích của bài.
- **Sinh bài bằng AI:** chủ đề có từ vựng thì hộp thoại có "Từ mục tiêu mỗi bài" (mặc định 8/10/12 theo trình độ, 0 để sinh như
  cũ). App chia sẵn nhóm từ cho từng bài (chưa dùng trước, rồi dùng ít nhất; không trùng giữa các bài khi đủ từ); quản trị viên bỏ
  hoặc thêm từ rồi Sinh. Vẫn 1 request AI. Bản nháp hiện "Dùng a/b từ mục tiêu" và "Còn thiếu: …", không bị loại vì thiếu từ.
  Chủ đề thiếu từ chưa dùng cho số bài đã chọn thì hộp thoại báo "Chủ đề thiếu N từ" và có nút **Bổ sung bằng AI**: AI gợi ý
  tối đa 50 từ mới đúng chủ đề và trình độ (1 request riêng, `POST /api/admin/topics/{id}/words/suggest`), từ hợp lệ và chưa có
  được thêm vào danh sách của chủ đề rồi app chia nhóm lại. Danh sách tối đa 300 từ.
- **Chú thích:** từ của chủ đề có trong bài được gửi kèm request chú thích (vẫn 1 request); từ AI bỏ sót được thêm với nghĩa từ từ
  điển (từ điển không có thì bỏ qua). Người học không thấy danh sách từ của chủ đề.

## Bước Đọc và từ điển

Bước Đọc tra nghĩa bằng chú thích AI của bài, rồi bằng từ điển Anh–Việt offline
([minhqnd/dictionary](https://github.com/minhqnd/dictionary), MIT, ~180 MB). Tải từ điển một lần trước khi chạy:

```bash
./deploy/fetch-dictionary.sh        # hoặc PowerShell: .\deploy\fetch-dictionary.ps1
```

Script lưu vào `deploy/data/dictionary/dictionary.db` (git bỏ qua) và kiểm tra SHA-256. Thiếu file thì app vẫn chạy,
tra từ chỉ dùng chú thích của bài.

Mở bước Đọc (tài khoản người học): trang bài đang học → bước Đọc (sau phần luyện tập), hoặc `/lessons/<id>/read`. Chạm một từ để tra,
bôi đen nhiều từ để tra cụm, bấm ▶ để nghe, **Lưu vào sổ từ** để lưu kèm câu.

### Hỏi AI

Khi từ điển không có nghĩa, hoặc chỉ có nghĩa chung chung, popup hiện nút **Hỏi AI**. AI giải thích từ hoặc cụm đó theo
đúng câu đang đọc: dạng gốc, nghĩa tiếng Việt và một ghi chú ngắn. Lưu vào sổ từ thì thẻ dùng nghĩa của AI.

- Kết quả được lưu trong collection `ai_lookups`, dùng chung cho mọi người học, theo khoá (bài, phiên bản bài, câu, từ).
  Lần tra sau trong cùng câu sẽ ra luôn kết quả này (trước cả từ điển), không gọi AI nữa.
- Nhiều người hỏi cùng một khoá cùng lúc thì AI chỉ được gọi **một lần**.
- Sửa nội dung bài (phiên bản mới) thì kết quả cũ không được dùng nữa.
- AI lỗi thì popup báo lỗi ngay dưới nút; nghĩa từ điển hoặc ô tự nhập nghĩa vẫn dùng được. Kết quả lỗi không được lưu.
  - `503 ai_not_configured`: chưa cấu hình AI hoặc khoá sai.
  - `429 ai_quota`: hết lượt AI.
  - `502 ai_failed`: AI lỗi, quá 12 giây, hoặc không trả nghĩa.

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
  - Trả lời hết (kể cả khi sai) thì bước Đọc hoàn thành. Máy chủ kiểm tra việc này: `POST /api/lessons/{id}/steps/read/complete` trả 409
    `read_incomplete` khi còn câu chưa trả lời.
- Bài chưa có câu hỏi (AI lỗi hoặc bài cũ) vẫn dùng nút "Đã đọc xong".
- Xem lại bài từ trang **Bài học** thì thấy các câu trả lời cũ.

Câu trả lời được lưu trong collection `reading_answers`, mỗi câu một lần cho mỗi phiên bản bộ câu hỏi (`quizVersion`). Khi
quản trị viên sửa câu hỏi, chạy lại chú thích hoặc sửa nội dung bài, bộ câu hỏi mới bắt đầu lại từ đầu; câu trả lời cũ vẫn
tính vào thống kê.

Quản trị viên sửa câu hỏi (thêm, xoá, đổi đáp án), ngữ pháp và đề viết ở trang chi tiết bài; ví dụ ngữ pháp phải chép đúng
từ bài. **Chạy lại chú thích** dùng được cả khi chú thích đã xong, nên bài có từ trước F15 bấm nút này là có thêm câu hỏi.
Nếu bài có chú thích từ hoặc câu hỏi đã sửa tay, trang hỏi xác nhận trước khi thay.

## Trang chi tiết bài và luyện tập từ vựng

Chạm một bài (bài đang học hoặc bài đã học) ở trang **Bài học** để mở `/lessons/<id>`. Bài sắp tới vẫn khoá: API trả 403
`lesson_locked`. Trang gồm thanh trên (đóng, tiến độ theo số bước, streak), "Bài N" theo thứ tự trong lộ trình chủ đề, mục tiêu,
mức độ, và hai tab **Bài học** / **Bài đọc**. Tab Bài học có tối đa 4 bước luyện tập (chỉ những bước có nội dung), chuyển bằng
nút **Tiếp theo** / **← Bước trước** cố định cuối màn hình:

1. **Từ vựng quan trọng**: từ, phiên âm, nghĩa, nút nghe từ, câu ví dụ nghe được.
2. **Hội thoại mẫu**: lời ẩn sẵn để nghe trước ("Hiện lời" từng lượt hoặc tất cả); mỗi lượt có dạng sóng tô theo tiến độ phát
   (hình minh hoạ dựng từ câu, không phải sóng thật của audio); nghe cả đoạn (0.75×/1×/1.25×) hoặc từng lượt, bật/tắt nghĩa.
3. **Điền vào ô trống**: hội thoại có tối đa 5 ô trống ở từ vựng của bài; gõ từ vào từng ô (Enter sang ô kế tiếp) hoặc bấm từ
   trong ngân hàng từ để điền vào ô đang chọn, **Kiểm tra** (không phân biệt hoa thường).
4. **Dịch câu sang tiếng Anh**: ghép câu bằng các ô từ (có từ gây nhiễu), **Làm lại**, **Kiểm tra** từng câu.

Với **bài đang học**, sau các bước luyện tập là **Đọc → Nghe → Viết** ngay trên trang này (xem "Học lần lượt từng bài"); xong
bài thì có nút **Sang bài tiếp theo**. Với bài khác, cuối cùng là tổng kết (số ô điền đúng, số câu dịch đúng) và **Làm lại**. Phần luyện tập là tự chọn: chấm ngay trên trình
duyệt, kết quả không gửi lên máy chủ, không ảnh hưởng các bước của bài, tiến độ, streak hay thống kê; không có điểm XP.

Nội dung luyện tập (`GET /api/lessons/{id}/practice`):

- Chú thích bài xong thành công thì job nền `practice` gọi AI **đúng 1 request**: mục tiêu bài, câu ví dụ cho mỗi từ, hội
  thoại 2 người, mẹo ngữ pháp, 3–5 câu dịch. Phần hỏng bị bỏ (câu ví dụ không chứa từ, hội thoại dưới 4 lượt, câu dịch không
  dùng từ vựng…); không còn gì dùng được thì báo lỗi, không gọi lại AI. Ô trống bước 3 do máy chủ tính, không cần AI.
- Không tạo file audio: mọi chỗ nghe trong app (bước Đọc, Nghe, sổ từ, ôn tập, trang chi tiết bài, trang
  quản trị) đọc bằng **giọng đọc của trình duyệt**; trình duyệt không có giọng đọc thì nút nghe bị ẩn.
- Trang chỉ hiện các bước có nội dung (số bước và thanh tiến độ theo số bước thật); bài chưa có từ vựng lẫn phần luyện tập
  thì không có tab Bài học, chỉ hiện bài đọc. AI lỗi thì chỉ phần luyện tập trống; chú thích, câu hỏi và các bước của bài
  vẫn bình thường.
- Sửa nội dung bài hoặc chạy lại chú thích thì phần luyện tập cũ bị bỏ và được sinh lại. Bài có từ trước F17 không tự sinh:
  quản trị viên bấm **Tạo lại phần luyện tập** ở trang chi tiết bài (mục "Phần luyện tập": trạng thái, lỗi, nội dung).

## Bước Nghe (chép chính tả)

Mở từ trang bài đang học, sau bước Đọc (hoặc `/lessons/<id>/listen`). Câu được đọc bằng **giọng đọc của trình duyệt** (Web Speech API, không cần
audio tạo sẵn), kèm dạng sóng minh hoạ chạy theo câu đang đọc. Nghe từng câu (trước/sau, nghe lại, tốc độ 0.5x–1.25x, chữ ẩn
mặc định), gõ lại câu rồi **Kiểm tra** (hoặc Enter). Trình duyệt không có giọng đọc thì trang báo rõ. Cách so sánh:

- từ sai bị gạch ngang, kèm từ đúng; từ thừa bị gạch ngang; từ thiếu được gạch chân chấm;
- so sánh không phân biệt hoa thường và dấu câu; `don't` khác `dont`, `9.30` khác `9 30`.

Kết quả mới nhất của mỗi câu được lưu theo người học (`POST /api/lessons/{id}/dictation`); tỷ lệ đúng = số từ đúng / tổng
số từ (`GET /api/lessons/{id}/dictation/summary`). Kiểm tra hết các câu là hoàn thành bước Nghe. Sửa nội dung bài thì kết
quả cũ không còn được tính.

## Học lần lượt từng bài

Cập nhật 2026-10-02: bỏ trang "Hôm nay" và giới hạn một bài mỗi ngày, bỏ bước Ôn khỏi bài.

- **Mục tiêu** (`/goal`): chọn trình độ rồi một chủ đề của trình độ đó; mục tiêu là học hết lộ trình của chủ đề. Tiến độ từng
  chủ đề lưu riêng; đổi chủ đề có hiệu lực ngay.
- **Bài đang học** là bài đầu tiên chưa xong trong lộ trình. Học ngay ở trang bài `/lessons/<id>` (trang chủ → **Bắt đầu học /
  Tiếp tục: <bước>**, hoặc trang **Bài học**): các phần luyện tập có nội dung (F17), rồi **Đọc** → **Nghe** → **Viết** (F8, tuỳ
  chọn), chung một thanh tiến độ. Thoát ra vào lại thì tiếp tục đúng bước và đúng câu. Nút **← Bước trước** mở lại bước đã qua
  (bước Đọc, Nghe đã xong chỉ để xem, không đổi tiến độ).
- Xong bài (nộp bài viết hoặc bấm **Bỏ qua** ở bước Viết): mục tiêu +1, ngày đó tính vào chuỗi ngày học, và nút **Sang bài tiếp
  theo** mở ngay bài kế tiếp — học bao nhiêu bài một ngày cũng được. Nghỉ một ngày trọn thì chuỗi về 0. Bài đã xong trước khi có
  bước Viết vẫn tính là xong.
- **Bài học** (header → **Bài học**, `/lessons`): các bài đã học (đọc/nghe lại, không đổi tiến độ), bài đang học, các bài sắp tới
  bị khoá — người học không mở được bài chưa tới lượt. Không có ô trống: bài nào có thì hiện bài đó.
- Ôn thẻ đến hạn ở **Ôn tập** (`/vocabulary/review`), không còn là bước của bài.
- API: `GET /api/lessons/{id}/study` (trạng thái các bước, bài kế tiếp), `POST /api/lessons/{id}/steps/{step}/complete`,
  `POST /api/lessons/{id}/steps/write/skip`, `PUT /api/lessons/{id}/position`. Bài không phải bài đang học trả 409
  `not_current_lesson`.

## Bước Viết và bài viết

Bước **Viết** (F8) đến sau bước Nghe và là **tuỳ chọn**:

- Bước Viết hiện đề viết của bài (F15), hoặc "Tóm tắt bài bằng 3–5 câu." khi bài chưa có đề.
- Kèm theo là độ dài gợi ý theo trình độ (A1 30–60 từ, A2 50–80, B1 80–120, B2 120–180, C1 và C2 150–250).
- Nháp tự lưu 1 giây sau khi ngừng gõ (`PUT /api/lessons/{id}/writing`); thoát ra vào lại hay mở máy khác vẫn còn.
- **Nộp** (5–400 từ) hoàn thành bước Viết và bài ngay, không chờ AI. Đã nộp thì không sửa hay nộp lại được.
- **Bỏ qua** hoàn thành bài mà không viết (`POST /api/lessons/{id}/steps/write/skip`): không có bài viết, không gọi AI.
  `POST /api/lessons/{id}/steps/write/complete` vẫn trả 409 `write_incomplete` khi chưa nộp, để không lỡ tay bỏ qua.
- Chỉ bài đang học, đang ở bước Viết mới viết được; mở lại bài cũ (trang **Bài học** → **Bài viết**) chỉ xem.

AI chấm ở nền (job `grade`, thử lại tối đa 3 lần, 1 request AI mỗi lần chấm):

- Mỗi bài có điểm 1–5 và nhận xét tiếng Việt cho 4 tiêu chí: hoàn thành yêu cầu, ngữ pháp, từ vựng, mạch lạc.
- Kèm theo là nhận xét chung, bản đã sửa và phần so sánh theo từ: gạch chân là chỗ thêm, gạch ngang là chỗ bớt, có nhãn cho trình
  đọc màn hình.
- Có kết quả thì header hiện số trên liên kết **Bài viết** và hiện một thông báo ngắn ở cuối trang. Trình duyệt chỉ hỏi máy chủ
  mỗi 30 giây khi còn bài đang chấm.
- Chấm lỗi (AI tắt, hết lượt, kết quả hỏng) thì bài viết vẫn còn, trang bài viết ghi lý do và có nút **Chấm lại**.

Trang **Bài viết** (`/writings`) liệt kê bài đã nộp, mới nhất trước, kèm điểm trung bình hoặc trạng thái. Bài viết lưu trong
collection `writings`, mỗi người một bài cho mỗi bài học.

## Màn hình chính và thống kê

- **Màn hình chính** (`/`): chuỗi ngày học, thanh mục tiêu ("A1 · Gia đình", số bài đã xong / tổng số bài) kèm thanh Đọc,
  Nghe và Viết (số bài của lộ trình hiện tại đã xong bước đó), bài đang học với thanh bước, nút **Bắt đầu học** (chưa xong bước nào) hoặc
  **Tiếp tục: <bước>** mở trang bài, và số thẻ cần ôn tới hết ngày mai (gồm cả thẻ còn nợ). Chưa có mục tiêu → **Chọn
  chủ đề**; hết bài → **Ôn tự do** / **Chọn chủ đề khác**. Số liệu tải lại mỗi lần quay về trang; mở trang không làm bài "bắt đầu".
- **Thống kê** (màn hình chính → **Xem thống kê**, `/stats`): số từ đã học, số câu đã chép chính tả và tỷ lệ đúng, số bài đã
  xong bước Đọc / Nghe và số bài hoàn thành, tính trên mọi chủ đề. Thẻ **Hiểu bài** (F15) là tỷ lệ trả lời đúng câu hỏi hiểu bài
  trên mọi câu đã trả lời (hiện "—" khi chưa trả lời câu nào). Thẻ **Bài viết** (F8) có số bài viết đã nộp và điểm trung bình của
  các bài đã chấm.
- **Trạng thái kết nối** (F0) nằm ở chân trang và chỉ hiện khi không kết nối được máy chủ hoặc cơ sở dữ liệu (kiểm tra lại sau mỗi
  lần chuyển trang).

## Cài đặt

Thanh trên → **Cài đặt** (`/settings`); cài đặt lưu theo tài khoản, đăng nhập máy khác vẫn giữ.

- **Giao diện**: Sáng, Tối, Theo hệ thống (mặc định); đổi là có hiệu lực ngay. Nút giao diện nhanh ở thanh trên cũng lưu vào tài
  khoản. Trước khi đăng nhập app dùng lựa chọn lưu trên trình duyệt; sau khi đăng nhập lựa chọn của tài khoản được dùng.
- **Số thẻ ôn mỗi ngày**: 5–200, mặc định 30. Từ 2026-10-02 bài không còn bước Ôn nên giá trị này chưa được dùng; Ôn tập không giới hạn.
- **Múi giờ** tính ngày cho chuỗi ngày học: mặc định là múi giờ lúc đăng ký. Đổi múi giờ không làm mất tiến độ: bài đang dở, bước
  đã xong và vị trí đang học giữ nguyên.
- Cài đặt nằm trong `users.settings {theme, dailyReviewLimit, timezone}`; trường `timezone` cũ của tài khoản được chuyển vào đó tự
  động khi khởi động.
- **Xuất dữ liệu** (nhóm Dữ liệu cuối trang, `GET /api/export`): tải file `luna-export-YYYYMMDD.json` gồm tài khoản (email, vai trò,
  ngày tạo), cài đặt, sổ từ và lịch ôn, lịch sử ôn, mục tiêu, tiến độ từng bài, ngày học, kết quả chép chính tả,
  câu trả lời câu hỏi hiểu bài, bài viết (cả nháp) kèm nhận xét và nội dung các bài
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
tắt lúc 03:00 thì khi bật lại sẽ sao lưu bù. Sao lưu lỗi chỉ ghi log `ERROR`, không xoá bản nào, app vẫn chạy bình thường. Từ điển
không được sao lưu: tải lại bằng `deploy/fetch-dictionary.sh`. Nên chép `deploy/backups/` sang máy/ổ
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
| `AI_PROVIDER` | backend | Không | `gemini` | `gemini` hoặc `none` |
| `GEMINI_API_KEY` | backend | Không | rỗng | Khoá Gemini (bí mật, chỉ để trong `.env`) |
| `GEMINI_MODEL` | backend | Không | `gemini-3.5-flash-lite` | Model dùng để chú thích và sinh bài |
| `DICTIONARY_PATH` | backend | Không | `./data/dictionary/dictionary.db` (compose: `/data/dictionary/dictionary.db`) | File từ điển SQLite |
| `BACKUP_TZ` | backup | Không | `Asia/Ho_Chi_Minh` | Múi giờ của giờ sao lưu và ngày trong tên bản sao lưu |
| `BACKUP_TIME` | backup | Không | `03:00` | Giờ sao lưu mỗi ngày (HH:MM) |
| `BACKUP_KEEP` | backup | Không | `7` | Số bản sao lưu giữ lại |

Thiếu `MONGO_URI`, hoặc `LOG_LEVEL` / `COOKIE_SECURE` / `AI_PROVIDER` sai thì backend in lỗi nêu tên biến và dừng với mã 1.
Mọi file `.env` đều bị git bỏ qua; không commit bí mật vào repo.

## Phát triển

Cần thêm Go 1.25 và Node 22. Chạy từng phần riêng:

**1. Cơ sở dữ liệu** (MongoDB trong Docker, chỉ mở cổng trên localhost):

```bash
docker compose -f deploy/docker-compose.yml up -d mongo
```

Dùng MongoDB Atlas (cloud) thay cho MongoDB trong Docker: đặt `MONGO_URI=mongodb+srv://...` trong `backend/.env`
(chuỗi kết nối lấy ở Atlas → **Connect** → **Drivers**; thêm IP máy này ở **Network Access**), không cần chạy
container nào. Backend tự tạo index trên DB mới.

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
backend/    Go 1.25, net/http: cmd/api, internal/{auth,lesson,topic,job,ai,dictionary,vocab,progress,health,platform,storage,wordmatch}
deploy/     docker-compose.yml, .env.example, fetch-dictionary.sh/.ps1, data/ (dữ liệu tải về)
docs/       Tài liệu sản phẩm (nguồn sự thật)
specs/      Spec, plan, tasks của từng tính năng
```

Chi tiết: [`docs/architecture.md`](docs/architecture.md), [`docs/design-system.md`](docs/design-system.md).
#   l u n a  
 