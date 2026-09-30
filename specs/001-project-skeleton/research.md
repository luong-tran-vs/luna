# Research: Khung dự án Luna (F0)

Các quyết định kỹ thuật chính đã có trong khối 2 của `docs/spec-inputs/f0-khung-du-an.md`. File này
ghi lại những điểm khối 2 chưa nói rõ, hoặc cần chốt cách làm để đạt tiêu chí nghiệm thu. Không còn
mục NEEDS CLARIFICATION.

## R1. Máy phát triển chưa có `make` và `golangci-lint`

- **Decision**: Vẫn viết `backend/Makefile` (run, test, lint, build) như khối 2 yêu cầu. Mỗi đích là
  một lệnh đơn, và README ghi thêm lệnh tương đương để chạy trực tiếp trên PowerShell. `lint` chạy
  golangci-lint qua image `golangci/golangci-lint:v2.x` (ghim phiên bản cụ thể khi implement), mount
  thư mục backend và cache module Go.
- **Rationale**: Máy hiện tại là Windows, không có `make`. Makefile vẫn đúng cho Linux/WSL/CI; README
  đảm bảo người dùng Windows chạy được mà không cài thêm gì ngoài Go và Docker.
- **Lint không cần mạng**: sau lần `docker pull` đầu tiên và `go mod download`, image và module cache
  đã có trên máy nên chạy offline được (đúng giả định "không cần mạng" trong spec).
- **Alternatives considered**: Cài `make` qua winget (thêm bước cài đặt, trái SC-001 với máy mới);
  script `.ps1` riêng (hai bộ lệnh phải giữ đồng bộ).

## R2. Cấu hình golangci-lint

- **Decision**: `.golangci.yml` định dạng v2, chọn bộ linter từ `references/linter-reference.md` của
  skill golang-lint (file mẫu `assets/.golangci.yml` của skill không có trong repo). Bật tối thiểu:
  `govet`, `staticcheck`, `errcheck`, `ineffassign`, `unused`, `gosec`, `revive`, `errorlint`,
  `bodyclose`, `noctx`, `contextcheck`, `misspell`, `gocritic`; formatter `gofumpt`, `goimports`.
- **Rationale**: Đủ bắt lỗi thật (error chưa xử lý, request thiếu context, header timeout của
  `http.Server`) mà không quá ồn cho một codebase nhỏ.
- **Alternatives considered**: Bật `default: all` (quá nhiều cảnh báo phong cách, trái nguyên tắc VII).

## R3. Backend khởi động khi MongoDB chưa sẵn sàng

- **Decision**: `mongo.Connect` (driver v2 không mở kết nối ngay) chạy lúc khởi động; **không** ping
  bắt buộc ở startup. Server vẫn lắng nghe khi Mongo tắt; `/api/health` trả 503 `database: down`.
  Đặt `ServerSelectionTimeout` 2 giây để ping không treo lâu hơn timeout của health.
- **Rationale**: Tiêu chí nghiệm thu yêu cầu tắt cơ sở dữ liệu thì trang hiện "Mất kết nối cơ sở dữ
  liệu"; nếu backend chết theo Mongo thì trang chỉ hiện "Không kết nối được máy chủ". Mongo có lại
  thì driver tự kết nối lại, không cần khởi động lại backend (edge case trong spec).
- **Chỉ lỗi cấu hình mới làm backend dừng**: `MONGO_URI` thiếu hoặc sai cú pháp (Connect trả lỗi
  parse) → log lỗi và thoát mã 1.
- **Alternatives considered**: Ping lúc khởi động và retry (phức tạp, không cần cho F0).

## R4. Cấu hình từ biến môi trường

- **Decision**: `config.Load(getenv func(string) string) (Config, error)` nhận hàm đọc biến để test
  không phải sửa môi trường tiến trình. Gom **mọi** lỗi (thiếu `MONGO_URI`, `LOG_LEVEL` không hợp lệ)
  bằng `errors.Join` rồi trả một lần. Thông báo nêu đúng tên biến, không in giá trị.
- **Biến**: `MONGO_URI` (bắt buộc), `HTTP_ADDR` (mặc định `:8080`), `LOG_LEVEL` (`debug|info|warn|error`,
  mặc định `info`). Chi tiết ở `contracts/config.md`.
- **Chưa thêm** `MONGO_DATABASE`: F0 không dùng database nào; thêm ở F1 (nguyên tắc VII).
- **Alternatives considered**: viper (khối 2 đã loại); `os.LookupEnv` trực tiếp (khó test song song).

## R5. HTTP server, middleware và graceful shutdown

