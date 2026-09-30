# Contract: Sao lưu và khôi phục (F13)

Chạy từ gốc repo. `COMPOSE="docker compose -f deploy/docker-compose.yml"`.

## Service `backup` (tự chạy cùng `$COMPOSE up -d`)

| Biến (`deploy/.env`) | Mặc định | Ý nghĩa |
| --- | --- | --- |
| `BACKUP_TZ` | `Asia/Ho_Chi_Minh` | múi giờ của giờ sao lưu và ngày trong tên file |
| `BACKUP_TIME` | `03:00` | giờ sao lưu mỗi ngày (HH:MM) |
| `BACKUP_KEEP` | `7` | số bản giữ lại |
| `BACKUP_NOW` | `0` | `1`: sao lưu ngay khi service khởi động, rồi theo lịch |

Kết quả: `deploy/backups/luna-YYYYMMDD.archive.gz`. Log: `$COMPOSE logs backup` — mỗi dòng có thời điểm, mức (`INFO`/`ERROR`) và
nội dung (bản đã tạo, kích thước, thời gian chạy, bản đã xoá, lần chạy tới, lỗi và nguyên nhân).

Sao lưu ngay (một lần, không chờ 03:00):

```bash
$COMPOSE run --rm -e BACKUP_NOW=1 -e BACKUP_ONCE=1 backup
```

(`BACKUP_ONCE=1`: sao lưu rồi thoát, không vào vòng lặp.)

## Khôi phục

```bash
$COMPOSE stop backend
$COMPOSE run --rm --entrypoint bash backup /backup/restore.sh luna-20260930.archive.gz
$COMPOSE start backend
```

| Tình huống | Kết quả | Mã thoát |
| --- | --- | --- |
| thiếu tham số | in cách dùng + danh sách bản trong `/backups` | 1 |
| file không tồn tại | `ERROR backup not found: …` + danh sách bản | 1 |
| file hỏng (`gzip -t` lỗi) | `ERROR backup is corrupt: …`, DB không bị đụng | 1 |
| thành công | `mongorestore --drop`: mọi collection trong bản sao lưu được thay; in số document đã khôi phục | 0 |
| lỗi `mongorestore` | in lỗi của mongorestore | ≠ 0 |

Dữ liệu hiện tại của các collection có trong bản sao lưu bị thay hoàn toàn (kể cả tài khoản và phiên đăng nhập).
