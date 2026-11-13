# Luna — hướng dẫn cho agent (Codex)

Luna là web app học tiếng Anh: backend Go (`backend/`), frontend Angular 21 (`frontend/`), MongoDB hoặc MySQL.

## Việc đang làm

- Tính năng hiện tại: **F26 Trò chơi** — `specs/021-learning-games/`.
  - Đọc theo thứ tự: `spec.md` (làm gì) → `plan.md` (cấu trúc) → `research.md` (quyết định R1–R8) → `data-model.md`
    → `contracts/api.md` → `tasks.md` (danh sách việc).
  - Làm task theo thứ tự trong `tasks.md`, mỗi lần một phase (bắt đầu từ Phase 1–3 = MVP). Xong task nào thì đổi
    `- [ ]` thành `- [x]` trong `tasks.md`.
  - Gặp chỗ spec/plan không rõ hoặc mâu thuẫn với code: dừng lại và hỏi, không tự đoán.
- Nguồn sự thật của sản phẩm: `docs/` (`docs/phases/*.md`, `docs/architecture.md`, `docs/design-system.md`) và
  `.specify/memory/constitution.md`.

## Quy ước bắt buộc

- **Trả lời người dùng, viết tài liệu, chữ trên giao diện: tiếng Việt.** Tên trong code, API, commit message: tiếng Anh.
- **Câu đưa vào prompt gửi AI phải viết bằng tiếng Anh.**
- **Không tự commit hay push.** Chỉ commit khi người dùng yêu cầu rõ.
- Đổi biến môi trường thì sửa **cả** `backend/.env` và `backend/.env.example`.
- **Dev và production dùng chung một DB** (cả hàng đợi job): thay đổi schema chỉ được thêm (collection, bảng, cột), không
  đổi tên hay xoá. MySQL thêm migration mới trong `backend/internal/storage/mysql/migrate.go` (số tiếp theo).
- Mọi repository làm cho **cả** Mongo (`backend/internal/storage/mongo`) và MySQL (`backend/internal/storage/mysql`),
  khai báo trong `backend/internal/storage/storage.go`.
- Dữ liệu mới của người học phải có trong file xuất (`backend/internal/export`) và bị xoá khi xoá tài khoản
  (`storage/{mongo,mysql}/users.go`).
- Thêm chức năng gọi AI: thêm op vào `frontend/src/app/features/admin/ai-usage/ai-usage-catalog.ts`.

## Backend (Go 1.25)

- Chia theo domain trong `backend/internal/<domain>`; interface chỉ cho DB và AI (`ai.Provider`); DI bằng constructor
  viết tay, nối trong `backend/internal/service/` (`init<Domain>()` + `routes.go`).
- Quyền kiểm tra ở API: `requireAuth`, `httpx.RequireMember` (khách bị 403 `members_only`), `httpx.RequireAdmin`.
- Lỗi trả bằng `httpx.WriteError` / `httpx.WriteFieldErrors` (thông báo tiếng Việt).
- Skill Go trong `.agents/skills/golang-*` (samber/cc-skills-golang): theo đó khi viết code Go.
- Test: `cd backend; go test ./...` và `go vet ./...`. Test không gọi mạng (AI dùng fake). Endpoint mới cần test thành
  công, đầu vào sai, sai quyền.

## Frontend (Angular 21)

- Đọc `frontend/.claude/CLAUDE.md` (quy ước Angular: standalone, signals, `input()`/`output()`, OnPush, `@if/@for`, không
  `ngClass`/`ngStyle`, WCAG AA).
- Màu, font, cỡ chữ chỉ dùng token (`frontend/src/styles/tokens.css`, `docs/design-system.md`); có sáng và tối; dùng tốt
  từ 360 px, không cuộn ngang.
- `localStorage` luôn bọc try/catch.
- Test: `cd frontend; npm test` và `npm run lint`. Chạy thử: `npm run start:dev`.

## Kiểm tra trước khi báo xong

1. Test và lint backend + frontend đều qua.
2. Làm các bước kiểm tra thủ công liên quan trong `specs/021-learning-games/quickstart.md` (360 px, sáng và tối).
3. Báo lại cho người dùng (tiếng Việt): đã làm task nào, file nào đổi, test nào chạy, còn gì chưa làm.