- **Decision**:
  - `http.ServeMux` với pattern `GET /api/health`. Middleware bọc theo thứ tự ngoài → trong:
    `RequestID` → `Logger` → `Recover` → mux.
  - `RequestID`: dùng header `X-Request-ID` nếu client gửi và hợp lệ (≤ 64 ký tự chữ số/chữ/`-`),
    ngược lại sinh 16 byte ngẫu nhiên (`crypto/rand`) dạng hex; gắn vào context và header phản hồi.
  - `Logger`: một dòng slog JSON mỗi request: method, path, status, duration_ms, request_id.
  - `Recover`: bắt panic, log stack, trả 500 `{"error":"internal_error"}`.
  - `http.Server` đặt `ReadHeaderTimeout` 5s, `ReadTimeout` 15s, `WriteTimeout` 15s, `IdleTimeout` 60s.
  - Shutdown: `signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)`; khi có tín hiệu gọi
    `srv.Shutdown` với context 10 giây, sau đó `client.Disconnect` với context 5 giây.
- **Rationale**: Đúng khối 2; `Shutdown` ngừng nhận kết nối mới và chờ request đang xử lý (FR-013).
  Timeout server tránh cảnh báo gosec G112 và request treo vô hạn.
- **Kiểm tra**: tự động qua test middleware; tắt an toàn kiểm tra thủ công (quickstart bước 6) vì phụ
  thuộc tín hiệu hệ điều hành.

## R6. Hiển thị trạng thái kết nối đạt WCAG AA

- **Decision**: Chữ thông báo luôn dùng `--color-text`; màu trạng thái (`--color-ok`, `--color-warn`,
  `--color-bad`) chỉ tô **biểu tượng** và viền trái của thẻ. Mỗi trạng thái có biểu tượng riêng
  (✓ đã kết nối, ! mất cơ sở dữ liệu, ✕ không kết nối máy chủ, vòng xoay khi đang kiểm tra), biểu tượng
  có `aria-hidden`, chữ nằm trong vùng `role="status"` / `aria-live="polite"`.
- **Rationale**: Tính theo WCAG, `--color-warn` sáng `#B97A00` trên nền trắng chỉ ≈ 3.6:1, không đủ
  4.5:1 cho chữ thường nhưng đủ 3:1 cho biểu tượng. `--color-ok` sáng ≈ 4.5:1, sát ngưỡng. Dùng màu cho
  biểu tượng giữ được ý nghĩa màu mà vẫn đạt AA, và thông tin không chỉ nằm ở màu (nguyên tắc IV).
- **Alternatives considered**: Tô màu chữ (trượt AA với warn); thêm token mới (phải sửa design-system
  trước theo nguyên tắc I, không cần thiết).

## R7. Ánh xạ phản hồi `/api/health` sang trạng thái giao diện

- **Decision**: `ApiService.getHealth()` gọi `GET /api/health` với timeout 5 giây. Trang chủ ánh xạ:

  | Kết quả | Trạng thái |
  |---|---|
  | Đang chờ | `checking` → "Đang kiểm tra kết nối…" |
  | 200 và `database: "up"` | `connected` → "Đã kết nối" |
  | 503 và body có `database: "down"` | `database-down` → "Mất kết nối cơ sở dữ liệu" |
  | Lỗi mạng, timeout, 502/504 từ proxy, body không đúng hợp đồng, mã khác | `server-unreachable` → "Không kết nối được máy chủ" |

- **Rationale**: Khi backend tắt, nginx (hoặc dev proxy) trả 502/504 chứ không phải lỗi mạng, nên phải
  phân biệt bằng nội dung body chứ không chỉ mã trạng thái. Timeout 5 giây > 2 giây ping + độ trễ,
  vẫn trong mục tiêu 3 giây ở trường hợp thường.
- **Error interceptor**: chỉ chuẩn hoá `HttpErrorResponse` thành `ApiError { kind: 'network' | 'http',
  status, body }` để mọi tính năng sau xử lý thống nhất; F0 chưa hiện toast hay chuyển trang (401 → đăng
  nhập sẽ thêm ở F1).

## R8. ThemeService và tránh nháy màu khi tải trang

