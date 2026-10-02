# Kiểm tra giai đoạn 1

Gom toàn bộ việc kiểm tra thủ công còn tồn của 11 spec (`specs/001` → `specs/011`), sắp theo từng màn hình. Mỗi mục ghi `[spec › bước]` để mở `quickstart.md` tương ứng khi cần lệnh chi tiết. Làm xong thì tích task trong `tasks.md` của spec đó và tiêu chí trong [phases/giai-doan-1.md](phases/giai-doan-1.md).

`COMPOSE="docker compose -f deploy/docker-compose.yml"`

## Chưa có trong giai đoạn 1 (cần quyết định)

| Thiếu | Ảnh hưởng | Gợi ý |
|---|---|---|
| Quên mật khẩu | Chỉ có một tài khoản quản trị viên; quên mật khẩu là mất quyền quản trị | Lệnh CLI `reset-password` chạy trong container, không cần email |
| Truy cập từ điện thoại khi ra ngoài | Trong mạng nhà vào được qua `http://<IP máy>:8000`; ra ngoài thì không | Tailscale (miễn phí, chỉ thiết bị của bạn) |
| Nội dung bài học thật | Chưa có bài để học hằng ngày | Tự soạn vài chủ đề A1, hoặc làm F7 (AI sinh bài) |

---

## 0. Chuẩn bị

- [ ] `./deploy/fetch-dictionary.sh` (hoặc `.ps1`); `deploy/.env` có `GEMINI_API_KEY`.
- [ ] **Sao lưu dữ liệu hiện có trước khi làm gì khác:** `$COMPOSE run --rm -e BACKUP_NOW=1 -e BACKUP_ONCE=1 backup`.
- [ ] `$COMPOSE up --build -d`, mở <http://localhost:8000>.
- [ ] Hai tài khoản: `admin@example.com` (quản trị viên) và `hoc@example.com` (người học).
- [ ] Dữ liệu mẫu: chủ đề **A1 · Gia đình** (≥ 4 bài) và **A1 · Mua sắm** (≥ 1 bài), mọi bài có chú thích "Xong". Một bài dùng nội dung mẫu tách câu `[003 › 1]`, một bài dùng nội dung mẫu tra từ `[004 › chuẩn bị]`, một bài dùng nội dung mẫu chép chính tả `[005 › chuẩn bị]`.

## 1. Hệ thống (F0)

- [ ] Một lệnh chạy toàn bộ; trang chủ không lỗi `[001 › 1]`.
- [ ] Tắt `mongo` → chân trang báo lỗi cơ sở dữ liệu, không trắng trang; bật lại thì tự hết `[001 › 2]` `[009 › 3]`.
- [ ] Tắt `backend` → "Không kết nối được máy chủ", header vẫn dùng được `[001 › 3]`.
- [ ] Chạy riêng backend + `npm start`; sửa `features/home` → trình duyệt tự cập nhật `[001 › 10]`.
- [ ] Thiếu `MONGO_URI` → báo lỗi và thoát mã 1; `git ls-files` không có `.env` `[001 › 8]`.
- [ ] Ctrl+C khi đang có request → request trả đủ, backend thoát mã 0 `[001 › 9]`.
- [ ] Ngắt mạng: `npm test`, `npm run lint`, `go test ./...`, `make lint` vẫn chạy `[001 › 7]`.
- [ ] Ngắt mạng (DevTools › Offline), tải lại: chữ tiếng Việt vẫn hiện bằng Lexend, không request tới domain ngoài `[001 › 6]`.

## 2. Giao diện sáng và tối (F0, F12)

- [ ] Lần đầu mở (xoá site data): app theo chế độ của thiết bị; nút giao diện chỉ hiện một chế độ, bấm vào mở menu Sáng/Tối `[001 › 4]` `[010 › 1.2]`.
- [ ] Chọn Tối → đổi ngay; tải lại không nháy nền sáng `[001 › 4]`.
- [ ] Cửa sổ ẩn danh chặn site data: chọn Tối vẫn chạy, không lỗi console `[001 › 4]`.
- [ ] Cài đặt → Tối; mở trình duyệt khác, đăng nhập → tự chuyển tối; đăng xuất → trang đăng nhập vẫn tối `[010 › 1.3]`.
- [ ] Nút nhanh trên header và trang Cài đặt luôn khớp nhau `[010 › 1.4]`.

