# Implementation Plan: Cài đặt

**Branch**: `010-user-settings` | **Date**: 2026-09-30 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/010-user-settings/spec.md` + khối 2 trong `docs/spec-inputs/f12-cai-dat.md`

**Note**: This template is filled in by the `/speckit-plan` command. See `.specify/templates/plan-template.md` for the execution workflow.

## Summary

Cài đặt theo tài khoản: chế độ giao diện, số thẻ ôn tối đa mỗi ngày, múi giờ. Backend thêm package `internal/settings` (model,
kiểm tra đầu vào, service) và lưu vào `users.settings {theme, dailyReviewLimit, timezone}`; trường `timezone` của F1 chuyển vào
`settings` bằng migration. API `GET /api/settings`, `PUT /api/settings` (cập nhật từng phần). `internal/progress` và `internal/vocab`
đọc múi giờ và giới hạn thẻ qua `settings` (adapter trong `main.go`) thay cho giá trị mặc định. L thêm quy tắc "ngày học không bao
giờ lùi" để đổi múi giờ giữa ngày không làm mất tiến độ hay sinh thêm bài. Frontend thêm trang `/settings` (Giao diện, Học tập, Múi
giờ), link "Cài đặt" ở thanh trên, và `ThemeService` đồng bộ với tài khoản: sau khi đăng nhập lấy theme từ server ghi vào
localStorage; đổi theme (ở trang Cài đặt hay nút nhanh ở thanh trên) thì lưu cả hai nơi.

## Technical Context

**Language/Version**: Go 1.25.1 (`GOTOOLCHAIN=local`); TypeScript 5.9 strict + Angular 21

**Primary Dependencies**: không thêm thư viện (Go `time.LoadLocation` với `time/tzdata` đã nhúng; trình duyệt
`Intl.supportedValuesOf('timeZone')`, `Intl.DateTimeFormat` để hiện giờ lệch)

**Storage**: MongoDB `users` thêm tài liệu con `settings {theme, dailyReviewLimit, timezone}`; migration một lần chuyển
`users.timezone` → `users.settings.timezone` (chạy trong `PrepareInBackground` như migration F14). Không collection, không index mới

**Testing**: Go `testing`: `settings` (kiểm tra theme, 5 ≤ limit ≤ 200, múi giờ hợp lệ/không hợp lệ/"Local"/rỗng, cập nhật từng
phần, giá trị mặc định cho tài khoản cũ), handler (200, 400 theo trường, 401, chỉ sửa của mình); `progress` (đổi múi giờ cùng ngày
giữ bài và bước; sang ngày sau giữ bài đang dở; lùi ngày không sinh bài mới; giới hạn thẻ đổi giữa ngày). Vitest: settings page,
`SettingsApiService`, `ThemeService` (lấy từ server sau đăng nhập, lưu hai nơi, không lưu server khi chưa đăng nhập), link header

**Target Platform / Project Type**: như F0–F6

**Performance Goals**: đổi giao diện < 0,5 s (chỉ đổi thuộc tính `data-theme`, không chờ server); `GET/PUT /api/settings` < 100 ms

**Constraints**: cài đặt theo tài khoản; trước đăng nhập dùng localStorage (script inline F0, không nhấp nháy); dữ liệu theo người
học; 360px + bàn phím + trình đọc màn hình; tiến độ không mất khi đổi múi giờ

**Scale/Scope**: 1 tài khoản, 3 cài đặt, ~420 múi giờ trong danh sách

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

Đối chiếu `.specify/memory/constitution.md` v1.0.0.

- [x] **I. docs/ là nguồn sự thật**: F12 theo `docs/phases/giai-doan-1.md` và `design-system.md` mục 2 (3 lựa chọn, `data-theme`,
  `color-scheme`); package `internal/settings` và `features/settings` đúng `docs/architecture.md`.
- [x] **II. Chi phí 0 đồng**: không dịch vụ hay thư viện mới.
- [x] **III. AI là phần bổ sung**: F12 không gọi AI.
- [x] **IV. Mobile-first**: một cột ở 360px; nhóm lựa chọn giao diện là radio có nhãn; ô số và danh sách múi giờ có nhãn, lỗi gắn
  `aria-describedby`; tương phản theo token.
- [x] **V. Dữ liệu người học**: API chỉ đọc/sửa cài đặt của người dùng trong phiên; DB qua repository; không nhận `userId` từ client.
- [x] **VI. Kiểm thử**: kiểm tra đầu vào, handler, quy tắc đổi múi giờ trong `progress`, component và service frontend.
- [x] **VII. Đơn giản trước**: lưu trong tài liệu `users` (không collection mới), đọc cài đặt khi cần (không cache), cập nhật từng
  phần bằng một `$set`.

**Kết quả**: ĐẠT, không có ngoại lệ.

## Project Structure

### Documentation (this feature)

```text
specs/010-user-settings/
├── plan.md, research.md, data-model.md, quickstart.md
├── contracts/settings-api.md
└── tasks.md                                 # /speckit-tasks
```

### Source Code (repository root)

```text
backend/
├── cmd/api/main.go                          # settings service/handler; userTimezones + ReviewLimit đọc từ settings
└── internal/
    ├── settings/                            # MỚI
    │   ├── model.go                         # Theme, Settings, Defaults, Patch, ValidationError
    │   ├── validate.go (+ _test)            # theme, limit 5–200, timezone (LoadLocation, không rỗng/"Local")
    │   ├── repository.go                    # Repository: Get, Update(patch)
    │   ├── service.go (+ _test)             # Get (mặc định cho tài khoản cũ), Update, Location, ReviewLimit
    │   ├── handler.go (+ _test)             # GET/PUT /api/settings
    │   └── fake_test.go
    ├── auth/                                # Register ghi settings mặc định + múi giờ đăng ký
    ├── progress/
    │   ├── rules.go (+ _test)               # EffectiveDayKey: ngày học không lùi
    │   ├── study.go                         # load dùng EffectiveDayKey
    │   └── study_repository.go              # DayRepository.LatestKey
    └── storage/mongo/
        ├── users.go                         # userDoc.Settings; timezone đọc từ settings (dự phòng trường cũ)
        ├── settings.go                      # MỚI: Settings repository trên users
        ├── study.go                         # StudyDays.LatestKey
        └── migrate.go                       # migration users-settings-timezone

frontend/src/
├── index.html                               # (giữ) script inline đọc luna.theme
└── app/
    ├── app.routes.ts                        # + 'settings'
    ├── core/models/settings.ts
    ├── core/services/settings-api.service.ts (+ spec)
    ├── core/services/theme.service.ts (+ spec)   # đồng bộ với tài khoản
    ├── shared/components/app-header/        # + link "Cài đặt"
    └── features/settings/ (+ spec)          # MỚI: trang /settings
```

**Structure Decision**: `settings` là domain riêng như khối 2 và `docs/architecture.md`; `auth` giữ `User.Timezone` (đọc từ
`settings.timezone`) để `/api/auth/me` và F1 không đổi. `progress`/`vocab` không import `settings`: `main.go` nối qua port
`Timezones` và hàm `ReviewLimit` đã có từ L.

## Post-Design Constitution Check

Một package mới, hai endpoint, một migration, một quy tắc thuần mới trong `progress`, một trang frontend. Không thư viện, collection,
index, job hay cache mới. **Vẫn ĐẠT.**

## Complexity Tracking

Không có vi phạm cần giải trình.
