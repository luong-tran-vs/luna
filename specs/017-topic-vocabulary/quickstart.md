# Quickstart: kiểm tra F18 (Từ vựng theo chủ đề)

## Chuẩn bị

- `docker compose -f deploy/docker-compose.yml up --build -d`, đăng nhập quản trị viên.
- Muốn sinh bài thật: ghi `GEMINI_API_KEY` vào `deploy/.env`.

## Kiểm tra tự động

```bash
cd backend && CGO_ENABLED=1 GOTOOLCHAIN=local go test -race ./... && make lint
cd frontend && npx ng test --watch=false && npm run lint && npx ng build
```

## 1. Dữ liệu ban đầu (US1)

1. Lần khởi động đầu sau F18: `docker compose logs backend | grep "topic words seeded"` → số chủ đề nhận từ (42 với DB hiện có).
2. Trang **Chủ đề**: "A1 · Gia đình" hiện "Từ vựng: đã dùng X/Y"; "Chung" hiện "Chưa có từ vựng".
3. Sửa danh sách của một chủ đề (xoá một từ, Lưu), rồi `docker compose restart backend`: danh sách giữ nguyên. Xoá hết, khởi động lại:
   vẫn rỗng.

## 2. Sửa danh sách (US1)

1. Mở **Từ vựng** của "Gia đình". Từ có trong bài có nhãn "Đã dùng · n bài", từ khác "Chưa dùng".
2. Dán "cousin, nephew" và "Family" (trùng) vào ô thêm, **Lưu**: báo "Từ bị trùng" dưới "Family", không lưu gì. Bỏ "Family", **Lưu**:
   hai từ mới có trong danh sách.
3. Thêm "Intensive care unit (ICU)": báo lỗi ký tự. Thêm một từ 41 ký tự: báo "Tối đa 40 ký tự".
4. Tạo bài trong chủ đề có "grandmothers" và "took a shower": "grandmother", "take a shower" chuyển sang "Đã dùng".

## 3. Sinh bài theo từ mục tiêu (US2)

1. **Lộ trình** › "A1 · Gia đình" › **Sinh bài bằng AI**: có "Từ mục tiêu mỗi bài" = 8; chọn 3 bài → 3 nhóm 8 từ không trùng, toàn từ
   "Chưa dùng" (đối chiếu trang Từ vựng).
2. Bỏ một từ ở bài 1 (nút ✕), thêm một từ bằng ô gợi ý; thử gõ "banana" → báo không có trong danh sách.
3. Đổi số bài thành 2: các nhóm được chia lại.
4. `docker compose logs -f backend | grep '"op":"generate"'`, bấm **Sinh**: đúng 1 dòng log; mỗi bản nháp có "Dùng a/b từ mục tiêu", có
   "Còn thiếu: …" nếu thiếu.
5. Đặt 0 từ mục tiêu: không có nhóm từ, bản nháp không có dòng từ mục tiêu. Chủ đề "Chung": không có trường từ mục tiêu.
6. Đặt `AI_PROVIDER=none`, Sinh: báo lỗi AI, hộp thoại giữ số bài và nhóm từ.

## 4. Chú thích (US3)

1. Lưu một bản nháp (hoặc tạo bài tay có "father", "grandmother", "take a shower"). Chú thích xong: trang chi tiết bài quản trị có các
   từ đó trong danh sách chú thích; log có đúng 1 dòng `op=annotate`.
2. Với AI tắt, chú thích lỗi như cũ; bài thuộc "Chung" chú thích như cũ.

## 5. Chung

- Tài khoản người học: trang chủ, Khóa học, tiến độ, streak, thống kê không đổi; không thấy danh sách từ.
- 360px, sáng và tối: trang Chủ đề, trang Từ vựng, hộp thoại sinh bài không cuộn ngang. Làm toàn bộ bằng bàn phím.
