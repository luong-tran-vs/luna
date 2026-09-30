---

description: "Task list for F13 – Sao lưu và xuất dữ liệu"
---

# Tasks: Sao lưu và xuất dữ liệu (F13)

**Input**: Design documents from `specs/011-backup-export/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/, quickstart.md

**Tests**: Tests are REQUIRED by constitution principle VI for the export (only the session user's data, no `passwordHash`/sessions,
deleted lessons, new learner, value normalisation, file name date in the learner's timezone, endpoint 200/401/500, download button).
Backup and restore scripts are verified by running them (quickstart.md bước 1–2), as recorded in plan.md Complexity Tracking.
Every acceptance criterion maps to a test task or a manual check task.

**Organization**: Tasks are grouped by user story (US1–US3 from spec.md).

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1–US3)

---

## Phase 1: Setup (Shared Infrastructure)

- [X] T001 [P] Add `deploy/backups/` to `.gitignore`; create `deploy/backup/` for the scripts
- [X] T002 [P] Add `BACKUP_TZ` (Asia/Ho_Chi_Minh), `BACKUP_TIME` (03:00), `BACKUP_KEEP` (7) with Vietnamese comments to `deploy/.env.example`

---

## Phase 2: Foundational (Blocking Prerequisites)

**⚠️ CRITICAL**: US1 and US2 share the backup service; US3 is independent of this phase

- [X] T003 Add service `backup` to `deploy/docker-compose.yml`: `image: mongo:8`, `restart: unless-stopped`, `entrypoint: ["bash", "/backup/backup.sh"]`, volumes `./backup:/backup:ro` and `./backups:/backups`, environment `MONGO_URI` (`${MONGO_URI:-mongodb://mongo:27017}`), `MONGO_DATABASE` (`${MONGO_DATABASE:-luna}`), `TZ: ${BACKUP_TZ:-Asia/Ho_Chi_Minh}`, `BACKUP_TIME: ${BACKUP_TIME:-03:00}`, `BACKUP_KEEP: ${BACKUP_KEEP:-7}`, `BACKUP_NOW: ${BACKUP_NOW:-0}`, `depends_on: [mongo]`, with a comment pointing to README (research R1)

**Checkpoint**: `$COMPOSE config` is valid

---

## Phase 3: User Story 1 - Tự sao lưu mỗi ngày (Priority: P1) 🎯 MVP

**Goal**: Sao lưu 03:00 mỗi ngày, giữ 7 bản, lỗi chỉ ghi log

**Independent Test**: quickstart.md bước 1

### Implementation for User Story 1

- [X] T004 [US1] Write `deploy/backup/backup.sh` (bash, `set -euo pipefail` around each run but the loop never exits on a failed backup): `log LEVEL msg` with ISO time; `run_backup` → name `luna-$(date +%Y%m%d).archive.gz`, `mongodump --uri "$MONGO_URI" --db "$MONGO_DATABASE" --archive="$tmp" --gzip` into `$name.tmp`, check non-empty, `mv` to the real name, log size and duration, then rotate (only files matching `luna-[0-9]{8}.archive.gz`, sorted by name, delete all but the newest `BACKUP_KEEP`, log each deletion); on failure remove `.tmp`, log `ERROR backup failed: <reason>` and skip rotation; startup: remove stale `*.tmp`, validate `BACKUP_TIME`/`BACKUP_KEEP`, run now when `BACKUP_NOW=1` or when today's time has passed and today's file is missing; `BACKUP_ONCE=1` exits after that run (exit code = backup result); loop: compute the next `BACKUP_TIME` with `date -d "today $BACKUP_TIME"`/`"tomorrow $BACKUP_TIME"`, log "next backup at …", `sleep`, run; handle SIGTERM for a quick stop (research R2–R4)
- [X] T005 [US1] Manual check quickstart.md bước 1: `BACKUP_NOW=1 BACKUP_ONCE=1` creates today's file; service log shows the next 03:00; rotation with 7 older fake files keeps exactly 7 and leaves `ghi-chu.txt`; with `mongo` stopped the run logs `ERROR`, creates nothing, deletes nothing, leaves no `.tmp`, and the app keeps serving

**Checkpoint**: MVP — có bản sao lưu mỗi ngày, xoay vòng 7 bản

---

## Phase 4: User Story 2 - Khôi phục bằng một lệnh (Priority: P1)

**Goal**: Một lệnh khôi phục có kiểm tra file, README hướng dẫn

**Independent Test**: quickstart.md bước 2

### Implementation for User Story 2

- [X] T006 [US2] Write `deploy/backup/restore.sh` per contracts/backup-cli.md: usage + list of `/backups/luna-*.archive.gz` when the argument is missing or the file does not exist (a bare name resolves under `/backups`), exit 1; `gzip -t` before touching the database (corrupt → `ERROR backup is corrupt`, exit 1); `mongorestore --uri "$MONGO_URI" --archive="$file" --gzip --drop --nsInclude "$MONGO_DATABASE.*"`; print per-collection document counts after restoring (`mongosh --quiet --eval` over `db.getCollectionNames()`); exit with mongorestore's code on failure (research R5)
- [X] T007 [US2] Add README section "Sao lưu và khôi phục" in `README.md`: where backups live, schedule and variables, `docker compose logs backup`, back up now (`run --rm -e BACKUP_NOW=1 -e BACKUP_ONCE=1 backup`), list backups, restore (stop backend → restore.sh → start backend), overwrite warning (accounts and sessions return to the backup time), check after restoring, audio and dictionary not included (how to regenerate), copy `deploy/backups/` elsewhere for safety
- [X] T008 [US2] Manual check quickstart.md bước 2: count documents, drop the database, restore with the README command, counts match, `hoc@example.com` sees the same notebook and progress; missing file and corrupt file both exit 1 without touching data

**Checkpoint**: sao lưu và khôi phục dùng được

---

## Phase 5: User Story 3 - Xuất dữ liệu của mình (Priority: P2)

**Goal**: Nút Xuất dữ liệu tải file JSON chỉ có dữ liệu của người học

**Independent Test**: quickstart.md bước 3

### Tests for User Story 3 (REQUIRED - constitution VI) ⚠️

- [X] T009 [P] [US3] Create `backend/internal/export/model.go` (`Account`, `Doc = map[string]any`, `Export` struct with JSON tags `version`, `exportedAt`, `account`, `settings`, `cards`, `reviewLogs`, `goals`, `lessonProgress`, `studyDays`, `dictationResults`, `lessons`; `ErrNotFound`; allowed collection names) and `backend/internal/export/repository.go` (`Repository`: `Account`, `UserDocs(collection, userID)`, `Lessons(ids)`; port `Settings`: `Get` returning a `map[string]any` or small struct, `Location`) per data-model.md
- [X] T010 [US3] Write `backend/internal/export/fake_test.go` (in-memory docs for two users incl. a user doc with `passwordHash` and a `sessions` collection that must never be read; fake settings) and `backend/internal/export/service_test.go`: export of u1 has only u1's docs in every section, no `userId` field, `account` has exactly id/email/role/createdAt, the marshalled JSON contains no `passwordHash`, `tokenHash`, u2's email or u2's ids; `lessons` = distinct lesson ids of `lessonProgress`, deleted ones as `{id, deleted: true}`; new learner → every array `[]` (not null) and `settings` with defaults; `UserDocs` is never called with a collection outside the allowed list; file name `luna-export-20261001.json` when it is already 2026-10-01 in the learner's timezone; unknown user → `ErrNotFound`
- [X] T011 [US3] Implement `backend/internal/export/service.go` (`NewService(repo, settings, now)`, `Build(ctx, userID) (Export, filename string, err)`) so T010 passes (research R7, R8)
- [X] T012 [US3] Write `backend/internal/export/handler_test.go`: `GET /api/export` 200 with `Content-Type: application/json; charset=utf-8`, `Content-Disposition: attachment; filename="luna-export-YYYYMMDD.json"`, `Cache-Control: no-store`, body parses and has every key; 401 without session; 401 when the account is gone; 500 on repository error
- [X] T013 [US3] Implement `backend/internal/export/handler.go` (`NewHandler(svc, log)`, `Register(mux, requireAuth)`, indented JSON) so T012 passes (research R9)
- [X] T014 [US3] Write a pure test for BSON normalisation in `backend/internal/storage/mongo/export_test.go`: `ObjectID` → hex, `bson.DateTime`/`time.Time` → RFC 3339 string, nested documents and arrays converted, `userId` removed at the top level, `bson.D`/`bson.M`/`bson.A` handled
- [X] T015 [US3] Implement `backend/internal/storage/mongo/export.go`: `Export` repository with `Account` (projection `email`, `role`, `createdAt` only; never `passwordHash`), `UserDocs` (rejects names outside the allowed list; filter `{userId: oid}`; sort `_id`), `Lessons` (`_id $in`), and the `normalize` function so T014 passes; wire `export.NewService(mongo.NewExport(database), exportSettings{settingsSvc}, time.Now)` and `export.NewHandler(...).Register` in `backend/cmd/api/main.go` with a small adapter to the settings service
- [X] T016 [P] [US3] Create `frontend/src/app/core/services/export-api.service.ts` (`ng g s core/services/export-api --type=service`) with `download(): Observable<{blob, filename}>` (`responseType: 'blob'`, `observe: 'response'`, filename from `Content-Disposition`, fallback `luna-export.json`) and `export-api.service.spec.ts` (filename parsed, fallback when the header is missing)
- [X] T017 [US3] Extend `frontend/src/app/features/settings/settings.spec.ts`: group "Dữ liệu" with a short description and button **Xuất dữ liệu**; clicking calls `download()`, shows "Đang xuất…" with the button disabled and `aria-busy`, then triggers a download through a temporary `<a download="luna-export-….json">` with an object URL (stub `URL.createObjectURL`/`revokeObjectURL`, spy on `HTMLAnchorElement.prototype.click`) and revokes the URL; an error shows `role="alert"` "Không xuất được dữ liệu, vui lòng thử lại." and the button works again
- [X] T018 [US3] Implement the "Dữ liệu" group in `frontend/src/app/features/settings/settings.html` / `settings.ts` / `settings.css` so T017 passes (research R10)

**Checkpoint**: tất cả user story xong

---

## Phase 6: Polish & Cross-Cutting Concerns

- [X] T019 Update `README.md`: section "Xuất dữ liệu" (nội dung file, chỉ dữ liệu của mình, không mật khẩu) next to "Cài đặt"
- [X] T020 Run lint (frontend `npm run lint` + backend golangci-lint via Docker) with zero errors; `bash -n deploy/backup/*.sh` (and `shellcheck` via Docker `koalaman/shellcheck` if available) clean
- [X] T021 Run all tests (`go test -race ./...`, `npx ng test --watch=false`) with zero failures; `npx ng build` without warnings
- [X] T022 Rebuild and smoke test in Docker (quickstart.md bước 3): export for `hoc@example.com` has the right counts and no `passwordHash`/`tokenHash`; export for `admin@example.com` has none of `hoc`'s data; headers; 401
- [ ] T023 Manual check of the export button at 360px, keyboard only, and offline error (quickstart.md bước 3.5)
- [X] T024 Tick the 3 F13 acceptance criteria in `docs/phases/giai-doan-1.md` once T005, T008, T022 pass

---

## Dependencies & Execution Order

- **Setup** → **Foundational** (compose service) → **US1** (backup.sh) → **US2** (restore.sh, README) → **Polish**
- **US3** (export) depends only on Setup and can run in parallel with US1/US2 (different files: Go, Angular vs shell, compose)
- Within US3: model/repository → service tests → service → handler tests → handler → Mongo repository + wiring; frontend service ∥ backend

### Parallel Opportunities

- T001 ∥ T002; T004–T008 (shell, README) ∥ T009–T018 (Go, Angular)
- T016 (frontend service) ∥ T010–T015 (backend)

---

## Parallel Example: User Story 3

```bash
Task: "export service tests in backend/internal/export/service_test.go"
Task: "export-api service + spec in frontend/src/app/core/services/"
Task: "BSON normalisation test in backend/internal/storage/mongo/export_test.go"
```

---

## Implementation Strategy

1. Setup + Foundational → US1 (sao lưu mỗi ngày) → **STOP and VALIDATE** với quickstart bước 1
2. US2 → khôi phục + README → quickstart bước 2
3. US3 → xuất dữ liệu (có thể làm song song với 1–2)
4. Polish → README, lint, test, Docker, kiểm tra thủ công, tick tiêu chí nghiệm thu

---

## Notes

- Script chạy trong image `mongo:8` (Ubuntu 24.04: bash, GNU date, tzdata, mongodump/mongorestore 100.18); dùng LF, không CRLF
- `export` không import `settings`; `main.go` nối qua adapter nhỏ
- Không bao giờ đọc `sessions` hay `passwordHash` khi xuất
- Chỉ commit khi người dùng yêu cầu

---

## Ghi chú triển khai

- `backup.sh`: `find -regex` mặc định (emacs) không hiểu `{8}` nên lần thử đầu không xoá bản nào; đã dùng
  `-regextype posix-extended`. Bỏ `--quiet` của `mongodump` vì nó ẩn cả thông báo lỗi (log lỗi giờ có nguyên nhân, ví dụ
  `Failed: can't create session … server selection error`).
- `restore.sh`: mã thoát của `mongorestore` lấy ngay sau lệnh (không trong `if !`, vì khi đó `$?` luôn 0).
- `.gitattributes`: `*.sh text eol=lf` (repo có `core.autocrlf=true`; script chạy trong container Linux).
- Git Bash trên Windows đổi `/backup/restore.sh` thành đường dẫn Windows: README ghi thêm `MSYS_NO_PATHCONV=1`.
- `docker compose run backup` khởi động cả `mongo` (depends_on); khi thử lỗi với mongo tắt dùng `--no-deps`.
- Kiểm tra thủ công đã chạy (T005, T008): sao lưu ngay tạo `luna-20260930.archive.gz`; 7 bản giả 20260901–07 + bản hôm nay → xoá
  20260901, giữ `ghi-chu.txt`, xoá `.tmp` sót; mongo tắt → `ERROR backup failed: …` sau 30 s, không file mới/không xoá, app vẫn trả
  trang và `/api/health` báo DB down; service nền log "next backup at 2026-10-01T03:00:00+07:00", xoá bản hôm nay rồi khởi động lại →
  sao lưu bù ngay, SIGTERM dừng ngay. Khôi phục: thiếu tham số / file không có / file hỏng đều thoát 1 không đụng DB; xoá database
  rồi khôi phục → 12 collection khớp số document (64 document), đăng nhập `hoc@example.com` thấy lại mục tiêu và bài hôm nay. Các
  file sao lưu giả đã xoá, còn lại bản thật `luna-20260930.archive.gz`.
- Xuất dữ liệu: repository `storage/mongo/export.go` chỉ đọc 6 collection trong danh sách cho phép (từ chối tên khác bằng
  `ErrCollection`), tài khoản chỉ lấy `email`, `role`, `createdAt`; service bỏ `userId` khỏi mọi tài liệu. Ngày dạng RFC 3339
  (UTC), `exportedAt` theo múi giờ người học.
- Smoke test Docker (T022): `hoc` → header đúng, 3 thẻ, 1 log, 2 mục tiêu, 1 tiến độ, 1 ngày học, 4 kết quả chép, 1 bài — khớp đếm
  trong MongoDB; `admin` → không có email/thẻ của `hoc`; cả hai file 0 `passwordHash`, 0 `tokenHash`, 0 `userId`; 401 khi thiếu
  phiên.
- T024: đã tick 3 tiêu chí F13 (dựa trên T005, T008, T022). Còn lại cho người dùng: T023 (nút Xuất dữ liệu ở 360px, chỉ bàn phím,
  lỗi khi offline).