- **Decision**:
  - `ThemeService`: signal `preference` (`'light' | 'dark' | 'system'`), computed `resolved`
    (`'light' | 'dark'`) dựa trên signal `systemDark` từ `matchMedia('(prefers-color-scheme: dark)')`
    (lắng nghe `change`). Một `effect` gắn `data-theme` và `color-scheme` lên `<html>`, và ghi
    localStorage khoá `luna.theme` (đọc/ghi đều trong try/catch; giá trị lạ → `system`).
  - `index.html` có một script inline vài dòng đọc cùng khoá và gắn `data-theme` trước khi Angular
    khởi động, để người đã chọn Tối không thấy nháy nền sáng.
  - `tokens.css`: sáng trong `:root`; tối trong `:root[data-theme="dark"]` và trong
    `@media (prefers-color-scheme: dark) { :root:not([data-theme]) { … } }` (khi JS chưa chạy).
- **UI** (sửa 2026-09-30: thay bằng một nút mở menu CDK `cdkMenu` gồm Sáng, Tối, `role="menuitemradio"`; giá trị `system` chỉ còn là trạng thái "chưa chọn"). Bản gốc: nút trong header là nhóm 3 nút chọn một (`role="radiogroup"`, mỗi nút `role="radio"` +
  `aria-checked`), có biểu tượng + nhãn ẩn cho trình đọc màn hình; vùng bấm tối thiểu 44×44px để vừa
  360px.
- **Alternatives considered**: `<select>` (xấu trên mobile, hai lần chạm); chỉ dùng media query không
  script inline (nháy màu khi đã chọn Tối).

## R9. Docker Compose chạy được trên máy mới mà không cần `.env`

- **Decision**:
  - `deploy/docker-compose.yml` dùng biến có giá trị mặc định, ví dụ
    `MONGO_URI: ${MONGO_URI:-mongodb://mongo:27017}`, `WEB_PORT` mặc định `8000`. `deploy/.env` là tuỳ
    chọn để ghi đè; `.env.example` liệt kê đủ biến. Mongo local không bật xác thực nên giá trị mặc định
    không phải bí mật.
  - `mongo`: `mongo:8`, volume `mongo-data`, cổng `127.0.0.1:27017:27017` (để chạy backend ngoài
    Docker khi phát triển; chỉ bind localhost).
  - `backend`: build `backend/Dockerfile` (golang:1.25 → `gcr.io/distroless/static-debian12:nonroot`,
    `CGO_ENABLED=0`), không mở cổng ra ngoài; `stop_grace_period: 15s` (> 10 giây shutdown).
  - `frontend`: build `frontend/Dockerfile` (node:22 `npm ci` + `ng build` → `nginx:alpine`), cổng
    `${WEB_PORT:-8000}:80`, nginx proxy `/api/` tới `http://backend:8080` và fallback SPA về
    `index.html`. Dùng `resolver` của Docker để nginx không chết khi backend chưa lên.
- **Rationale**: SC-001 yêu cầu đúng một lệnh trên máy mới; bắt buộc tạo `.env` trước là thêm một bước.
  Cổng web 8000 tránh đụng cổng 8080 của backend khi chạy riêng.
- **Alternatives considered**: `env_file: .env` bắt buộc (thêm bước); mở cổng backend ra host (không cần,
  mọi truy cập đi qua nginx).

## R10. Font đóng gói và subset

- **Decision**: Import CSS theo subset trong `styles.css`:
  `@fontsource-variable/lexend/wght.css` được chia file theo `unicode-range` nên trình duyệt chỉ tải
  subset cần; chỉ import các file subset `latin`, `latin-ext`, `vietnamese` của Lexend và Noto Sans
  (400) thay vì file `index.css` gộp. Font được build vào `dist` và phục vụ từ cùng origin.
- **Rationale**: Nguyên tắc II (OFL), FR-008 (hiển thị offline), design-system mục 4 (chỉ nạp subset cần).
  Noto Sans chưa dùng ở F0 (chưa có IPA) nhưng được khai báo token `--font-ipa` để tính năng F3 dùng; chỉ
  cài gói, chưa import vào bundle.
- **Alternatives considered**: Google Fonts CDN (vi phạm offline); tự tải file .woff2 vào repo (khó cập
  nhật).

## R11. Tạo frontend bằng Angular CLI

- **Decision**: `npx ng new luna --directory frontend --prefix lu --style css --routing --ssr false
  --ai-config claude --interactive false`, sau đó `ng add angular-eslint`. Giữ mặc định Angular 21:
  standalone, zoneless, Vitest. Tạo file bằng `ng generate` theo `docs/architecture.md`.
  `proxy.conf.json` cho `ng serve` chuyển `/api` tới `http://localhost:8080`.
- **Rationale**: Làm theo skill angular-new-app và architecture.md.
- **Test không cần mạng**: Vitest chạy trong jsdom sau `npm ci`; `HttpClient` được giả lập bằng
  `provideHttpClientTesting`.
