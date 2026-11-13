# Quickstart: kiểm tra Trò chơi (F26)

## Chạy

```powershell
# backend (thư mục backend/, đọc backend/.env)
go run ./cmd/server
# frontend (thư mục frontend/)
npm run start:dev
```

Không cần biến môi trường mới. Soạn câu Bắn lỗi bằng AI dùng cấu hình AI hiện có (`AI_PROVIDER`, khoá Gemini).

## Test tự động

```powershell
cd backend; go test ./internal/games/... ./internal/vocab/... ./internal/verbs/... ./internal/storage/...
cd frontend; npm test; npm run lint
```

## Kiểm tra thủ công (360 px, sáng và tối)

1. **Lối vào**: chạm nút giữa **Luyện** → ô "Trò chơi" → thấy danh sách trò, tab Luyện đang được tô.
2. **Ảnh nhanh tay** (tài khoản đã học ≥ 1 bài): chơi hết 60 giây, chọn sai ít nhất một từ có thẻ ôn. Màn kết quả có điểm,
   số đúng/sai, kỷ lục, từ đã sai. Mở **Ôn tập** → từ đã sai có trong lượt hôm nay.
3. **Thoát giữa lượt**: không có kỷ lục mới, từ đã sai vẫn đến hạn.
4. **Tạm dừng**: chuyển sang tab khác của trình duyệt rồi quay lại → màn "Tạm dừng", đồng hồ không chạy trong lúc ẩn.
5. **Đấu động từ**: gõ sai một động từ đã học → mất 1 mạng, hiện đáp án. Mở trang Động từ → từ đó đến hạn ôn hôm nay.
   Gõ đúng một động từ chưa đến hạn → hạn ôn không đổi.
6. **Chữ rơi**: chạm sai chữ → mất mạng; qua màn 1 → chữ rơi nhanh hơn. Bật "giảm chuyển động" của hệ điều hành → chữ đi
   từng bước, vẫn chơi được.
7. **Lật thẻ** từ màn xong bài → 6 cặp là từ của bài đó.
8. **Bắn lỗi** (thành viên): chạm chữ sai → 3 phương án; chọn đúng → quy tắc và nhãn chủ điểm. Tài khoản khách: ô Bắn lỗi
   bị khoá, `GET /api/games/grammar/round` trả `403 members_only`.
9. **Admin**: Ngữ pháp → cột "Câu Bắn lỗi" đánh dấu điểm < 10 câu → mở điểm → "Nhờ AI soạn thêm" → câu nháp chưa xuất hiện
   trong trò cho đến khi bấm Duyệt. Trang **Sử dụng AI** có op `grammar_spot`.
10. **Thử thách hằng ngày**: Trang chủ có thẻ thử thách → chơi xong → chuỗi +1, có so sánh với hôm qua/7 ngày/kỷ lục,
    "Chia sẻ" sao chép chữ dạng `Luna 13/11 · 9/10 🔥13` + 🟩/🟥. Mở lại → thấy kết quả đã tính; chơi lại không đổi điểm.
    Trang chủ: hôm nay là ngày học.
11. **Mất mạng** khi gửi kết quả (DevTools → Offline): báo nhẹ, mở lại trang Trò chơi khi có mạng → kết quả được gửi.
12. **Xuất dữ liệu** (Cài đặt → Sao lưu): file có `gameBests`, `dailyChallenges`. Xoá tài khoản → hai collection không
    còn bản ghi của người đó.
