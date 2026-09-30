# Quickstart: kiểm tra F13

`COMPOSE="docker compose -f deploy/docker-compose.yml"`. Chuẩn bị: `$COMPOSE up --build -d`, có tài khoản `hoc@example.com` (có thẻ,
đã học bài) và `admin@example.com`.

Kiểm tra tự động: `cd backend && CGO_ENABLED=1 GOTOOLCHAIN=local go test -race ./...`; `cd frontend && npx ng test --watch=false &&
npm run lint && npx ng build`.

## 1. Sao lưu ngay và lịch (US1)

1. `$COMPOSE run --rm -e BACKUP_NOW=1 -e BACKUP_ONCE=1 backup` → log `INFO backup done luna-<hôm nay>.archive.gz`; file có trong
   `deploy/backups/`.
2. `$COMPOSE logs backup` (service chạy nền) → dòng "next backup at … 03:00 +07".
3. Xoay vòng: tạo 7 file giả cũ hơn `for d in 20260901 20260902 … 20260907; do cp deploy/backups/luna-<hôm nay>.archive.gz
   deploy/backups/luna-$d.archive.gz; done`, thêm `deploy/backups/ghi-chu.txt`, chạy lại bước 1 → còn đúng 7 file `luna-*`
   (20260901 bị xoá), `ghi-chu.txt` vẫn còn.
4. Lỗi: `$COMPOSE stop mongo`, chạy bước 1 → log `ERROR backup failed: …`, không file mới, không file nào bị xoá, không còn `.tmp`;
   app vẫn phục vụ trang đăng nhập (báo lỗi DB ở chân trang như F6). `$COMPOSE start mongo`.

## 2. Khôi phục (US2)

1. Ghi lại số document: `$COMPOSE exec mongo mongosh luna --quiet --eval 'db.getCollectionNames().sort().forEach(c =>
   print(c, db[c].countDocuments()))'`.
2. Khôi phục vào DB trống: `$COMPOSE exec mongo mongosh luna --quiet --eval 'db.dropDatabase()'`, rồi làm theo README:
   `$COMPOSE stop backend && $COMPOSE run --rm --entrypoint bash backup /backup/restore.sh luna-<hôm nay>.archive.gz && $COMPOSE
   start backend`.
3. Chạy lại lệnh đếm ở bước 1 → mọi collection khớp. Đăng nhập `hoc@example.com` → sổ từ, tiến độ như trước.
4. `restore.sh khong-co.archive.gz` → báo không tìm thấy + danh sách, mã 1; `echo hong > deploy/backups/luna-20260101.archive.gz`
   rồi khôi phục bản đó → báo hỏng, dữ liệu không đổi (xoá file giả sau đó).

## 3. Xuất dữ liệu (US3)

1. Đăng nhập `hoc@example.com` → Cài đặt → **Xuất dữ liệu** → tải về `luna-export-<ngày>.json`.
2. Mở file: có `account.email = hoc@example.com`, không có chuỗi `passwordHash` hay `tokenHash`; số `cards` bằng
   `db.cards.countDocuments({userId: <id hoc>})`; `lessons` là các bài trong `lessonProgress`.
3. Đăng nhập `admin@example.com` → xuất → file không có email/thẻ của `hoc`.
4. curl:

```bash
curl -s -D - -o export.json -b hoc.txt http://localhost:8000/api/export | grep -i "content-disposition\|cache-control"
grep -c passwordHash export.json   # 0
curl -s -o /dev/null -w "%{http_code}\n" http://localhost:8000/api/export   # 401
```

5. Tắt mạng (DevTools → Offline) → bấm **Xuất dữ liệu** → thông báo lỗi, bấm lại được.