## 3. Tài khoản (F1)

- [ ] Đóng hẳn trình duyệt, mở lại → vẫn đăng nhập; cookie `luna_session` HttpOnly, SameSite Strict, 30 ngày `[002 › 2]`.
- [ ] Đăng xuất ở tab 1 → tab 2 tải lại thì về `/login` `[002 › 2]`.
- [ ] Xoá phiên trong DB khi đang ở `/admin` → về `/login?returnUrl=%2Fadmin`, đăng nhập xong quay lại `/admin`; `returnUrl` ra ngoài → về `/` `[002 › 3]`.

## 4. Quản trị: chủ đề, bài học, lộ trình (F2, F14)

- [ ] Kiểm tra kết quả chuyển dữ liệu cũ: mọi bài có `topicId`, không còn `db.roadmap`, `db.migrations` có `f14-topics`; restart backend dữ liệu không đổi `[007 › 0.3–0.4]`.
- [ ] Chủ đề: thêm, trùng tên cùng trình độ bị chặn, khác trình độ thì được; xoá chủ đề còn bài bị chặn `[007 › 1]`.
- [ ] Form bài bắt buộc chọn chủ đề; đổi chủ đề của bài đang ở lộ trình → bài xuống cuối lộ trình mới; lọc theo trình độ và chủ đề `[007 › 2]`.
- [ ] Sau khi lưu bài: trạng thái tự đổi "Xong" trong ≤ 10 giây; bấm ▶ từng câu nghe được (giọng đọc của trình duyệt) `[003 › 2.1–2.3]`.
- [ ] Không có key AI → chú thích "Lỗi"; thêm key, Chạy lại → "Xong", log `ai request` tăng đúng 1 `[003 › 2.4]`.
- [ ] ~~Tắt Kokoro → audio "Lỗi"…~~ Bỏ từ 2026-10-02 (không còn sinh audio; nghe bằng giọng đọc của trình duyệt).
- [ ] Sửa chú thích: sửa, thêm cụm có trong bài, cụm không có bị báo lỗi, xoá; nhãn "Đã sửa tay" `[003 › 4]`.
- [ ] Lộ trình theo chủ đề: kéo thả bằng tay nắm, tải lại giữ thứ tự; Lên/Xuống bằng bàn phím; cảnh báo "còn dưới 3 bài" theo từng chủ đề; không thêm trùng `[003 › 3]` `[007 › 3]`.

## 5. Người học: chọn chủ đề và bài hôm nay (L)

- [ ] Trang chủ → chọn A1 → danh sách chủ đề kèm số bài → chọn "Gia đình" → bài hôm nay là bài 1 `[008 › 1]`.
- [ ] Ôn ● · Đọc 🔒 · Nghe 🔒; ôn xong mở Đọc; gõ URL bài 2 → "Bài này sẽ mở khi tới lượt" `[008 › 2.1–2.2]`.
- [ ] Đọc tới giữa bài, tải lại → đúng vị trí; Nghe tới câu 3, đóng tab, mở lại → câu 3 `[008 › 2.3–2.4]`.
- [ ] Xong bài → "Đã xong bài hôm nay", streak 1, mục tiêu 1/N; mở lại cùng ngày không có bài mới, ôn tự do vẫn vào được `[008 › 2.4, 3.1]`.
- [ ] Không có thẻ đến hạn → bước Ôn tự xong `[008 › 2.5]`.
- [ ] Lùi ngày: bài 2 thành bài hôm nay, streak đúng; nghỉ 2 ngày → vẫn 1 bài, streak 0 `[008 › 3.2]`.
- [ ] 45 thẻ đến hạn → hôm nay ôn 30, hôm sau 15 `[008 › 3.4]`.
- [ ] Hết bài trong lộ trình → "Chưa có bài mới" + ôn tự do + chọn chủ đề khác `[008 › 3.3]`.
- [ ] Đổi chủ đề trước khi bắt đầu → đổi ngay; sau khi xong bước Ôn → "bắt đầu từ ngày mai"; quay lại chủ đề cũ → học tiếp đúng bài, streak không đổi `[008 › 4]`.
- [ ] Trang **Bài học**: Hôm nay, Đã học (mở lại không đổi tiến độ), Sắp tới 🔒 `[008 › 5]`.

