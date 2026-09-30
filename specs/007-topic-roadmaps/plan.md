# Implementation Plan: Chủ đề và lộ trình theo trình độ

**Branch**: `007-topic-roadmaps` | **Date**: 2026-09-30 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/007-topic-roadmaps/spec.md` + khối 2 trong `docs/spec-inputs/f14-chu-de.md`

**Note**: This template is filled in by the `/speckit-plan` command. See `.specify/templates/plan-template.md` for the execution workflow.

## Summary

Thay "chủ đề gõ tự do + một lộ trình chung" của F2 bằng danh mục chủ đề theo trình độ, mỗi chủ đề một lộ trình. Backend thêm
gói `internal/topic` (chủ đề + lộ trình lưu trong document chủ đề, collection `topics` với unique `(level, nameKey)`), sửa
`internal/lesson` (bài có `topicId`, trình độ lấy theo chủ đề, đổi chủ đề thì chuyển lộ trình, bỏ lộ trình chung), và chuyển
dữ liệu F2 một lần khi khởi động (kế hoạch thuần trong `topic.PlanMigration`, áp dụng idempotent trong
`storage/mongo/migrate.go`). Frontend thêm trang `/admin/topics`, sửa form bài (một ô chọn chủ đề), bộ lọc (trình độ + chủ đề),
trang lộ trình (chọn chủ đề, cảnh báo theo chủ đề).

## Technical Context

**Language/Version**: Go 1.25.1 (`GOTOOLCHAIN=local`); TypeScript 5.9 strict + Angular 21

**Primary Dependencies**: không thêm thư viện (Go chuẩn + mongo-driver v2; Angular + `@angular/cdk/drag-drop` đã có)

**Storage**: MongoDB **mới** `topics` (unique `(level, nameKey)`), `migrations`; `lessons` bỏ `topic`, thêm `topicId` (index);
**bỏ** `roadmap`

**Testing**: Go `testing` + `httptest` với fake repository/port; `PlanMigration` test thuần (không DB); migration Mongo kiểm tra
bằng smoke test Docker chạy 2 lần; Vitest cho topics, lesson-form, lesson-list, roadmap

**Target Platform / Project Type**: như F0–F5 (web app, Docker Compose)

**Performance Goals**: danh sách chủ đề và lộ trình < 1 giây với vài trăm bài; migration chạy một lần, vài trăm bài < 5 giây

**Constraints**: không mất bài, giữ thứ tự tương đối khi chuyển dữ liệu; chạy lại không trùng; chỉ quản trị viên; 360px

**Scale/Scope**: vài chục chủ đề, vài trăm bài

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

Đối chiếu `.specify/memory/constitution.md` v1.0.0.

- [x] **I. docs/ là nguồn sự thật**: F14 theo `docs/phases/giai-doan-1.md` và `mvp-features.md` v8 (R1, F14); spec bao phủ 4
  tiêu chí nghiệm thu; spec F2 (FR-024, ô chủ đề) được thay có ghi chú.
- [x] **II. Chi phí 0 đồng**: không dịch vụ hay thư viện mới.
- [x] **III. AI là phần bổ sung**: F14 không dùng AI; chú thích AI vẫn nhận trình độ từ bài (lấy theo chủ đề).
- [x] **IV. Mobile-first**: trang Chủ đề, form, bộ lọc, lộ trình một cột ở 360px; kéo thả có nút Lên/Xuống thay thế (F2); cảnh
  báo bằng chữ và biểu tượng.
- [x] **V. Dữ liệu**: DB qua `topic.Repository`, `lesson.Repository`; hai gói nối qua port; chỉ quản trị viên ghi; người học chỉ
  đọc danh sách chủ đề công khai.
- [x] **VI. Kiểm thử**: service topic (trùng tên, xoá khi còn bài, lộ trình chỉ bài của chủ đề, cảnh báo 0–3, đổi trình độ kéo
  bài theo), service lesson (bắt buộc chủ đề, đổi chủ đề chuyển lộ trình), `PlanMigration` (bài không chủ đề, gộp tên, giữ thứ
  tự, id đã xoá, chạy lại với chủ đề đã có), handler (thành công, lỗi, 401/403); frontend topics, lesson-form, lesson-list,
  roadmap.
- [x] **VII. Đơn giản trước**: lộ trình nằm trong document chủ đề (không collection riêng); migration tự viết một hàm, marker
  một document; không transaction (thứ tự ghi an toàn, research R2, R4).

**Kết quả**: ĐẠT, không có ngoại lệ.

## Project Structure

### Documentation (this feature)

```text
specs/007-topic-roadmaps/
├── plan.md, research.md, data-model.md, quickstart.md
├── contracts/topics-api.md
└── tasks.md
```

### Source Code (repository root)

```text
backend/
├── cmd/api/main.go                     # + topic service/handler, ports lesson.Topics ⇄ topic.Lessons, MigrateTopics trước khi serve
└── internal/
    ├── topic/                          # MỚI
    │   ├── model.go, repository.go     # Topic, Input, Summary, Roadmap, errors, Repository, port Lessons
    │   ├── service.go (+ _test)        # List, Create, Update, Delete, Roadmap, SetRoadmap, Public; port lesson.Topics
    │   ├── handler.go (+ _test)        # /api/admin/topics…, /api/topics
    │   ├── migrate.go (+ _test)        # PlanMigration (thuần)
    │   └── fake_test.go
    ├── lesson/                         # SỬA
    │   ├── model.go, validate.go       # TopicID thay Topic; Input bỏ Level/Topic; Filter.TopicID; port Topics; bỏ Roadmap
    │   ├── service.go (+ _test)        # Create/Update theo chủ đề, MoveLesson, Delete kiểm tra lộ trình qua port; bỏ Get/SetRoadmap
    │   ├── handler.go (+ _test)        # topicId/topicName, lọc topicId; bỏ /api/admin/roadmap
    │   ├── reading.go                  # tên chủ đề qua port
    │   └── repository.go, fake_test.go # bỏ RoadmapRepository; + CountByTopic, TopicOf, SetLevelByTopic
    └── storage/mongo/
        ├── topics.go                   # topic.Repository
        ├── lessons.go                  # topicId, lọc, CountByTopic, TopicOf, SetLevelByTopic
        ├── migrate.go                  # MigrateTopics (marker, upsert, unset topic, drop roadmap)
        ├── roadmap.go                  # XOÁ
        └── indexes.go                  # + topics unique, lessons.topicId; bỏ lessons.topic

frontend/src/app/
├── core/models/{topic.ts (mới), lesson.ts (sửa)}
└── features/admin/
    ├── admin-api.service.ts (+ spec)   # topics CRUD, topic roadmap; bỏ roadmap chung
    ├── admin.routes.ts                 # + 'topics'
    ├── topics/ (+ spec)                # MỚI: danh sách nhóm theo trình độ, form, xoá
    ├── lesson-form/ (+ spec)           # ô chọn chủ đề (optgroup)
    ├── lesson-list/ (+ spec)           # lọc trình độ + chủ đề
    ├── lesson-detail/ (+ spec)         # hiện tên chủ đề
    └── roadmap/ (+ spec)               # chọn chủ đề (?topicId=), cảnh báo theo chủ đề
```

**Structure Decision**: `topic` và `lesson` là hai domain không import nhau (research R1); `main.go` là nơi duy nhất biết cả
hai. Migration: logic ở `topic` (test được), ghi ở `storage/mongo`.

## Post-Design Constitution Check

Thêm 1 gói, 2 collection nhỏ, 2 port; bỏ 1 collection và 1 repository. Không thư viện mới, không transaction. **Vẫn ĐẠT.**

## Complexity Tracking

Không có vi phạm cần giải trình.
