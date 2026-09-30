# Đầu vào cho F0: Khung dự án

Nguồn: [giai-doan-1.md › F0](../phases/giai-doan-1.md), [architecture.md](../architecture.md), [design-system.md](../design-system.md).

Máy hiện có: Go 1.25.1, Node 22.19, Angular CLI 21.0.3, Docker 28.4 + Compose v2. **Chưa có** golangci-lint.

---

## Khối 1: dán vào `/speckit-specify`

Chỉ mô tả **làm gì và vì sao**, không nói công nghệ.

```text
F0 – Khung dự án (Giai đoạn 1) cho Luna, web app học tiếng Anh cá nhân.

Mục tiêu: có một bộ khung chạy được từ đầu đến cuối (giao diện, máy chủ, cơ sở dữ liệu) để các tính năng sau (F1 trở đi) chỉ việc thêm vào. Chưa có tính năng nghiệp vụ nào.

Người dùng thấy gì:
- Mở app trên trình duyệt thấy thanh trên cùng có tên "Luna" và một trang chủ tạm.
- Trang chủ tạm cho biết app có kết nối được tới máy chủ và cơ sở dữ liệu hay không: "Đã kết nối", "Mất kết nối cơ sở dữ liệu", hoặc "Không kết nối được máy chủ".
- Có nút chọn giao diện hiện chế độ đang dùng; bấm vào mở menu Sáng / Tối (khi chưa chọn thì theo thiết bị). Lựa chọn có hiệu lực ngay và được nhớ trên trình duyệt đó sau khi tải lại trang.
- Toàn bộ chữ trên giao diện bằng tiếng Việt, hiển thị đúng dấu kể cả khi mất mạng.
- Dùng tốt trên điện thoại từ 360px, không cuộn ngang; màu, font, cỡ chữ đúng tài liệu design-system; độ tương phản đạt WCAG AA ở cả hai chế độ.

Người phát triển cần gì:
- Máy mới chỉ cần cài Docker, chạy một lệnh là toàn bộ app chạy được; có hướng dẫn trong README.
- Có cách chạy riêng từng phần khi phát triển (giao diện tự tải lại khi sửa code).
- Cấu hình bằng biến môi trường, có file mẫu; repo không chứa bí mật. Thiếu cấu hình bắt buộc thì máy chủ báo lỗi rõ ràng và dừng.
- Có sẵn test mẫu và kiểm tra lint cho cả giao diện và máy chủ, chạy được không cần mạng.
- Dừng máy chủ thì tắt an toàn, không cắt ngang yêu cầu đang xử lý.

Ngoài phạm vi: tài khoản và đăng nhập (F1), mọi tính năng học, dịch vụ AI, chuyển văn bản thành giọng nói, nhận dạng giọng nói, từ điển, sao lưu.

Tiêu chí nghiệm thu: theo mục F0 trong docs/phases/giai-doan-1.md.
```

---

## Khối 2: dán vào `/speckit-plan`

Chạy sau khi `/speckit-specify` (và `/speckit-clarify` nếu cần) xong.

```text
Làm theo docs/architecture.md và constitution. Các quyết định kỹ thuật cho F0:

Cấu trúc repo: frontend/ (Angular), backend/ (Go), deploy/ (docker-compose.yml, .env.example), README.md ở gốc.

Frontend
- Angular 21: tạo bằng `ng new` trong thư mục frontend/ (standalone, routing, style CSS, zoneless mặc định, test runner mặc định Vitest), prefix component "lu".
- Cấu trúc core/ shared/ features/ theo architecture.md; F0 chỉ cần: core/services/theme.service.ts, core/services/api.service.ts, core/interceptors/error.interceptor.ts (dạng HttpInterceptorFn), features/home/ (trang chủ tạm), shared/components/app-header.
- ThemeService dùng signal, giá trị light | dark | system (system = chưa chọn, không hiện trên giao diện); nút header dùng @angular/cdk/menu, lưu localStorage (bọc try/catch), gắn data-theme lên <html>, theo dõi prefers-color-scheme khi chọn system.
- src/styles/tokens.css: toàn bộ token trong docs/design-system.md (màu sáng/tối, cỡ chữ, bo góc, khoảng cách); component chỉ dùng var(--...).
- Font đóng gói bằng npm: @fontsource-variable/lexend, @fontsource/noto-sans (chỉ subset latin, latin-ext, vietnamese).
- Gọi API qua proxy của dev server (/api → backend) khi phát triển; trong Docker do nginx proxy.
- Lint: angular-eslint. Test: Vitest cho ThemeService và trang chủ (3 trạng thái kết nối).

Backend
- Go 1.25, module github.com/luongtran/luna/backend, layout theo skill golang-project-layout: cmd/api/main.go, internal/platform/{config,httpx,logger}, internal/health/, internal/storage/mongo/.
- HTTP: thư viện chuẩn net/http (ServeMux có method pattern), không dùng framework. Middleware: recover, request log, request id.
- Endpoint: GET /api/health → 200 {"status":"ok","database":"up"} hoặc 503 {"status":"degraded","database":"down"}; kiểm tra MongoDB bằng ping có timeout 2 giây.
- Config: đọc biến môi trường bằng code tự viết trong internal/platform/config (không dùng viper); bắt buộc MONGO_URI; tuỳ chọn HTTP_ADDR (mặc định :8080), LOG_LEVEL.
- Log: log/slog JSON ra stdout.
- MongoDB driver v2 (go.mongodb.org/mongo-driver/v2). health phụ thuộc interface nhỏ (Pinger) để test không cần Mongo thật.
- Graceful shutdown: bắt SIGINT/SIGTERM, http.Server.Shutdown với timeout 10 giây, đóng kết nối Mongo.
- DI bằng constructor viết tay trong main.go.
- Test: thư viện testing chuẩn + httptest; test handler health với Pinger giả (up/down) và test config thiếu MONGO_URI.
- Lint: .golangci.yml theo skill golang-lint; chạy golangci-lint qua Docker image chính thức trong Makefile (máy chưa cài golangci-lint).
- Makefile: run, test, lint, build.

Deploy
- deploy/docker-compose.yml: mongo (image mongo:8, volume dữ liệu), backend (Dockerfile multi-stage, image cuối distroless), frontend (build Angular, phục vụ bằng nginx, proxy /api tới backend).
- deploy/.env.example; .env nằm trong .gitignore.
- Một lệnh chạy toàn bộ: docker compose -f deploy/docker-compose.yml up --build.

Kiểm tra thủ công trong tasks: 360px sáng/tối, tắt container mongo để thấy trạng thái "Mất kết nối cơ sở dữ liệu", tắt mạng để kiểm tra font.
```
