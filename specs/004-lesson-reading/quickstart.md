# Quickstart: kiểm tra F3

Chuẩn bị: `./deploy/fetch-dictionary.sh` (hoặc `.ps1`), rồi `docker compose -f deploy/docker-compose.yml up --build -d`.
Có một bài đã có chú thích (cần `GEMINI_API_KEY`, hoặc tự thêm chú thích ở trang quản trị), ví dụ nội dung:

```text
We went to the park yesterday. He gave up smoking last year.
She studies English every morning. The children were running and laughing.
```

Chú thích mẫu (tự thêm nếu không có AI): *went → go "đã đi"*, *gave up → give up "từ bỏ"*.

Mở trang chi tiết bài ở `/admin` → **Mở bước Đọc** (hoặc `/lessons/<id>/read`).

## 1. Tra một từ (US1, SC-001, SC-002)

1. Chạm *went* → popup: *went → go*, "đã đi", nhãn **AI · theo ngữ cảnh**.
2. Chạm *park* → nghĩa từ điển, IPA `/ˈpɑːrk/`, nhãn **Từ điển**.
3. Chạm *studies* → nghĩa của *study*; *running* → *run*; *laughing* → *laugh*; *children* → mục từ điển.
4. DevTools › Network: mỗi lần tra một request `lookup` < 300ms; không có request nào tới Gemini
   (`docker compose … logs backend | grep -c "ai request"` không đổi).
5. 360px: popup không che từ, nằm trọn màn hình; chạm ra ngoài hoặc Esc thì đóng.

## 2. Lưu từ (US2, SC-003, SC-004)

1. Popup *went* → **Lưu vào sổ từ** → nút thành **✓ Đã có trong sổ**; *went* được tô màu và gạch chân.
2. Chạm *go* ở bài khác (hoặc *goes*) → **✓ Đã có trong sổ**.
3. `curl -b "luna_session=<token>" http://localhost:8000/api/vocab/words` → 1 mục `go`.
4. Mở bài khác có *went*/*go* → được tô màu.
5. Chạm một từ không có nghĩa (ví dụ tên riêng lạ) → "Chưa có nghĩa", ô nhập; lưu khi trống → lỗi; nhập nghĩa → lưu được.

## 3. Tra cụm (US3)

1. Bôi đen *gave up* (điện thoại: chạm giữ rồi kéo) → popup cụm, nhãn AI. Lưu → cả cụm được tô.
2. Chọn "ave u" → vẫn tra *gave up*. Chọn vắt qua hai câu → chỉ lấy phần câu đầu. Chọn 7 từ → "Chọn tối đa 6 từ".
3. *look forward to* (không có chú thích) → thường "Chưa có nghĩa" vì từ điển không chứa cụm (research R1).

## 4. Phát âm (US4, SC-007)

Bấm ▶ trong popup → nghe; bấm lại ngay → phát từ đầu, không chồng. Lần hai cùng từ phát ngay.
`docker compose … stop kokoro`, bấm ▶ một từ mới → "Chưa phát được âm thanh", popup vẫn dùng được; `start kokoro`.

## 5. Đã đọc xong (US5, SC-005)

Bài dài: nút vô hiệu tới khi cuộn tới cuối; cuộn lên lại vẫn bật; bấm → "Đã hoàn thành bước Đọc". Bài ngắn: bật ngay.

## 6. Quyền và dữ liệu riêng

Đăng nhập tài khoản khác → `/api/vocab/words` không có từ của tài khoản trước. Không cookie → 401 cho mọi endpoint mới.

## 7. 360px, sáng và tối

Bài đọc, popup (IPA hiển thị đúng ký tự ˈ ɑ ː), từ đã lưu (màu + gạch chân), nút Đã đọc xong: không cuộn ngang, đủ tương
phản.
