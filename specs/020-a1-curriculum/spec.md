# Feature Specification: Giáo trình theo trình độ (A1 trước, sau đó mọi trình độ)

**Feature Branch**: `020-a1-curriculum`

**Created**: 2026-10-22 · **Updated**: 2026-10-26

**Status**: Implemented (A1: FR-001 … FR-009; mở rộng 2026-10-23: FR-010 … FR-025; mở rộng 2026-10-26: FR-026 … FR-031).

**Input**: mục 3.2 của bản đánh giá hệ thống, bản phác thảo `design/de-xuat-giao-trinh-a1.png` đã được duyệt
("chọn phương án 2": mỗi chủ đề 3 bài chính + 2–3 bài thêm). Đã chốt: có danh sách từ theo cấp CEFR để cảnh báo từ vượt
trình độ; điểm ngữ pháp chưa tới vẫn mở xem được ("Học trước").

Mở rộng 2026-10-23: người dùng hỏi vì sao trang Giáo trình chỉ có A1. Đã chốt: giáo trình lưu trong DB, không lưu file
JSON; chủ đề được đánh dấu theo trình độ; trang Giáo trình chỉ gọi API; giáo trình A2 trở lên do **AI soạn, admin duyệt**.

## Tóm tắt

A1 có một giáo trình cố định: 12 chủ đề theo thứ tự ngữ pháp, mỗi chủ đề 3 bài chính và 2–3 bài thêm (36 + 29 = 65 bài),
đủ 18/18 điểm ngữ pháp A1. Ban đầu giáo trình nằm trong mã nguồn (`backend/internal/curriculum/a1.json`), như phần Ngữ
pháp.

Từ 2026-10-23, giáo trình chuyển vào DB và gắn vào từng chủ đề theo từng trình độ. Một chủ đề có thể nằm trong giáo
trình của nhiều trình độ, ví dụ "Bạn bè và các mối quan hệ" ở cả A1 và A2. Trang Giáo trình có nút chọn trình độ. Với
trình độ chưa có giáo trình, admin bấm "Soạn giáo trình": AI soạn bản nháp, admin xem, sửa, rồi duyệt. Chỉ giáo trình đã
duyệt mới hiện với người học và mới sinh được bài.

## Yêu cầu: giáo trình A1 (đã làm)

Từ FR-010 trở đi, các yêu cầu dưới đây áp dụng cho mọi trình độ đã duyệt và đọc giáo trình từ DB, không đọc `a1.json`
nữa (xem FR-017).

- **FR-001** Giáo trình: mỗi chủ đề có tên như danh mục chủ đề, câu "Sau chủ đề này bạn có thể…", điểm ngữ pháp, và các ô
  bài (khoá, loại chính/thêm, tên bài, điểm ngữ pháp, ý tưởng tiếng Anh cho AI). 3 ô đầu là bài chính. Kiểm tra khi khởi
  động: điểm ngữ pháp đúng trình độ, khoá và tên bài không trùng.
- **FR-002** Một ô có bài khi lộ trình của chủ đề (trình độ A1) có bài cùng tên (không phân biệt hoa thường, khoảng trắng,
  kiểu dấu nháy). Không thêm cột/trường mới cho bài học.
- **FR-003** Trang quản trị "Giáo trình" (`GET /api/admin/curriculum?level=A1`): từng chủ đề, từng ô (trống, nháp, đã
  đăng), bài khác trong chủ đề, điểm ngữ pháp nào chưa đăng bài ngữ pháp.
