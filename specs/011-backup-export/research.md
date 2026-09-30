# Research: Sao lưu và xuất dữ liệu (F13)

Không còn mục NEEDS CLARIFICATION.

## R1. Chạy sao lưu ở đâu

- **Decision**: Service `backup` riêng trong `deploy/docker-compose.yml`, image `mongo:8` (đã dùng cho `mongo`), `entrypoint: ["bash",
  "/backup/backup.sh"]`, `restart: unless-stopped`, volumes `./backup:/backup:ro` (script) và `./backups:/backups` (bản sao lưu),
  env `MONGO_URI` (mặc định `mongodb://mongo:27017`), `MONGO_DATABASE` (`luna`), `TZ` (`${BACKUP_TZ:-Asia/Ho_Chi_Minh}`),
  `BACKUP_TIME` (`03:00`), `BACKUP_KEEP` (`7`), `BACKUP_NOW` (`0`).
- **Rationale**: Khối 2; đã kiểm tra image `mongo:8` là Ubuntu 24.04 có `/usr/share/zoneinfo`, GNU `date`, `bash`, `mongodump`/
  `mongorestore` 100.18 → không cần image mới. Container riêng nên sao lưu lỗi/treo không chạm backend (FR-003).
- **Alternatives**: Cron trên máy chủ (không chạy được giống nhau trên Windows/Docker Desktop); goroutine trong backend (backend phải
  có công cụ mongodump, lỗi sao lưu có thể ảnh hưởng app).

## R2. Lịch 03:00 và không trùng/không bỏ lỡ

- **Decision**: Vòng lặp trong `backup.sh`:
  1. Khi khởi động: xoá file tạm `*.tmp` còn sót (bản dở dang). Nếu `BACKUP_NOW=1` → sao lưu ngay. Ngược lại, nếu đã qua `BACKUP_TIME`
     hôm nay mà chưa có `luna-<hôm nay>.archive.gz` → sao lưu ngay (bù khi máy chủ khởi động lại sau 03:00).
  2. Tính thời điểm 03:00 kế tiếp bằng `date -d "today 03:00"` / `"tomorrow 03:00"` (theo `TZ`), `sleep` tới đó, sao lưu, lặp.
  Mỗi ngày một file tên theo ngày (`luna-YYYYMMDD.archive.gz`), nên chạy lại trong ngày chỉ thay bản của ngày đó (edge case).
- **Rationale**: Không cần cron; khởi động lại trước 03:00 vẫn chạy đúng giờ, sau 03:00 thì bù đúng một lần.
- **Alternatives**: `sleep 86400` cố định (trôi giờ, sai khi đổi giờ mùa hè).

## R3. Ghi bản sao lưu an toàn và xoay vòng 7 bản

- **Decision**: `mongodump --uri "$MONGO_URI" --db "$MONGO_DATABASE" --archive="$tmp" --gzip` với `tmp=/backups/luna-YYYYMMDD.archive.gz.tmp`;
  thành công và file không rỗng → `mv` thành tên thật (thay bản cùng ngày nếu có). Sau đó xoay vòng: liệt kê chỉ file khớp
  `luna-[0-9]{8}.archive.gz`, sắp theo tên (ngày), xoá những file ngoài 7 file mới nhất. Lỗi ở bất kỳ bước nào → xoá file tạm, log
  `ERROR`, **không** xoay vòng, chờ lần sau.
- **Rationale**: Bản dở dang không bao giờ mang tên thật nên không được tính và không đẩy bản cũ ra (edge case); file lạ trong thư mục
  không bị xoá.
- **Alternatives**: Ghi thẳng tên thật (bản hỏng có thể được giữ và làm mất bản tốt cũ nhất).

## R4. Log

- **Decision**: Mọi dòng log ra stdout (`docker compose logs backup`) dạng `2026-10-01T03:00:02+07:00 INFO backup done
  luna-20261001.archive.gz (4.2 MB, 3 s)` / `ERROR backup failed: <thông điệp mongodump>`; mỗi lần thức dậy ghi thời điểm lần tới.
- **Rationale**: FR-003 "ghi log rõ ràng" với thời điểm và nguyên nhân; cùng chỗ xem log với các service khác.

## R5. Khôi phục

