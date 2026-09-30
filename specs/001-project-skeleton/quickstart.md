# Quickstart: kiểm tra F0

Kịch bản kiểm tra thủ công cho các tiêu chí nghiệm thu F0 không tự động hoá được. Lệnh viết cho
PowerShell hoặc bash, chạy từ gốc repo.

## 1. Chạy toàn bộ bằng một lệnh (SC-001)

```bash
docker compose -f deploy/docker-compose.yml up --build
```

Mở <http://localhost:8000>. **Mong đợi**: thanh trên cùng có chữ "Luna"; trang chủ hiện "Đã kết nối".

## 2. Mất kết nối cơ sở dữ liệu (SC-002)

```bash
docker compose -f deploy/docker-compose.yml stop mongo
```

Tải lại trang. **Mong đợi**: "Mất kết nối cơ sở dữ liệu" trong ≤ 3 giây, trang không trắng.

```bash
docker compose -f deploy/docker-compose.yml start mongo
```

Tải lại trang. **Mong đợi**: lại "Đã kết nối", không cần khởi động lại backend.

## 3. Không kết nối được máy chủ

```bash
docker compose -f deploy/docker-compose.yml stop backend
```

Tải lại trang. **Mong đợi**: "Không kết nối được máy chủ"; header và nút chọn giao diện vẫn dùng được.
Sau đó `start backend`.

## 4. Giao diện Sáng / Tối (SC-003)

1. Lần đầu mở (xoá site data trước): nút giao diện hiện chế độ của thiết bị, màu khớp cài đặt thiết bị. Bấm nút → menu chỉ có Sáng và Tối.
2. Chọn Tối → đổi ngay. Tải lại → vẫn Tối, không nháy nền sáng.
3. Khi chưa chọn lần nào (xoá site data), đổi chế độ sáng/tối của hệ điều hành (hoặc DevTools › Rendering › Emulate
   `prefers-color-scheme`) → app đổi theo không cần tải lại.
4. Mở cửa sổ ẩn danh chặn cookie/site data → chọn Tối vẫn có hiệu lực, không lỗi console.

## 5. 360px và độ tương phản (SC-004)

DevTools › Device toolbar, rộng 360px, lần lượt ở chế độ Sáng và Tối:
- Không có thanh cuộn ngang; header, nút chọn giao diện, thẻ trạng thái vừa màn hình.
- Lighthouse › Accessibility hoặc DevTools CSS Overview: không có lỗi tương phản.
- Trạng thái kết nối có biểu tượng + chữ (không chỉ màu).

## 6. Font offline (SC-005)

DevTools › Network › Offline, tải lại (bản đã build, phục vụ từ nginx hoặc cache): chữ "Đang kiểm tra
kết nối…", "Mất kết nối cơ sở dữ liệu" hiển thị bằng Lexend với dấu đúng. Tab Network không có request
nào tới domain ngoài.

## 7. Test và lint không cần mạng (SC-005)

Sau khi đã `npm ci`, `go mod download` và pull image golangci-lint một lần, ngắt mạng rồi chạy:

```bash
cd frontend; npm test -- --watch=false; npm run lint
cd backend; go test ./...; make lint   # hoặc lệnh docker tương đương ghi trong README
```

**Mong đợi**: tất cả chạy xong, không lỗi.

## 8. Thiếu cấu hình bắt buộc (SC-007)

```bash
cd backend; go run ./cmd/api        # không đặt MONGO_URI
```

**Mong đợi**: in `config: MONGO_URI is required`, thoát mã 1. Kiểm tra `git ls-files` không có `.env`.

## 9. Tắt an toàn (SC-006)

1. Chạy backend (local hoặc container) với một handler chậm tạm thời, hoặc dùng `LOG_LEVEL=debug` và gửi
   `curl http://localhost:8080/api/health` khi Mongo đang tắt (mất ~2 giây).
2. Trong lúc request đang chạy, nhấn Ctrl+C (hoặc `docker compose stop backend`).
3. **Mong đợi**: curl nhận đủ phản hồi 503 JSON; log có dòng shutdown sau dòng request; tiến trình thoát
   mã 0.

## 10. Chạy riêng từng phần khi phát triển

```bash
docker compose -f deploy/docker-compose.yml up -d mongo
cd backend; $env:MONGO_URI="mongodb://localhost:27017"; go run ./cmd/api   # bash: MONGO_URI=... go run ./cmd/api
cd frontend; npm start                                                      # http://localhost:4200, /api proxy tới :8080
```

Sửa `features/home` → trình duyệt tự cập nhật.
