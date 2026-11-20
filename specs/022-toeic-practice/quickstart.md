# Quickstart: kiểm tra Luyện thi TOEIC (F27)

## Chạy

```powershell
# backend (thư mục backend/, đọc backend/.env)
go run ./cmd/server
# frontend (thư mục frontend/)
npm run start:dev
```

Không cần biến môi trường mới. Vẽ ảnh Part 1 dùng cấu hình ảnh hiện có (Gemini; Codex chỉ khi `CODEX_IMAGES=true` ở máy local).

## Nhập nội dung mẫu

1. Đăng nhập admin → **TOEIC** (`/admin/toeic`) → **Nhập JSON** → chọn `content/toeic/part5/…json` → **Chạy thử** → xem báo
   cáo → **Nhập** → các cụm ở dạng Nháp → **Đăng**.
2. Tạo đề: **Tạo đề** → dán `content/toeic/tests/luna-test-01.json` → trang đề hiện số câu mỗi Part, ảnh thiếu → **Vẽ ảnh
   còn thiếu…** (chọn Gemini/Codex) → **Đăng đề**.

## Test tự động

```powershell
cd backend; go test ./internal/toeic/... ./internal/storage/... ./internal/export/...
cd frontend; npm test; npm run lint
```

## Kiểm tra thủ công (360 px và ≥ 1024 px, sáng và tối)

1. **Nhập lỗi** (US1): sửa một `answer` thành 5 → Chạy thử báo `groups[0].questions[0].answer`; Nhập bị từ chối, danh sách
   không đổi. Cụm Nháp không có trong Luyện theo Part của người học.
2. **Đề chưa đủ**: đề thiếu câu Part 3 hoặc thiếu ảnh Part 1 → nút Đăng đề khoá, trang ghi rõ thiếu gì.
3. **Thư viện** (US2): chạm **Luyện** → ô **TOEIC** → thấy thẻ đề, tab, cột mục tiêu (2 lần chạm).
4. **Luyện tập Part 5 + 6, 20 phút**: chọn đáp án, đánh dấu 2 câu, nhảy câu bằng bảng câu. DevTools → Network: phản hồi
   `GET /api/toeic/attempts/{id}` không có `answer`/`explainVi`. Nộp khi còn câu chưa làm → hỏi lại.
5. **Làm tiếp trên máy khác**: đóng tab khi còn ~10 phút, mở bằng trình duyệt khác → đúng câu, thời gian lệch ≤ 5 giây.
6. **Kết quả**: số đúng/sai/bỏ qua, bảng phân loại câu hỏi theo Part (bấm số → đáp án chi tiết), danh sách đáp án; luyện
   vài Part thì không có điểm thang 990.
7. **Đáp án chi tiết**: văn bản có đoạn tô vàng, bản dịch, giải thích, "+ Sổ từ" → từ vào thẻ ôn.
8. **Sổ câu sai** (US3): câu sai có trong sổ; làm lại đúng 2 lần liên tiếp → ra khỏi sổ; sai lại → về 0/2.
9. **Phần Nghe** (US4): luyện Part 1–4 → Directions đọc lên (bỏ qua được), Part 1 chỉ ảnh + A–D, Part 2 chỉ "Mark your
   answer" + A–C, Part 3 hai giọng khác nhau (máy có nhiều giọng), tự sang cụm sau, tạm dừng/nghe lại/đổi tốc độ được; biểu
   đồ rõ ở 360 px tối. Sau khi nộp: transcript, nghe lại cụm.
10. **Full test** (US5): không tạm dừng/tua phần Nghe, tab Part 5–7 khoá; đóng app giữa phần Nghe → mở lại phát từ đầu cụm;
    sang phần Đọc đồng hồ 75 phút; nộp → điểm Listening/Reading/tổng ghi "ước tính". Làm lại đề → thẻ đề hiện lần mới, biểu
    đồ vẫn dùng lần đầu.
11. **Nhập lại để sửa** (FR-040a): đổi đáp án của một cụm đã có người làm, nhập với "thay thế" → kết quả bài cũ đổi, sổ câu
    sai cập nhật.
12. **Dạng câu, thống kê, mục tiêu** (US6): đặt mục tiêu 750 + ngày thi → thư viện hiện số ngày còn lại; Luyện theo dạng câu
    xếp yếu nhất trước, có link sang điểm ngữ pháp; thống kê có đường mục tiêu.
13. **Khách**: làm được Mini Test đầu tiên và luyện Part 1, 2, 5; đề khác, Part khác, sổ câu sai, thống kê hiện khoá;
    `POST /api/toeic/attempts` với đề khác → `403 members_only`.
14. **Mất mạng** khi đang làm (DevTools → Offline): vẫn chọn đáp án; bật mạng lại → tiến độ được gửi; nộp khi mất mạng báo
    thử lại, không mất bài.
15. **Chuỗi ngày học**: nộp một lượt → Trang chủ tính hôm nay là ngày học.
16. **Xuất dữ liệu / xoá tài khoản**: file xuất có `toeicAttempts`, `toeicMistakes`, `toeicGoals`; xoá tài khoản → không còn
    bản ghi của người đó.
17. **Sử dụng AI**: sau khi vẽ ảnh Part 1, trang Sử dụng AI có op `toeic_photo`.
