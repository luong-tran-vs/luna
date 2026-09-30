# Implementation Plan: Sao lưu và xuất dữ liệu

**Branch**: `011-backup-export` | **Date**: 2026-09-30 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/011-backup-export/spec.md` + khối 2 trong `docs/spec-inputs/f13-sao-luu.md`

**Note**: This template is filled in by the `/speckit-plan` command. See `.specify/templates/plan-template.md` for the execution workflow.

## Summary

Hai phần độc lập. **Sao lưu**: service `backup` trong `deploy/docker-compose.yml` dùng image `mongo:8` (có `mongodump`,
`mongorestore`, tzdata, GNU `date`), chạy `deploy/backup/backup.sh`: vòng lặp chờ tới 03:00 theo `TZ`, `mongodump --archive
--gzip` vào `deploy/backups/luna-YYYYMMDD.archive.gz` (ghi file tạm rồi đổi tên), thành công mới giữ lại 7 bản mới nhất; `BACKUP_NOW=1`
sao lưu ngay. `deploy/backup/restore.sh <file>` kiểm tra file rồi `mongorestore --archive --gzip --drop`; README có hướng dẫn. Audio
và từ điển không sao lưu. **Xuất dữ liệu**: package `internal/export` với `GET /api/export` gom `users` (không `passwordHash`),
`settings`, `cards`, `review_logs`, `goals`, `lesson_progress`, `study_days`, `dictation_results` và các `lessons` người học đã bắt
đầu, trả JSON `Content-Disposition: attachment; filename="luna-export-YYYYMMDD.json"`. Frontend thêm nút **Xuất dữ liệu** vào trang
Cài đặt.

## Technical Context

**Language/Version**: Go 1.25.1 (`GOTOOLCHAIN=local`); TypeScript 5.9 strict + Angular 21; Bash (Ubuntu 24.04 trong `mongo:8`)

**Primary Dependencies**: không thêm thư viện; `mongodump`/`mongorestore` 100.18 có sẵn trong image `mongo:8` đang dùng

**Storage**: đọc mọi collection của người học (không ghi); bản sao lưu là file trên máy chủ (`deploy/backups/`, bind mount). Không
collection hay index mới (mọi truy vấn xuất theo `userId` đã có index tiền tố `userId`)

**Testing**: Go `testing`: `export` (chỉ dữ liệu của người yêu cầu, không `passwordHash`/phiên, bài đã xoá ghi `deleted`, người học mới →
danh sách rỗng, chuẩn hoá ObjectID/ngày sang chuỗi, tên file theo ngày ở múi giờ người học), handler (200 + header, 401, 500). Vitest:
nút Xuất dữ liệu (tải blob, tên file từ header, đang xuất, lỗi). Sao lưu: kiểm tra thủ công theo quickstart (BACKUP_NOW, xoay vòng 7
bản, khôi phục vào DB trống, so số document) — script shell không có bộ test tự động

**Target Platform / Project Type**: như F0–F12 (Docker compose; máy chủ Linux hoặc Docker Desktop)

**Performance Goals**: xuất tài khoản vài nghìn thẻ, vài chục nghìn lượt ôn < 10 s (SC-006): 8 truy vấn theo `userId` + 1 truy vấn bài học
`$in`; sao lưu DB vài chục MB trong vài giây, không khoá app (mongodump chỉ đọc)

**Constraints**: file xuất chỉ dữ liệu của phiên, không mật khẩu/phiên (nguyên tắc V); sao lưu lỗi chỉ ghi log, không xoá bản cũ,
không ảnh hưởng app (container riêng); khôi phục lỗi không xoá dữ liệu hiện tại

**Scale/Scope**: 1 máy chủ, DB vài chục MB, 7 bản sao lưu

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

Đối chiếu `.specify/memory/constitution.md` v1.0.0.

- [x] **I. docs/ là nguồn sự thật**: F13 theo `docs/phases/giai-doan-1.md` (3 tiêu chí nghiệm thu); `internal/export`,
  `features/settings`, `deploy/` đúng `docs/architecture.md`.
- [x] **II. Chi phí 0 đồng**: dùng image `mongo:8` sẵn có, lưu trên máy chủ; không cloud.
- [x] **III. AI là phần bổ sung**: không gọi AI.
- [x] **IV. Mobile-first**: nút xuất trong trang Cài đặt một cột, ≥ 44px, có trạng thái đang xuất và lỗi đọc được.
- [x] **V. Dữ liệu người học**: xuất lọc theo `userId` của phiên qua repository; tài khoản chỉ lấy trường cho phép (email, vai trò,
  ngày tạo); không bao giờ đọc `sessions` hay `passwordHash`. Bản sao lưu chỉ người vận hành truy cập (thư mục trên máy chủ, không
  phục vụ qua web).
- [x] **VI. Kiểm thử**: service + handler xuất có test; sao lưu/khôi phục có kịch bản kiểm tra thủ công trong quickstart (ngoại lệ
  hợp lý: script vận hành, kiểm bằng chạy thật).
- [x] **VII. Đơn giản trước**: một script shell + một service compose; xuất đồng bộ trong một request, không job nền, không lưu file.

**Kết quả**: ĐẠT, không có ngoại lệ cần giải trình (kiểm thử script bằng tay ghi ở Complexity Tracking).

## Project Structure

### Documentation (this feature)

```text
specs/011-backup-export/
├── plan.md, research.md, data-model.md, quickstart.md
├── contracts/export-api.md, contracts/backup-cli.md
└── tasks.md                                 # /speckit-tasks
```

### Source Code (repository root)

```text
deploy/
├── docker-compose.yml                       # + service backup (mongo:8, TZ, BACKUP_NOW, BACKUP_KEEP)
├── .env.example                             # + BACKUP_TZ, BACKUP_TIME, BACKUP_KEEP (nếu file có)
├── backup/                                  # MỚI
│   ├── backup.sh                            # lịch 03:00, mongodump, xoay vòng 7 bản, log
│   └── restore.sh                           # kiểm tra file, mongorestore --drop
└── backups/                                 # MỚI (bind mount, .gitignore): luna-YYYYMMDD.archive.gz

