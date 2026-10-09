# Luna

Web app học tiếng Anh cá nhân: luyện 4 kỹ năng Nghe, Nói, Đọc, Viết quanh một bài học mỗi ngày.

Tài liệu sản phẩm nằm trong [`docs/`](docs/); spec từng tính năng nằm trong [`specs/`](specs/).

## Chạy nhanh

Chạy backend và frontend như ở mục [Phát triển](#phát-triển), rồi mở <http://localhost:4200>. Lần đầu, bấm **Đăng ký** để tạo tài khoản: **tài khoản đầu tiên là quản trị viên**,
các tài khoản sau là người học. App chia hai khu riêng, mỗi khu một menu:

- **Khu học** (`/`): 4 tab Trang chủ, Khóa học, Ôn tập, Tài khoản (ở đáy màn hình điện thoại, trên đầu trang khi màn hình
  rộng). Bài viết, Sổ từ, Thống kê, Cài đặt, chế độ tối và Đăng xuất nằm trong **Tài khoản**. Chỉ dành cho người học.
- **Khu quản trị** (`/admin`): Bài học, Chủ đề, Lộ trình. Chỉ dành cho quản trị viên; quản trị viên đăng nhập vào thẳng
  đây và mở trang học nào cũng được chuyển về `/admin`. Muốn tự học thì dùng một tài khoản người học riêng.

Nếu máy chủ hoặc cơ sở dữ liệu không chạy, chân trang báo lỗi kết nối (bình thường không hiện gì).

Muốn tạo lại quản trị viên từ đầu: xoá database `luna` (hoặc trỏ `MONGO_DATABASE` sang database mới), tài khoản đăng ký
đầu tiên sẽ là quản trị viên. Xem danh sách tài khoản:

```bash
mongosh "$MONGO_URI" --quiet --eval 'db.getSiblingDB("luna").users.find({}, {email: 1, role: 1, timezone: 1}).toArray()'
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
  <https://aistudio.google.com/apikey>, rồi ghi vào `backend/.env`:

  ```bash
  GEMINI_API_KEY=khoá-của-bạn
  ```

  và khởi động lại backend. Không có khoá thì bài vẫn lưu và nghe được,
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

Request sinh bài dài hơn mọi request khác: route `/api/admin/topics/{id}/generate` có thể chờ tới 120 giây (thời gian tối
đa của AI). Nếu đặt reverse proxy trước backend, hãy cho riêng route này thời gian chờ đủ dài (ví dụ 135 giây).

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

Mở bước Đọc (tài khoản người học): trang bài đang học → bước Đọc (sau bước Từ vựng), hoặc `/lessons/<id>/read`. Chạm một từ để tra,
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

Với **bài đang học** (cập nhật 2026-10-05: hiểu bài trước, luyện lại sau), thứ tự trên trang này là **Từ vựng → Đọc → Nghe →
Hội thoại → Điền ô trống → Dịch câu → Viết** (xem "Học lần lượt từng bài"); xong
bài thì có nút **Sang bài tiếp theo**. Với bài khác, cuối cùng là tổng kết (số ô điền đúng, số câu dịch đúng) và **Làm lại**. Phần luyện tập là tự chọn: chấm ngay trên trình
duyệt, điểm không gửi lên máy chủ, không ảnh hưởng các bước của bài, tiến độ, streak hay thống kê; không có điểm XP.
Riêng **bài đang học** (cập nhật 2026-10-05): từ làm sai ở bước Điền ô trống, hoặc có trong câu dịch làm sai, được đưa vào lịch ôn
(`POST /api/vocab/practice-misses`): từ chưa có trong sổ thành thẻ mới đến hạn ngay (nghĩa, phiên âm, câu lấy từ chú thích của bài,
không lấy từ trình duyệt); thẻ đã có mà đến hạn muộn hơn thì được kéo về bây giờ, giữ nguyên trạng thái FSRS. Mỗi từ chỉ gửi một
lần mỗi lần mở bài; trang ghi "Đã đưa N từ làm sai vào lịch ôn". Gửi lỗi thì bỏ qua, bài học không bị ảnh hưởng.

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
  Tiếp tục: <bước>**, hoặc trang **Bài học**): **Từ vựng** (F17), **Đọc** → **Nghe**, rồi phần luyện tập trên chính bài (**Hội
  thoại** → **Điền ô trống** → **Dịch câu**, F17), cuối cùng **Viết** (F8, tuỳ chọn), chung một thanh tiến độ (cập nhật 2026-10-05: trước
  đây luyện tập đứng trước bước Đọc). Thoát ra vào lại thì tiếp tục đúng bước và đúng câu. Máy chủ chỉ biết Đọc, Nghe, Viết, nên bước đang
  học được nhớ thêm **trên thiết bị** (`localStorage`, cập nhật 2026-10-05): vào lại giữa phần luyện tập thì mở đúng bước đó, không nhảy
  tới Viết hay về bước 1; không bao giờ lùi về trước bước mà máy chủ đã ghi là xong; xoá khi xong bài. Đổi máy hoặc xoá dữ liệu trình
  duyệt thì chỉ còn trạng thái của máy chủ (vào lại sau bước Nghe sẽ mở bước Viết, **← Bước trước** để quay lại). Nút **← Bước trước** mở lại bước đã qua
  (bước Đọc, Nghe đã xong chỉ để xem, không đổi tiến độ).
- Xong bài (nộp bài viết hoặc bấm **Bỏ qua** ở bước Viết): mục tiêu +1, ngày đó tính vào chuỗi ngày học, và nút **Sang bài tiếp
  theo** mở ngay bài kế tiếp — học bao nhiêu bài một ngày cũng được. Nghỉ **đúng một ngày** thì chuỗi vẫn giữ (ngày nghỉ không được tính; cập nhật 2026-10-05), nhưng lần tha sau phải cách ít nhất 7 ngày
  học; nghỉ hai ngày liền thì chuỗi về 0. Không lưu gì thêm, chỉ tính từ các ngày đã xong bài. Bài đã xong trước khi có
  bước Viết vẫn tính là xong.
- **Bài học** (header → **Bài học**, `/lessons`): các bài đã học (đọc/nghe lại, không đổi tiến độ), bài đang học, các bài sắp tới
  bị khoá — người học không mở được bài chưa tới lượt. Không có ô trống: bài nào có thì hiện bài đó.
- Ôn thẻ đến hạn ở **Ôn tập** (`/vocabulary/review`), không còn là bước của bài. Nhưng (cập nhật 2026-10-05) khi xong bài mà còn thẻ
  đến hạn, thẻ **Hoàn thành bài học** hiện "Ôn trước khi học tiếp" (số thẻ, thời gian ước tính ~20 giây/thẻ; từ 30 thẻ trở lên nói rõ
  là đang tồn nhiều). Nút này là nút chính, **Sang bài tiếp theo** thành nút phụ và vẫn bấm được ngay. Ôn xong, trang Ôn tập có nút
  **Học bài tiếp theo** (`/vocabulary/review?next=<id bài>`). Hỏi số thẻ lỗi thì không hiện gợi ý.
- **Trang chủ** (cập nhật 2026-10-05): thẻ "Hôm nay: N thẻ cần ôn" nằm trên bài đang học khi có thẻ đến hạn; "Ngày mai: N thẻ" vẫn
  ở cuối. Ba thanh kỹ năng là **độ chính xác**, không phải số bài đã qua: Đọc = tỷ lệ trả lời đúng câu hỏi hiểu bài, Nghe = tỷ lệ từ
  đúng khi chép chính tả, Viết = điểm trung bình bài viết chia cho 5; kỹ năng chưa có dữ liệu ghi "Chưa có". Ô thứ ba là "Đọc & nghe
  đúng" (trung bình đọc và nghe). Bài đang học chỉ có **một** thẻ (trước đây có thêm thẻ vòng tròn lặp lại tiến độ); tiến độ của nó ghi
  "x/3 phần" vì máy chủ chỉ đếm ba phần chính Đọc, Nghe, Viết (phần Từ vựng và luyện tập không được đếm), không còn ghi "bước" gây
  lệch với 7 bước trong trang bài.
- **Chọn chủ đề** (cập nhật 2026-10-05): chủ đề chưa có bài vẫn hiện (ghi "Chưa có bài") nhưng nút chọn bị khoá, để người học không
  rơi vào một khoá rỗng; trình độ chưa có chủ đề nào thì báo rõ.
- **Nội dung do AI tạo** (cập nhật 2026-10-05, chỉ áp dụng cho bài tạo hoặc chú thích lại từ nay): số từ chú thích theo trình độ (A1–A2
  6–12, B1–B2 8–16, C 8–25) và bỏ qua từ quá cơ bản; câu hỏi hiểu bài phải đa dạng: tối đa một câu hỏi chi tiết thuần tuý, ít nhất một câu hỏi nghĩa của từ trong ngữ cảnh và ít nhất một câu hỏi suy luận hoặc "vì sao", lời giải thích nêu lý do chứ không chỉ chép lại câu; ghi chú ngữ pháp ngắn hơn cho người mới (A1–A2 tối đa 60 từ, B 90, C 120),
  chọn cấu trúc nổi bật của bài thay vì luôn là thì hiện tại đơn; đề viết không nêu số câu hoặc số từ (độ dài hiển thị riêng theo
  trình độ); khi sinh bài, AI phải viết đủ độ dài, văn tự nhiên, và được bỏ một từ mục tiêu nếu không dùng tự nhiên được ở trình độ đó.
  Bài đã có thì bấm **Chạy lại chú thích** để dùng quy tắc mới.
- **Giáo trình ngữ pháp** (cập nhật 2026-10-05, F19): `backend/internal/grammar/syllabus.json` có 67 điểm theo trình độ (A1 15, A2 14, B1 13, B2 11,
  C1 8, C2 6), xếp theo thứ tự dạy. Quản trị viên gán cho mỗi bài tối đa một điểm của đúng trình độ: ở ô **Điểm ngữ pháp** trong hộp thoại
  **Sinh bài bằng AI** (mặc định là "AI tự chọn": AI tự lồng ngữ pháp vào bài cho người học quen dần; chọn một điểm cụ thể chỉ khi muốn) và trong form thêm/sửa bài. AI sinh bài phải dùng điểm
  đó và ghi chú ngữ pháp của bài dạy đúng điểm đó, với tên lấy từ giáo trình. Sửa giáo trình bằng cách sửa file JSON (không đổi hay dùng lại mã đã
  gán). Đổi điểm của bài đã chú thích rồi thì bấm **Chạy lại chú thích** để ghi chú theo điểm mới. Chi tiết: `docs/phases/giai-doan-3.md`, mục F19.
- **Học ngữ pháp riêng** (cập nhật 2026-10-05, F20): tab **Ngữ pháp** (`/grammar`) là chỗ học và luyện ngữ pháp có hệ thống, tách khỏi bài học theo
  chủ đề. Mỗi điểm của giáo trình có một **bài ngữ pháp**: mục tiêu, giải thích, khi nào dùng, bảng cấu trúc, ví dụ (nghe được), lỗi thường gặp, bài
  **luyện tập** (phản hồi ngay từng câu) và bài **kiểm tra mức nắm vững** (chấm cuối bài; đạt từ 80% là "Đã nắm vững", không bao giờ mất). Ba dạng
  bài tập: trắc nghiệm, điền từ, sắp xếp câu, tự chấm trên trình duyệt (không gọi AI); câu làm sai được ghi để luyện lại. Danh sách theo trình độ,
  gợi ý điểm kế tiếp, không khoá cứng. Bài ngữ pháp do **AI sinh rồi quản trị viên duyệt và đăng** ở mục **Ngữ pháp** của khu quản trị
  (`/admin/grammar`: Sinh bằng AI, xem trước, sửa nội dung bằng JSON, Đăng/Gỡ); người học chỉ thấy bài đã đăng, điểm chưa đăng ghi "Sắp có".
  Mỗi bài sinh tốn **một** request AI. Từ trang đọc bài, ghi chú ngữ pháp có liên kết "Học kỹ điểm này →" tới bài ngữ pháp của điểm đó.
  Dữ liệu tiến độ (`grammar_progress`) nằm trong file xuất dữ liệu. Chưa có: đưa câu sai vào lịch ôn FSRS, thư viện tra cứu, kiểm tra trình độ.
- **Kiểm soát chất lượng bài ngữ pháp** (cập nhật 2026-10-05, F21): nút **Kiểm tra bằng AI** cho AI giải lại bài tập không kèm đáp án rồi so với đáp án đã lưu; câu lệch hoặc mơ hồ bị gắn cờ và Đăng sẽ hỏi xác nhận (`acknowledgeFlags`). Đáp án điền từ được mở rộng bằng mã (viết tắt, dấu nháy cong). Người học có nút **Báo lỗi câu này**; quản trị xem ở mục *Câu bị báo lỗi* và bấm *Đã xử lý*. Bảng/collection `grammar_reports`. Từng câu bị cờ có thể xác nhận, sửa hoặc xoá, và cả bài có nút "Xác nhận đã kiểm tra xong".
- **Kiểm tra bằng AI cho bài học** (cập nhật 2026-10-05, F22): trang quản trị bài học có nút **Kiểm tra bằng AI**: một AI thứ hai đọc lại câu của bài, chú thích, câu hỏi đọc hiểu (giải không thấy đáp án) và câu dịch ở phần luyện tập, rồi gắn cờ chỗ nghi sai. Quản trị xác nhận, sửa hoặc xoá từng chỗ, rồi bấm "Xác nhận đã kiểm tra xong". Chỉ chạy khi quản trị bấm; sửa nội dung bài thì kết quả bị xoá, sửa câu hỏi, chú thích, câu dịch thì giữ các cờ chưa đụng tới (F22b). Học viên không thấy phần này.
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
  biệt hoa thường); chọn Quên / Khó / Nhớ / Dễ (tương ứng Again / Hard / Good / Easy của FSRS; phím 1–4), ngày ôn tiếp theo tính bằng FSRS
  ([go-fsrs](https://github.com/open-spaced-repetition/go-fsrs), tham số mặc định). Thẻ mới đến hạn lần đầu lúc 0 giờ hôm sau.
- **Mục Từ vựng** ở cuối bước Đọc: các từ đã chú thích của bài, nút Lưu / Lưu tất cả (không tạo thẻ trùng), không gọi AI.

Thử ôn ngay một thẻ vừa lưu (mặc định phải chờ tới hôm sau):

```bash
mongosh "$MONGO_URI" --quiet \
  --eval 'db.getSiblingDB("luna").cards.updateMany({}, {$set: {due: new Date(Date.now() - 60000)}})'
```

## Sao lưu và khôi phục

App không tự sao lưu. Dùng `mongodump`/`mongorestore` ([MongoDB Database Tools](https://www.mongodb.com/try/download/database-tools)),
hoặc tính năng backup của MongoDB Atlas. Các bản cũ nằm ở `deploy/backups/luna-YYYYMMDD.archive.gz`. Từ điển không cần sao lưu: tải lại
bằng `deploy/fetch-dictionary.sh`.

```bash
mongodump --uri "$MONGO_URI" --db luna --archive=deploy/backups/luna-$(date +%Y%m%d).archive.gz --gzip
```

**Khôi phục** (thay toàn bộ dữ liệu hiện tại bằng dữ liệu của bản sao lưu, kể cả tài khoản và phiên đăng nhập — mọi thay đổi sau thời
điểm sao lưu sẽ mất). Dừng backend trước, khôi phục xong thì chạy lại:

```bash
mongorestore --uri "$MONGO_URI" --nsInclude 'luna.*' --drop --gzip --archive=deploy/backups/luna-20260930.archive.gz
```

## Cấu hình

Backend đọc biến môi trường. Khi chạy, nó tự đọc thêm file `.env` trong thư mục đang đứng (biến đặt thật luôn được ưu tiên).
Chép file mẫu rồi sửa:

```bash
cp backend/.env.example backend/.env
```

| Biến | Dùng ở | Bắt buộc | Mặc định | Ý nghĩa |
|---|---|---|---|---|
| `DB_DRIVER` | backend | Không | `mongo` | Loại cơ sở dữ liệu: `mongo` hoặc `mysql` (không phân biệt hoa thường); xem "Chọn cơ sở dữ liệu" |
| `MONGO_URI` | backend | Có khi `DB_DRIVER=mongo` | | Chuỗi kết nối MongoDB |
| `MYSQL_*` | backend | Khi `DB_DRIVER=mysql` | xem "Chọn cơ sở dữ liệu" | Máy chủ, cổng, người dùng, mật khẩu, database, pool; hoặc cả chuỗi `MYSQL_DSN`. Mật khẩu chỉ đặt trong `.env` |
| `HTTP_ADDR` | backend | Không | `:8080` | Địa chỉ backend lắng nghe |
| `MONGO_DATABASE` | backend | Không | `luna` | Tên database |
| `LOG_LEVEL` | backend | Không | `info` | `debug`, `info`, `warn`, `error` |
| `COOKIE_SECURE` | backend | Không | `false` | `true` khi chạy qua HTTPS (thêm cờ `Secure` cho cookie đăng nhập) |
| `CORS_ORIGINS` | backend | Không | trống | Origin được gọi API từ trình duyệt, cách nhau bằng dấu phẩy, hoặc `*` cho mọi origin. Cần khi frontend ở origin khác backend |
| `AI_PROVIDER` | backend | Không | `gemini` | `gemini` hoặc `none` |
| `GEMINI_API_KEY` | backend | Không | rỗng | Khoá Gemini (bí mật, chỉ để trong `.env`) |
| `GEMINI_MODEL` | backend | Không | `gemini-3.5-flash-lite` | Model dùng để chú thích và sinh bài |
| `DICTIONARY_PATH` | backend | Không | `./data/dictionary/dictionary.db` | File từ điển SQLite |

Thiếu `MONGO_URI` (khi `DB_DRIVER=mongo`) hoặc `MYSQL_USER`/`MYSQL_DSN` (khi `DB_DRIVER=mysql`), `DB_DRIVER` không phải `mongo`/`mysql`, hoặc `LOG_LEVEL` / `COOKIE_SECURE` / `AI_PROVIDER` sai thì backend in lỗi nêu tên biến và dừng với mã 1.
Mọi file `.env` đều bị git bỏ qua; không commit bí mật vào repo.

### Chọn cơ sở dữ liệu

Backend chọn CSDL lúc khởi động theo `DB_DRIVER` (mặc định `mongo`), không cần sửa code. Chỉ biến của loại đã chọn được đọc: `mongo`
dùng `MONGO_URI` và `MONGO_DATABASE`, `mysql` dùng các biến `MYSQL_*` ở bảng dưới (không được kiểm tra khi chọn `mongo`). Log có dòng
`database selected` ghi loại đang dùng.

| Biến MySQL | Bắt buộc | Mặc định | Ý nghĩa |
|---|---|---|---|
| `MYSQL_HOST` | Không | `localhost` | Máy chủ MySQL |
| `MYSQL_PORT` | Không | `3306` | Cổng (1–65535) |
| `MYSQL_USER` | Có, trừ khi dùng `MYSQL_DSN` | | Tên người dùng |
| `MYSQL_PASSWORD` | Không | trống | Mật khẩu; ký tự đặc biệt như `@ / : ? &` dùng được, không cần mã hoá |
| `MYSQL_DATABASE` | Không | `luna` | Tên database; phải tạo sẵn |
| `MYSQL_DSN` | Không | trống | Chuỗi `user:password@tcp(host:3306)/database`; nếu đặt thì **thay** năm biến trên |
| `MYSQL_MAX_OPEN_CONNS` | Không | `20` | Số kết nối tối đa của pool |
| `MYSQL_MAX_IDLE_CONNS` | Không | `5` | Số kết nối nhàn rỗi giữ lại |
| `MYSQL_CONN_MAX_LIFETIME` | Không | `5m` | Thời gian tối đa dùng lại một kết nối (ví dụ `30s`, `10m`) |
| `MYSQL_DIAL_TIMEOUT` | Không | `2s` | Thời gian chờ tối đa khi nối tới máy chủ |

Số hoặc thời lượng sai (ví dụ `MYSQL_PORT=abc`, `MYSQL_CONN_MAX_LIFETIME=5`) thì backend in lỗi nêu tên biến rồi dừng; lỗi không bao giờ
chứa giá trị, nên mật khẩu không lộ ra log. Backend luôn tự thêm `parseTime`, múi giờ UTC và `utf8mb4`.

**Trạng thái:** cả hai loại chạy đầy đủ tính năng (MongoDB và MySQL 8.0.19 trở lên). Chuyển loại không tự chuyển dữ liệu: mỗi CSDL
là một kho riêng, bắt đầu rỗng (tài khoản đầu tiên vẫn là quản trị viên). MySQL tự tạo bảng lúc khởi động (`schema_migrations` ghi các
đợt đã chạy; có khoá nên hai backend khởi động cùng lúc không đạp nhau) và nạp từ vựng chủ đề như MongoDB.

Chạy bằng MySQL: đặt `DB_DRIVER=mysql` và các biến `MYSQL_*` trong `backend/.env`. Sao lưu MySQL bằng `mysqldump`.

**Test MySQL:** test tích hợp ở `backend/internal/storage/mysql` cần một server MySQL và tự bỏ qua nếu thiếu biến `MYSQL_TEST_DSN`:

```bash
docker run -d --name luna-mysql-test -e MYSQL_ROOT_PASSWORD=luna -p 127.0.0.1:3307:3306 mysql:8
MYSQL_TEST_DSN='root:luna@tcp(127.0.0.1:3307)/' go test ./internal/storage/mysql/   # trong thư mục backend
```

Mỗi test tạo một database riêng rồi xoá, nên không đụng dữ liệu thật.

Cách thêm một loại CSDL: xem `docs/architecture.md`, mục "Lớp lưu trữ".

## Phát triển

Cần thêm Go 1.25 và Node 22. Chạy từng phần riêng:

**1. Cơ sở dữ liệu**: MongoDB Atlas (cloud) hoặc MongoDB cài trên máy. Đặt `MONGO_URI` trong `backend/.env`
(với Atlas: chuỗi kết nối lấy ở **Connect** → **Drivers**; thêm IP máy này ở **Network Access**). Backend tự tạo index trên
DB mới. Từ điển offline cho bước Đọc: chạy `deploy/fetch-dictionary.sh` (hoặc `.ps1`) một lần.

**2. Backend** (`http://localhost:8080`). Lần đầu, chép file cấu hình mẫu cho môi trường phát triển:

```bash
cd backend
cp .env.example .env      # PowerShell: Copy-Item .env.example .env
go run ./cmd/api
```

Khi chạy local, backend tự đọc `backend/.env` trong thư mục đang đứng (nên chạy lệnh từ `backend/`).
Biến môi trường đặt thật (terminal, dịch vụ hệ thống) luôn được ưu tiên hơn giá trị trong file.
`backend/.env` bị git bỏ qua.

**3. Frontend** (`http://localhost:4200`, tự tải lại khi sửa code). Trình duyệt gọi thẳng backend theo
`apiUrl` trong `frontend/src/environments/environment.development.ts` (mặc định `http://localhost:8080/api`; bản prod ở
`environment.ts`). `apiUrl` luôn kết thúc bằng `/api`, code chỉ gọi phần sau (ví dụ `/auth/login`).
`CORS_ORIGINS` trong `backend/.env` phải cho phép origin của frontend (mặc định `*`, mọi origin):

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
deploy/     fetch-dictionary.sh/.ps1, data/ (dữ liệu tải về), backups/ (bản sao lưu cũ)
docs/       Tài liệu sản phẩm (nguồn sự thật)
specs/      Spec, plan, tasks của từng tính năng
```

Chi tiết: [`docs/architecture.md`](docs/architecture.md), [`docs/design-system.md`](docs/design-system.md).
#   l u n a  
 