- **Decision**: `deploy/backup/restore.sh <file>` (tên file trong `deploy/backups/` hoặc đường dẫn), chạy bằng
  `docker compose -f deploy/docker-compose.yml run --rm --entrypoint bash backup /backup/restore.sh luna-20260930.archive.gz`:
  1. Không có tham số / file không tồn tại → in cách dùng + danh sách bản hiện có, thoát mã 1.
  2. `gzip -t` kiểm tra file; hỏng → báo lỗi, thoát 1, không chạm DB.
  3. `mongorestore --uri ... --archive=<file> --gzip --drop --nsInclude "$MONGO_DATABASE.*"`; in số document mỗi collection sau khi
     xong (`mongosh`… hoặc tóm tắt của mongorestore).
  README: dừng backend trước (`docker compose stop backend`), khôi phục, bật lại; cảnh báo dữ liệu hiện tại bị thay; phiên đăng nhập
  cũng về lúc sao lưu.
- **Rationale**: Một lệnh (FR-006); kiểm tra trước để không xoá dữ liệu khi file hỏng (FR-007); `--drop` thay hẳn từng collection có
  trong bản sao lưu (không trộn lẫn).
- **Alternatives**: `mongorestore` không `--drop` (trộn dữ liệu, trùng khoá).

## R6. Phạm vi sao lưu

- **Decision**: Chỉ database `luna` (mọi collection, gồm `users`, `sessions`, `migrations`). Không sao lưu volume audio và file từ
  điển (khối 2: tạo lại được bằng job TTS / `fetch-dictionary`).
- **Rationale**: FR-005.

## R7. Dạng file xuất

- **Decision**: JSON (UTF-8, thụt lề 2) một object:
  `{version: 1, exportedAt, account, settings, cards, reviewLogs, goals, lessonProgress, studyDays, dictationResults, lessons}`.
  Mỗi mảng là tài liệu gốc trong MongoDB của người học, chuẩn hoá: `ObjectID` → chuỗi hex, ngày → RFC 3339, bỏ trường `userId`
  (thừa). `account` chỉ có `id`, `email`, `role`, `createdAt` (không bao giờ đọc `passwordHash`). `lessons`: các bài có trong
  `lesson_progress` của người học (bài đã bắt đầu), bỏ trường nội bộ không cần (`audio` đường dẫn file giữ lại dạng tên file là
  được); bài đã bị xoá → `{id, deleted: true}`.
- **Rationale**: Đầy đủ dữ liệu người học (FR-010) mà không phải viết mapping cho từng domain; dạng dễ đọc, dễ nhập lại sau này (ngoài
  phạm vi).
- **Alternatives**: Extended JSON của MongoDB (`{"$oid": …}`) — khó đọc cho người học; mapping qua service từng domain — nhiều code, dễ
  sót trường.

## R8. Repository xuất và cách lọc

- **Decision**: `export.Repository`:
  - `Account(ctx, userID) (Account, map[string]any /*settings*/, error)` — projection chỉ `email`, `role`, `createdAt`, `settings`,
    `timezone` (cũ).
  - `UserDocs(ctx, collection, userID) ([]map[string]any, error)` — chỉ nhận tên trong danh sách cho phép (`cards`, `review_logs`,
    `goals`, `lesson_progress`, `study_days`, `dictation_results`); lọc `{userId: <oid>}`; sắp theo `_id`.
  - `Lessons(ctx, ids) ([]map[string]any, error)`.
  Service không bao giờ gọi với tên collection khác (`sessions`, `users` bị từ chối ở cả service và repository).
- **Rationale**: Nguyên tắc V: lọc `userId` ở một chỗ; danh sách cho phép chặn lộ `sessions`.

## R9. Tên file và phản hồi

- **Decision**: `Content-Type: application/json; charset=utf-8`, `Content-Disposition: attachment; filename="luna-export-20260930.json"`
  với ngày theo múi giờ trong Cài đặt (F12), `Cache-Control: no-store`. Lỗi → JSON lỗi như mọi API (401/500).
- **Rationale**: FR-009; `no-store` vì chứa dữ liệu cá nhân.

## R10. Nút Xuất dữ liệu

- **Decision**: Nhóm **Dữ liệu** cuối trang Cài đặt: mô tả ngắn + nút **Xuất dữ liệu**. `ExportApiService.download()` gọi
  `GET /api/export` với `responseType: 'blob', observe: 'response'`, lấy tên file từ `Content-Disposition` (dự phòng
  `luna-export.json`), tạo `URL.createObjectURL` + thẻ `<a download>` tạm để trình duyệt tải, rồi thu hồi URL. Trong lúc xuất nút
  disabled + "Đang xuất…" (`aria-busy`); lỗi → `role="alert"` "Không xuất được dữ liệu, vui lòng thử lại.".
- **Alternatives**: `<a href="/api/export" download>` (không báo lỗi được, lỗi mở trang JSON).