backend/
├── cmd/api/main.go                          # export service/handler
└── internal/
    ├── export/                              # MỚI
    │   ├── model.go                         # Export, Account, Lesson (deleted)
    │   ├── repository.go                    # Repository: Account, Collection(userID, name), Lessons(ids)
    │   ├── service.go (+ _test)             # Build(userID): gom, bài đã học, tên file theo múi giờ
    │   ├── handler.go (+ _test)             # GET /api/export
    │   └── fake_test.go
    └── storage/mongo/
        └── export.go                        # MỚI: đọc theo userId, chuẩn hoá BSON → JSON

frontend/src/app/
├── core/services/export-api.service.ts (+ spec)   # MỚI: tải blob + tên file
└── features/settings/                       # + nhóm "Dữ liệu" với nút Xuất dữ liệu

README.md                                    # + mục Sao lưu và khôi phục, Xuất dữ liệu
.gitignore                                   # + deploy/backups/
```

**Structure Decision**: Sao lưu là việc vận hành nên nằm trong `deploy/` như khối 2, chạy trong container riêng để lỗi không chạm
backend. Xuất dữ liệu là domain `export` riêng; đọc DB qua một repository chỉ đọc (`storage/mongo/export.go`) thay vì gọi service của
từng domain, vì cần bản ghi đầy đủ (kể cả lịch ôn, log) theo đúng dạng lưu trữ và không có domain nào đang lộ ra các trường đó.

## Post-Design Constitution Check

Một service compose, hai script, một package Go chỉ đọc, một endpoint, một nút. Không collection, index, thư viện, job hay cache mới.
**Vẫn ĐẠT.**

## Complexity Tracking

| Điểm | Vì sao cần | Phương án đơn giản hơn bị loại vì |
| --- | --- | --- |
| Script sao lưu/khôi phục kiểm thử bằng tay (quickstart), không có test tự động | Hành vi phụ thuộc Docker, MongoDB thật và đồng hồ | Viết khung test shell/bats cho 2 script nhỏ tốn hơn giá trị; kịch bản quickstart lặp lại được |