## 6. Bước Đọc và Từ vựng của bài (F3, F5)

- [ ] Tra *went* → nhãn AI; *park* → nhãn Từ điển + IPA; *studies/running/laughing* ra dạng gốc `[004 › 1.1–1.3]`.
- [ ] Mỗi lần tra < 300ms, log `ai request` không tăng `[004 › 1.4]`.
- [ ] Lưu từ → ✓, tô màu ở mọi bài; lưu trùng không tạo thẻ mới; từ không có nghĩa → tự nhập `[004 › 2]`.
- [ ] Bôi đen cụm (máy tính kéo, điện thoại chạm giữ); chọn một phần từ; chọn vắt hai câu; 7 từ → báo tối đa 6 `[004 › 3]`.
- [ ] Phát âm ▶ không chồng tiếng; trình duyệt không đọc được → "Chưa đọc được từ này" `[004 › 4]`.
- [ ] "Đã đọc xong": bài dài bật khi cuộn tới cuối, bài ngắn bật ngay `[004 › 5]`.
- [ ] Mục Từ vựng: Lưu một từ, Lưu tất cả hai lần không trùng, lưu từ popup cập nhật ✓, bài chưa có chú thích → "Chưa có danh sách từ vựng" `[006 › 1]`.
- [ ] Tài khoản khác không thấy từ của tài khoản trước `[004 › 6]`.

## 7. Bước Nghe (F4)

- [ ] "Câu 1/N", câu trước/sau/lặp lại (< 1 giây, không chồng tiếng), tốc độ giữ khi đổi câu, ẩn/hiện chữ `[005 › 1]`.
- [ ] Chép chính tả đúng bảng mẫu (thiếu dấu nháy, thiếu từ, thừa từ, hoa thường, dấu câu, ô trống) `[005 › 2]`.
- [ ] Bật thang xám (Emulate achromatopsia) → vẫn phân biệt sai và thiếu `[005 › 2]`.
- [ ] Xong hết câu (có câu sai) → hoàn thành + tỷ lệ; tải lại giữ kết quả; tài khoản khác 0/N; sửa nội dung bài → làm lại từ 0 `[005 › 3]`.
- [ ] Offline khi kiểm tra một câu → "Chưa lưu được, sẽ thử lại"; bật mạng → lưu đủ `[005 › 5]`.

## 8. Sổ từ và ôn tập (F5)

- [ ] Nhóm Hôm nay / Hôm qua / ngày; tìm "giv"; lọc theo bài và "Thẻ tự thêm" `[006 › 2.1–2.2]`.
- [ ] Thêm thẻ tay; thêm trùng bị chặn; sửa nghĩa giữ `due` và `reps`; xoá thẻ xoá cả lịch sử ôn `[006 › 2.3–2.5]`.
- [ ] Ôn kiểu Xem từ đoán nghĩa: 4 nút có khoảng thời gian; Space lật, phím 3 = Good; Again hiện lại cuối phiên `[006 › 3.2–3.3]`.
- [ ] Ôn kiểu Nghe rồi gõ: "WENT" đúng, "want" sai và hiện từ đúng `[006 › 3.4]`.
- [ ] Hết thẻ → "Không có thẻ nào đến hạn" + thời điểm sớm nhất; offline khi đánh giá rồi Thử lại → chỉ tính 1 lần `[006 › 3.5–3.6]`.
- [ ] Đổi múi giờ → nhóm ngày và hạn thẻ mới theo múi giờ mới `[006 › 4]`.

## 9. Trang chủ và thống kê (F6)