- **FR-004** "Áp dụng thứ tự giáo trình" (`POST /api/admin/curriculum/apply`): tạo chủ đề còn thiếu (mô tả = câu "Sau chủ
  đề này…"), đặt thứ tự chủ đề A1 theo giáo trình (chủ đề ngoài giáo trình xếp sau), sắp lộ trình mỗi chủ đề theo ô: bài
  chính, bài thêm, rồi bài khác.
- **FR-005** "Sinh bài này" (`POST /api/admin/curriculum/slots/{key}/generate`): 1 request AI; tên bài = tên ô, điểm
  ngữ pháp = điểm đầu của ô (các điểm sau đưa vào ý tưởng), 8 từ mục tiêu theo kế hoạch từ của chủ đề, 120 từ; lưu thành
  bài nháp ở đúng chỗ trong lộ trình. "Sinh các ô trống" của một chủ đề gọi lần lượt từng ô.
- **FR-005b** "Gỡ" bên mỗi bài (`POST /api/admin/curriculum/lessons/{id}/remove`): gỡ bài khỏi lộ trình để ô trống lại.
  Từ 2026-10-26 hộp thoại hỏi cách gỡ (FR-029): **chỉ gỡ** (mặc định, bài về nháp, còn ở trang Bài học) hoặc **gỡ và xoá
  hẳn** (như trước, cả bài đã đăng, từ 2026-10-23 theo yêu cầu của người dùng).
- **FR-006** Kiểm tra bài nháp (`GET /api/admin/curriculum/checks/{lessonId}`, cũng có sẵn trong bảng cho bài nháp): số
  lần dùng điểm ngữ pháp (≥ 3, đếm bằng mẫu cho 18 điểm A1), có đoạn he/she, câu phủ định, câu hỏi, đủ từ mục tiêu, độ dài
  (A1 90–150 từ), hội thoại (≥ 3 lượt hỏi, không chép nguyên câu bài đọc; khi đã có luyện tập), từ vượt trình độ.
- **FR-007** Từ vượt trình độ dùng CEFR-J Wordlist 1.5 (A1–B2, Tono Laboratory, TUFS; dùng miễn phí có ghi nguồn), gói
  `backend/internal/cefr`: tra theo dạng gốc, bỏ tên riêng (chữ hoa giữa câu), từ không có trong danh sách.
- **FR-007b** Sinh bài chỉ dùng từ đúng trình độ (người dùng chọn "cách 3", áp dụng cho cả trang Giáo trình và trang Lộ
  trình): lời nhắc sinh bài yêu cầu từ thông dụng của trình độ; sinh xong, bài nào còn từ vượt trình độ (CEFR-J, trừ từ
  mục tiêu và tên riêng) được gửi lại AI **1 request cho cả lô** (op `lesson_simplify`) để chỉ thay những từ đó. Bản sửa
  chỉ được nhận khi ít từ khó hơn, độ dài trong ±20 % và không mất từ mục tiêu; lỗi AI thì giữ bản gốc. Từ còn lại trả về ở
  `hardWords` của bản nháp và hiện cảnh báo.
- **FR-008** Lộ trình người học (`GET /api/lessons/path`): chủ đề trong giáo trình có `course {number, canDoVi,
  grammarVi}`, bài có `extra`. Học xong 3 bài chính thì có lối sang chủ đề sau; bài thêm vẫn học theo thứ tự nếu không chuyển.
- **FR-009** Ngữ pháp (`GET /api/grammar?level=A1`): điểm xếp theo thứ tự giáo trình, mỗi điểm có `courseNumber`,
  `courseTopic`; "Học tiếp" ưu tiên điểm của chủ đề đang học; điểm của chủ đề sau vẫn mở xem được (nhãn "Học trước").

## Yêu cầu: giáo trình trong DB, mọi trình độ (mở rộng 2026-10-23, đã làm)

### Dữ liệu

- **FR-010** Giáo trình lưu trong collection `curricula` (MySQL: bảng `curricula`), **mỗi trình độ một bản ghi**:
  `{ _id: level, approved: { topics[] }, draft: { topics[] }, approvedAt, updatedAt }`. Mỗi chủ đề có `topicId` (chủ đề
  trong danh mục, rỗng khi chưa tạo), `name`, `canDoVi`, `grammar[]`, `slots[]`; mỗi ô có `key`, `kind`, `title`,
  `grammar[]`, `idea` như FR-001. Chủ đề **thuộc giáo trình của trình độ L** khi giáo trình đã duyệt của L có nó; một chủ
  đề có thể ở nhiều trình độ. Khớp chủ đề theo `topicId`, chưa có id thì theo tên.
  _Đổi so với dự kiến_: không lưu `courses.<level>` trên từng chủ đề vì bản nháp AI soạn có thể có chủ đề chưa tồn tại.
- **FR-011** Người học, ngữ pháp "Học trước" và "Sinh bài này" chỉ đọc `approved`; trang Giáo trình hiện `draft` khi có.
  Bản đã duyệt được giữ trong bộ nhớ 30 giây (dev và production chung DB nên duyệt ở nơi này thì nơi kia thấy sau tối đa 30 giây).
- **FR-012** A1: lần đọc đầu tiên khi `curricula` chưa có A1 thì chép `a1.json` vào làm bản đã duyệt (khoá ô `a1-01-1`…
  giữ nguyên). Sau đó chỉ đọc DB; `a1.json` chỉ còn là dữ liệu gốc. Khoá ô luôn là `<level>-<số chủ đề>-<số bài>`, 3 bài
  đầu là bài chính (đổi chỗ bài thì khoá đổi theo; bài khớp ô theo tên nên không ảnh hưởng).
- **FR-013** Kiểm tra khi lưu (AI soạn hoặc admin sửa), thay cho kiểm tra lúc khởi động của FR-001:
  - **chặn lưu**: chủ đề chưa có tên hoặc trùng tên; điểm ngữ pháp không có trong danh mục hoặc khác trình độ; bài chưa có
    tên, ý tưởng hoặc ngữ pháp; tên bài trùng trong một chủ đề;
  - **chặn duyệt** thêm: chưa có chủ đề nào, hay chủ đề có ít hơn 3 bài chính + 2 bài thêm;
  - **cảnh báo** (không chặn): ít hơn 7 chủ đề; số bài thêm ngoài 4–5 (A1 hiện có 2–3, admin bổ sung qua FR-018); điểm
    ngữ pháp của trình độ chưa chủ đề nào dạy.

### Admin: chọn trình độ

- **FR-014** Trang "Giáo trình" có ô chọn trình độ A1 → C2, ghi vào URL (`?level=A2`). `GET /api/admin/curriculum?level=L`
  trả thêm `status` (`none` | `draft` | `approved`), `hasApproved`, `warnings`, `points` (điểm ngữ pháp của L). Trình
  độ chưa có giáo trình: "Chưa có giáo trình L" và nút "Soạn giáo trình bằng AI", không báo lỗi. Trình độ không hợp lệ: 400.

### Admin: AI soạn, admin duyệt

- **FR-015** AI soạn theo hai bước, **gọi thẳng, không qua hàng đợi job** (nên bản production cũ không lấy mất việc):
  - "Soạn giáo trình bằng AI" (`POST /api/admin/curriculum/draft?level=L`, op `curriculum_draft`, 1 request): đầu vào là
    các điểm ngữ pháp của L, tên các chủ đề trong DB, gợi ý chủ đề của L (`backend/internal/curriculum/suggestions.json`,
    chuyển từ `topic-curriculum.ts`), chủ đề của các giáo trình đã duyệt ở trình độ dưới. AI trả 7–10 chủ đề với
    `canDoVi` và `grammar`. Sai (ít/nhiều chủ đề, thiếu tên, điểm ngữ pháp lạ) thì gửi lại 1 lần kèm lỗi; vẫn sai thì
    báo, không lưu. Lưu thành bản nháp (thay bản nháp cũ), chủ đề trùng tên danh mục được gắn `topicId`.
  - "Soạn bài bằng AI" cho từng chủ đề (`POST /api/admin/curriculum/draft/topics/{n}/write?level=L`, op
    `curriculum_draft_topic`, 1 request/chủ đề): AI soạn 3 bài chính + 5 bài thêm (tên, ý tưởng tiếng Anh, ngữ pháp),
    tránh tên bài đã dùng. Bỏ bài trùng tên hoặc thiếu ý tưởng; ngữ pháp khác trình độ thì thay bằng điểm đầu của chủ đề;
    còn ít hơn 5 bài dùng được thì báo lỗi. Nút "Soạn bài cho các chủ đề chưa có bài" gọi lần lượt từng chủ đề.
  - Lời nhắc viết bằng tiếng Anh.
- **FR-016** Bản nháp sửa ngay trên trang Giáo trình (nhãn "Bản nháp, chưa duyệt"): tên chủ đề (gợi ý tên trong danh
  mục; đổi tên thì bỏ liên kết, khi duyệt tìm lại theo tên), "Sau chủ đề này bạn có thể…", ngữ pháp của chủ đề (ô
  chọn), từng bài (tên, ý tưởng, ngữ pháp chính); ↑ ↓ đổi thứ tự chủ đề và bài (thay cho kéo thả); bỏ/thêm chủ đề, bỏ/thêm
  bài; "Soạn lại bài" cho một chủ đề. Thay đổi giữ trên trang tới khi bấm "Lưu bản nháp" (`PUT /api/admin/curriculum/draft`);
  gọi AI hay duyệt thì tự lưu trước. "Soạn lại toàn bộ bằng AI" và "Bỏ bản nháp" (`DELETE …/draft`) có hộp xác nhận.
  Loại chính/thêm theo vị trí (3 bài đầu là chính), không chọn riêng.
- **FR-017** "Duyệt giáo trình" (`POST /api/admin/curriculum/approve?level=L`), có hộp xác nhận: kiểm tra FR-013, tạo chủ
  đề còn thiếu, bản nháp thành bản đã duyệt. Duyệt chưa đụng tới thứ tự chủ đề và lộ trình; admin bấm "Áp dụng thứ tự"
  (FR-004) khi đã sẵn sàng. Từ đây FR-002 … FR-009 chạy cho L như cho A1.
- **FR-018** Sửa giáo trình đã duyệt (cả A1): "Sửa giáo trình" (`POST /api/admin/curriculum/edit?level=L`) tạo bản nháp từ bản đã duyệt, sửa như FR-016, rồi duyệt
  lại. Ô đã có bài vẫn khớp theo tên như FR-002. Đổi tên hay bỏ ô đã có bài thì chủ đề hiện cảnh báo "Bài đã có sẽ không
  khớp ô nào nữa (thành “bài khác” của chủ đề)".

### Kiểm tra bài nháp cho trình độ khác A1

- **FR-019** FR-006 chạy theo trình độ của bài:
  - độ dài theo số từ mặc định của trình độ ±25 % (A1 120, A2 160, B1 220, B2 300, C1/C2 400), sinh bài cũng dùng số từ này;
  - đếm số lần dùng điểm ngữ pháp: chỉ chạy cho điểm đã có mẫu nhận dạng (18 điểm A1); điểm khác hiện "Điểm ngữ pháp: …
    (không tự đếm được)", không tính là lỗi;
  - từ vượt trình độ: CEFR-J chỉ tới B2 nên bài B2 trở lên ghi "Không kiểm tra từ vượt trình độ".

### Trang Chủ đề

- **FR-020** Bỏ khung "Gợi ý lộ trình chủ đề theo trình độ" ở trang Chủ đề, vì nút "Áp dụng thứ tự gợi ý" của nó cũng ghi
  `positions.<level>` và sẽ đè thứ tự của giáo trình đã duyệt; tạo chủ đề, sinh bài và trạng thái bài đã có ở trang Giáo
  trình. Thay bằng một dòng: "Thứ tự chủ đề theo trình độ: xem trang Giáo trình" (có liên kết). Danh sách gợi ý và lý do
  (`topic-curriculum.ts`) chuyển sang backend, chỉ làm đầu vào cho AI (FR-015).

### Trang Lộ trình

- **FR-021** Trang Lộ trình vẫn giữ (chủ đề ngoài giáo trình, thêm bài có sẵn, sinh bài tự do, kéo thả). Khi chủ đề và
  trình độ đang chọn thuộc giáo trình đã duyệt, hiện dòng: "Chủ đề này thuộc giáo trình L (chủ đề n). Thứ tự bài theo giáo
  trình, sinh bài theo ô ở trang Giáo trình. Đổi thứ tự ở đây sẽ bị xếp lại khi bấm “Áp dụng thứ tự”", có liên kết sang
  trang Giáo trình của L. Dữ liệu lấy từ `GET /api/admin/curriculum/topics` (mỗi trình độ đã duyệt: `topicId`, tên, số
  thứ tự), gọi một lần khi mở trang.

### Rà soát admin 2026-10-23

- **FR-022** Từ vựng của chủ đề trên trang Giáo trình: mỗi chủ đề có "Từ vựng: N từ dùng được ở L" và liên kết "Sửa
  từ vựng" (`/admin/topics/{id}/words`); dưới số từ mục tiêu của trình độ (A1–A2 8, B1–B2 10, C1–C2 12) thì có dấu
  cảnh báo, và bảng cảnh báo của giáo trình ghi chủ đề đó (bài sinh ra sẽ thiếu từ mục tiêu). API: `words` ở mỗi chủ đề.
- **FR-023** "Soạn thêm bài thêm (n)" trong bản nháp (`POST …/draft/topics/{n}/write?level=L&mode=extra`): chủ đề đã có
  đủ 3 bài chính và chưa đủ 5 bài thêm thì AI chỉ soạn thêm số bài thêm còn thiếu, giữ nguyên các bài đang có (để bài đã
  sinh vẫn khớp ô). Dùng cho A1 (đang 2–3 bài thêm): "Sửa giáo trình" → "Soạn thêm bài thêm" → duyệt.
- **FR-024** Trang chủ admin: lối tắt "Sinh bài bằng AI" sang trang Giáo trình; "Việc cần làm" thêm theo từng trình độ:
  bản nháp giáo trình chờ duyệt, giáo trình chưa áp dụng thứ tự, bài nháp trong giáo trình chờ đăng, ô trống (liên kết tới
  `/admin/curriculum?level=L`). API `GET /api/admin/curriculum/summary`.
- **FR-025** "Thử giọng đọc" và "Thử nhận dạng giọng nói" ra khỏi menu (mô hình đã chọn), còn liên kết ở mục "Công cụ thử
  nghiệm" cuối trang chủ admin; đường dẫn giữ nguyên.
- Để sau: lọc nháp/đã đăng và đăng hàng loạt ở trang Bài học; trang xem tiến độ người học; gộp "Giao diện" với cài đặt tài
  khoản.

## Yêu cầu: nhập bài soạn sẵn, không qua AI (mở rộng 2026-10-26)

Người dùng hỏi có thể thêm bài bằng tay để agent (Claude) soạn và đưa vào thay vì AI sinh không. Đã chốt (bản phác thảo
`design/de-xuat-nhap-bai-soan-san.png`): thêm API nhập bài cho ô giáo trình và bài ngữ pháp, agent đăng nhập tài khoản admin
riêng để gọi; làm API trước, giao diện nhập bài sau (cùng ngày). Nhãn nguồn của bài nhập là "Nhập".

- **FR-026** `GET /api/admin/curriculum/slots/{key}/brief`: mọi thứ AI nhận khi sinh bài của ô, để người soạn bám theo: tên
  bài (ô khớp với bài theo tên), ý tưởng, chủ đề và "học xong", các điểm ngữ pháp (tên, mẫu, gợi ý, ví dụ), từ mục tiêu
  dự kiến (như "Sinh bài này"), từ vựng của chủ đề ở trình độ, độ dài nên có, số lần tối thiểu dùng điểm ngữ pháp, ô đã
  có bài chưa.
- **FR-027** `POST /api/admin/curriculum/slots/{key}/import` (`?dryRun=true` chỉ kiểm tra, không lưu): nhận `content`,
  `targetWords` (bỏ trống thì lấy từ dự kiến của FR-026) và nội dung AI lẽ ra viết, **đúng khuôn AI trả về**: `annotate`
  (chú thích từ, câu hỏi đọc hiểu, ghi chú ngữ pháp, đề viết, bản dịch từng câu), `practice` (ví dụ, hội thoại, câu dịch
  và 6 bước) và `glosses` (nghĩa trong bài, có thể bỏ trống). Mọi phần qua đúng các hàm kiểm tra của job AI
  (`CleanAnnotations`, `CleanExtras`, `CleanPractice`, `CleanSteps`, `CleanGlosses`), nên bài nhập giống bài AI sinh.
  Trả về kiểm tra theo quy tắc (FR-006) và báo cáo từng phần "đưa n, giữ k"; phần bị bỏ kèm quy tắc (`ruleVi`) và, với
  chú thích, câu hỏi đọc hiểu, bài tập ngữ pháp, lý do từng mục (`reasons`), để người soạn sửa. Thiếu chú
  thích hợp lệ hoặc phần luyện tập không dùng được thì 400 kèm `fields` và báo cáo; JSON có trường lạ thì 400 nêu tên
  trường. Lưu thì bài thành **nháp** của ô, đúng chỗ trong lộ trình, chú thích và luyện tập đã xong, **không xếp job AI**
  nào (nghĩa trong bài để trống thì lưu danh sách rỗng, popup dùng từ điển).
- **FR-028** `POST /api/admin/grammar-lessons/{pointId}/import` (`?dryRun=true` chỉ kiểm tra): nhận `content` cùng khuôn
  với "Lưu" bài ngữ pháp, qua `Normalize` và `Validate`, lưu thành nháp, đánh dấu đã sửa (sinh lại bằng AI sẽ hỏi trước).
  Điểm đã có bài thì 409 `grammar_lesson_exists`, trừ khi gửi `replace: true` (giao diện sẽ hỏi xác nhận); bài bị thay
  về nháp.
- **FR-029** Hộp thoại "Gỡ" (FR-005b) có hai lựa chọn: **Chỉ gỡ khỏi giáo trình** (mặc định; bài rời ô và lộ trình, bài
  đã đăng chuyển về nháp, còn ở trang Bài học) và **Gỡ và xoá hẳn** (nút đỏ "Gỡ và xoá"). API nhận `{"delete": false}`;
  không gửi thì xoá như trước.
- **FR-030** Giao diện nhập bài: nút "Nhập bài" ở ô trống (trang Giáo trình) và ở mỗi điểm ngữ pháp; hộp thoại hiện thông
  tin của ô (FR-026), dán JSON hoặc chọn file `.json` (đọc vào ô dán để sửa được), "Kiểm tra" chạy `dryRun` và xem trước 6
  bước, kiểm tra theo quy tắc, phần bị bỏ kèm lý do, lỗi theo trường; "Lưu thành nháp" chỉ bật khi nội dung đã kiểm tra
  và không đổi sau đó; "Tải file mẫu" tạo khung JSON của ô. Nhãn nguồn "AI sinh" / "Nhập" bên mỗi bài (bài có trường
  `source` trong danh sách); bài ngữ pháp nhập có trường `imported` (MySQL: cột `imported`, migration 039), trạng thái
  "… · nhập". Nhập vào điểm ngữ pháp đã có bài hỏi xác nhận "Thay bài ngữ pháp đã có?" rồi gửi `replace: true`.
- **FR-031** Xoá bài ngữ pháp (`DELETE /api/admin/grammar-lessons/{pointId}`, 204): **chỉ bài nháp** (người dùng chốt
  2026-10-26); bài đã đăng trả 409 `grammar_lesson_published`, phải "Gỡ" (gỡ đăng) trước. Trang Ngữ pháp có nút "Xoá" ở
  bài nháp, hỏi "Xoá bài nháp?" rồi điểm về "Chưa có bài". Tiến độ và báo lỗi của người học ở điểm đó được giữ.

## Ngoài phạm vi

- ~~Giáo trình A2 trở lên.~~ Làm trong phần mở rộng 2026-10-23.
- ~~Admin sửa tên, ý tưởng của ô (sửa trong `a1.json`).~~ Làm trong FR-016, FR-018.
- Mẫu nhận dạng ngữ pháp cho A2 trở lên (để sau, có thể nhờ AI chấm).
- Gợi ý từ dễ hơn cho từ vượt trình độ (cần AI); hiện chỉ liệt kê từ và cấp.
- Sinh nhiều ô trong 1 request AI (mỗi ô 1 request vì mỗi ô có điểm ngữ pháp riêng).
- Người học chọn giáo trình khác trình độ đang học.
- Lịch sử phiên bản giáo trình (chỉ có bản nháp và bản đã duyệt).

## Lưu ý vận hành

- Dev và production dùng chung DB: soạn, sửa, duyệt giáo trình ở local sẽ hiện ngay trên luongtran.site. Bản nháp không
  hiện với người học (FR-011), nên chỉ cần cẩn thận khi bấm "Duyệt" và "Áp dụng thứ tự".
- Lần đầu mở trang Giáo trình (hoặc lộ trình người học) với bản mới, A1 được chép vào `curricula` (FR-012). Bản production
  cũ không đọc `curricula` nên vẫn chạy bình thường với `a1.json` trong lúc chờ deploy.
- AI soạn giáo trình gọi thẳng trong request (FR-015), không qua hàng đợi job chung, nên bản production cũ không lấy mất.

## Câu hỏi còn mở

1. ~~Mỗi trình độ bao nhiêu chủ đề?~~ Đã chốt 2026-10-23: ít nhất 7, AI soạn 7–10 chủ đề; mỗi chủ đề 3 bài chính + 4–5 bài
   thêm (A2 trở lên khoảng 49–80 bài mỗi trình độ).
2. ~~Có bỏ khung gợi ý chủ đề ở trang Chủ đề không?~~ Đã chốt 2026-10-23: bỏ (FR-020).
