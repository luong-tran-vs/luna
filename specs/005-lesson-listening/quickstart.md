# Quickstart: kiểm tra F4

Chuẩn bị: `docker compose -f deploy/docker-compose.yml up --build -d`; một bài đã có audio "Xong", ví dụ nội dung:

```text
I don't like green apples. We went to the park at 9.30. It's a well-known place. Why?
```

Mở `/admin` → bài → **Mở bước Nghe** (hoặc `/lessons/<id>/listen`).

## 1. Nghe từng câu (US1)

1. Thấy "Câu 1/4", câu trước bị vô hiệu, tốc độ 1x, chữ đang ẩn.
2. Phát, câu sau, câu trước, lặp lại nhiều lần (mỗi lần phát lại từ đầu, < 1 giây).
3. Chọn 0.75x rồi chuyển câu: vẫn 0.75x. Bấm "Hiện chữ" / "Ẩn chữ".
4. Chuyển câu khi đang phát: câu cũ dừng, không chồng tiếng.

## 2. Chép chính tả (US2, SC-001, SC-002)

Với câu 1 "I don't like green apples.":

| Gõ | Mong đợi |
|---|---|
| `i dont like green apples` | *dont* gạch ngang → **don't**; còn lại đúng; 4/5 |
| `I don't like apples` | *green* gạch chân chấm (thiếu); 4/5 |
| `I don't like the green apples` | *the* gạch ngang (thừa); 5/6 |
| `i DON'T like green apples!!` | toàn đúng; 5/5 |
| (trống) | "Hãy gõ câu bạn nghe được", chưa tính |

Câu 2 `we went to the park at 9 30` → *9 30* sai so với *9.30*; câu 3 `its a well known place` → *its* sai, *well known* đúng.
Bật chế độ thang xám của hệ điều hành (hoặc DevTools › Rendering › Emulate vision deficiencies › Achromatopsia): vẫn phân
biệt sai/thiếu.

## 3. Hoàn thành và lưu (US3, SC-004, SC-005, SC-006)

1. Kiểm tra hết 4 câu (có câu sai) → "Đã hoàn thành bước Nghe" + tỷ lệ đúng.
2. Tải lại trang → các câu vẫn hiện kết quả, "Đã kiểm tra 4/4", tỷ lệ không đổi.
3. Đăng nhập tài khoản khác, mở cùng bài → "Đã kiểm tra 0/4".
4. `curl -b "luna_session=<token>" http://localhost:8000/api/lessons/<id>/dictation/summary` → đúng số liệu.
5. Sửa nội dung bài ở trang quản trị → mở lại bước Nghe → bắt đầu lại từ 0.

## 4. Điện thoại 360px

Mở trên điện thoại thật (hoặc DevTools 360px): chạm ô gõ, bàn phím hiện, nút Kiểm tra vẫn thấy; Enter = Kiểm tra; không cuộn
ngang; cả sáng và tối.

## 5. Lưu thất bại

DevTools › Network › Offline, kiểm tra một câu → kết quả vẫn hiện, báo "Chưa lưu được, sẽ thử lại"; bật mạng, kiểm tra câu
tiếp → cả hai được lưu (summary đúng).