- [ ] 360 × 640: thấy streak, thanh mục tiêu "A1 · Gia đình x/y", thanh Đọc và Nghe, bài hôm nay, nút Bắt đầu/Tiếp tục mà không cuộn; tên bài dài xuống dòng `[009 › 1]`.
- [ ] Nút Tiếp tục đưa đúng bước, đúng câu; số liệu cập nhật ngay sau mỗi bước, không cần tải lại `[009 › 1.3–1.4, 2]`.
- [ ] Trạng thái đặc biệt: tài khoản mới, xong bài hôm nay, hết lộ trình `[009 › 3]`.
- [ ] "Ngày mai: N thẻ" đúng (thẻ 23:00 ngày mai tính, 01:00 ngày kia không tính) `[009 › 4]`.
- [ ] Trang Thống kê khớp số trong DB; tài khoản mới hiện 0 và "—" `[009 › 5]`.

## 10. Cài đặt (F12)

- [ ] Giới hạn thẻ 10 → bước Ôn 10 thẻ; nhập 4, 201, trống → lỗi, không lưu; giảm rồi tăng giới hạn tính đúng phần còn lại `[010 › 2]`.
- [ ] Đổi múi giờ khi đang học dở → vẫn đúng bài và vị trí; đổi sang múi giờ "hôm qua" → không mở bài mới, streak không đổi `[010 › 3]`.

## 11. Sao lưu, khôi phục, xuất dữ liệu (F13)

Làm **cuối cùng**, vì bước khôi phục xoá database.

- [ ] Sao lưu ngay tạo file; log service backup có "next backup at … 03:00"; xoay vòng giữ đúng 7 file `luna-*`, không xoá file khác `[011 › 1.1–1.3]`.
- [ ] Tắt mongo khi sao lưu → log lỗi, không file mới, không xoá file cũ, app vẫn chạy `[011 › 1.4]`.
- [ ] Đếm document → xoá DB → khôi phục theo README → số document khớp, đăng nhập thấy đủ dữ liệu `[011 › 2.1–2.3]`.
- [ ] Khôi phục file không tồn tại hoặc file hỏng → báo lỗi, dữ liệu không đổi `[011 › 2.4]`.
- [ ] Xuất dữ liệu: đúng tài khoản, không có `passwordHash`/`tokenHash`; tài khoản khác không lẫn dữ liệu; offline → báo lỗi, bấm lại được `[011 › 3]`.

## 12. Vòng cuối: 360px, sáng và tối, bàn phím

DevTools 360px, đi qua **mọi màn hình** ở cả hai chế độ. Mỗi màn hình: không cuộn ngang, đủ tương phản (Lighthouse Accessibility), trạng thái không chỉ bằng màu, dùng được bằng bàn phím, nút chạm ≥ 44px.

- [ ] `/login`, `/register`, `/forbidden`, header hai dòng `[002 › 7]`
- [ ] Trang chủ, trạng thái kết nối `[001 › 5]` `[009 › 1.5]`
- [ ] Quản trị: danh sách, form, chi tiết và bảng chú thích, chủ đề, lộ trình, hộp thoại `[003 › 7]` `[007 › 5]`
- [ ] Chọn mục tiêu, `/today` cả ba bước, trang Bài học `[008 › 6]`
- [ ] Bước Đọc: popup gần mép màn hình, IPA hiển thị đúng ký tự `[004 › 7]`
- [ ] Bước Nghe: bàn phím ảo mở vẫn thấy nút Kiểm tra `[005 › 4]`
- [ ] Sổ từ, form thẻ, phiên ôn hai kiểu, mục Từ vựng `[006 › 5]`
- [ ] Cài đặt, nút Xuất dữ liệu `[010 › 1]` `[011 › 3.5]`

## 13. Đóng giai đoạn

- [ ] Tích task thủ công trong `tasks.md` của 11 spec.
- [ ] Tích 56 tiêu chí nghiệm thu trong `docs/phases/giai-doan-1.md`.
- [ ] Cập nhật trạng thái trong `docs/spec-inputs/README.md` thành ✅.
