# Luna: Cấu trúc mã nguồn

> Phiên bản: v1 (2026-09-29). Dùng làm đầu vào cho `/speckit.plan`.

## 1. Tổng quan repo

```
Luna/
├── frontend/        ← Angular
├── backend/         ← Go
├── deploy/          ← docker-compose, sao lưu (F13)
└── docs/
```

## 2. Frontend (Angular)

**Quy ước**
- Standalone component + signals, không dùng NgModule. TypeScript strict.
- Chia theo **tính năng**; mỗi tính năng lazy load theo route.
- Interceptor và guard viết dạng hàm (`HttpInterceptorFn`, `CanActivateFn`).
- Tạo file bằng `ng generate` (component, service, guard, interceptor, pipe...).
- Chỉ có **một** thư mục hàm tiện ích: `shared/utils/` (không tách `helpers/`).
- Màu, font, cỡ chữ chỉ dùng token trong `styles/tokens.css` ([design-system.md](design-system.md)).

```
frontend/src/
├── app/
│   ├── core/                 ← khởi tạo một lần, dùng toàn app
│   │   ├── interceptors/     ← auth (gắn token), error (thông báo lỗi, 401 → đăng nhập)
│   │   ├── guards/           ← auth, admin
│   │   ├── services/         ← api, auth, theme
│   │   └── models/           ← Lesson, Sentence, Card, Progress, Settings...
│   ├── shared/               ← dùng lại giữa các tính năng
│   │   ├── components/       ← progress-bar, step-indicator, word-popup, waveform...
│   │   ├── pipes/
│   │   ├── directives/
│   │   └── utils/            ← hàm thuần, có unit test (so sánh chép chính tả, tách từ...)
│   ├── features/
│   │   ├── auth/             ← F1
│   │   ├── admin/            ← F2
│   │   ├── lesson/           ← L: review/, reading/ (F3), listening/ (F4)
│   │   ├── vocabulary/       ← F5
│   │   ├── home/             ← F6
│   │   └── settings/         ← F12, F13
│   ├── app.routes.ts
│   └── app.config.ts
└── styles/
    └── tokens.css
```

## 3. Backend (Go)

**Quy ước**
- Theo bộ skill `samber/cc-skills-golang` trong `.claude/skills/` (layout, naming, error handling, testing, lint...).
- Kiến trúc: **chia theo domain + ports**. Mỗi domain là một package gồm handler, service, và interface repository.
- Chỉ tách interface ở hai chỗ constitution bắt buộc: **cơ sở dữ liệu** (repository) và **AI** (provider).
- Dependency injection: **tự viết constructor** `NewXxx(...)`, nối mọi thứ trong `internal/service` (mỗi domain một hàm `initXxx` trong file riêng, gọi theo thứ tự phụ thuộc bởi `service.Init`); `cmd/server/main.go` chỉ mở CSDL, gọi `service.Init` rồi chạy HTTP. Không dùng thư viện DI.
- 12-factor: cấu hình đọc từ biến môi trường, log JSON ra stdout (`slog`), tắt server an toàn (graceful shutdown).
- Test đặt cạnh file code (`xxx_test.go`), dữ liệu mẫu trong `testdata/`.

```
backend/
├── cmd/
│   └── server/main.go        ← đọc config, mở CSDL, gọi service.Init, chạy HTTP và tắt an toàn
├── internal/
│   ├── service/              ← nối các domain: Init, các initXxx, adapter giữa domain, route, worker
│   ├── auth/                 ← F1
│   ├── lesson/               ← F2: bài học, lộ trình, tách câu
│   ├── vocab/                ← F5: thẻ, lịch ôn FSRS
│   ├── progress/             ← L, F6: luồng một ngày học, streak, thống kê
│   ├── settings/             ← F12
│   ├── dictionary/           ← F3: tra từ điển SQLite
│   ├── ai/                   ← interface Provider + gemini/, openrouter/, ollama/
│   ├── grammar/              ← F19–F21: giáo trình ngữ pháp (syllabus.json nhúng), bài ngữ pháp, bài tập, tiến độ người học, kiểm tra AI, báo lỗi
│   ├── wordmatch/            ← F18: khớp từ, cụm từ trong văn bản
│   ├── job/                  ← chạy nền: chú thích AI, phần luyện tập, chấm bài viết (trạng thái đang chạy/xong/lỗi)
│   ├── storage/              ← Store (abstract factory) + Registry chọn CSDL theo DB_DRIVER
│   │   ├── factory/          ← đăng ký các CSDL; main chỉ gọi factory.Open(cfg)
│   │   ├── mongo/            ← Store và các repository cho MongoDB
│   │   └── mysql/            ← Store và các repository cho MySQL 8 (database/sql)
│   └── platform/
│       ├── config/
│       ├── httpx/            ← router, middleware (auth, admin, log, recover)
│       └── logger/
├── Makefile
├── .golangci.yml
└── go.mod
```

**Mỗi domain gồm**

| File | Vai trò |
|---|---|
| `handler.go` | Nhận HTTP, kiểm tra đầu vào, gọi service, trả JSON |
| `service.go` | Logic nghiệp vụ; chỉ phụ thuộc interface |
| `repository.go` | Interface truy cập dữ liệu (hiện thực nằm ở `storage/mongo/`) |
| `model.go` | Kiểu dữ liệu của domain |
| `*_test.go` | Test service với repository giả lập |

### Lớp lưu trữ: chọn CSDL bằng biến môi trường

Domain chỉ biết interface repository. Việc chọn MongoDB hay MySQL nằm ở `internal/storage`, theo hai mẫu thiết kế:

- **Abstract Factory** (`storage.Store`): một `Store` là một CSDL đã mở, và tạo ra cả họ repository của app (`Users()`, `Lessons()`,
  `Cards()`...). Ba việc không thuộc repository nào cũng nằm ở đây vì mỗi CSDL làm khác nhau: `Ping` (kiểm tra sức khoẻ), `Prepare`
  (tạo index hoặc bảng, chạy migration và nạp dữ liệu ban đầu, ở nền và thử lại khi CSDL chưa lên) và `Close`.
- **Registry** (`storage.Registry`): ánh xạ tên (`mongo`, `mysql`) sang hàm mở `Store`. `storage/factory` là nơi duy nhất biết đủ các
  loại CSDL và biến cấu hình của từng loại; `main.go` chỉ gọi `factory.Open(cfg)` rồi lấy repository từ `Store`. Không dùng `init()` hay
  biến toàn cục để đăng ký, nên test tự dựng registry riêng.

```
DB_DRIVER ─► Registry.Open ─► mongo.Open(MONGO_URI, MONGO_DATABASE) ─► *mongo.Store ┐
                           └► mysql.Open(MYSQL_DSN)                 ─► *mysql.Store ┴─► storage.Store ─► services
```

**Thêm một loại CSDL** (cách MongoDB và MySQL đã làm):

1. Tạo `internal/storage/<tên>/` với một kiểu thoả `storage.Store` (xem `mongo/store.go`, có sẵn dòng kiểm tra lúc biên dịch
   `var _ storage.Store = (*Store)(nil)`), và hiện thực từng repository theo hành vi mà test của domain mô tả.
2. Thêm một dòng `reg.Register("<tên>", ...)` trong `internal/storage/factory/factory.go` và biến cấu hình trong `platform/config` (chỉ bắt buộc khi
   `DB_DRIVER` là loại đó).
3. Phần riêng của từng CSDL phải làm kèm: lược đồ và migration (`Prepare`), kiểu id, và công cụ sao lưu trong `deploy/backup`.

Quy ước giữ cho các hiện thực hoán đổi được: id là chuỗi đục đối với domain; thời gian lưu UTC; lỗi "không có" trả `ErrNotFound` của
domain (không để lộ lỗi của driver); các thao tác "ghi nếu còn đúng phiên bản" (ví dụ `SetStatus`, `UpdateSchedule`) phải nguyên tử.

**MySQL** (`internal/storage/mysql`) làm theo các quy ước đó, thêm vài lựa chọn riêng:

- Id là chuỗi hex 24 ký tự do app sinh (`newID`, cùng dạng với ObjectID của Mongo), cột `CHAR(24)`. Thời gian là `DATETIME(6)` UTC;
  MySQL làm tròn phần giây xuống micro giây khi lưu.
- Bảng dùng InnoDB, `utf8mb4`, collation nhị phân (so sánh chính xác như Mongo); tìm không phân biệt hoa thường dùng `LOWER(...)`.
- Tài liệu lồng nhau chỉ đọc-ghi nguyên khối (chú thích, câu hỏi, luyện tập, tiêu chí chấm) là cột `JSON`; thứ được lọc, sắp xếp hay
  phải duy nhất là cột thật. Không có khoá ngoại giữa các domain, nên mỗi migration đứng riêng.
- "Ghi nếu còn đúng phiên bản" là một câu `UPDATE ... WHERE`; `clientFoundRows` được bật để ghi lại đúng dữ liệu đang có vẫn tính là khớp.
  Hàng đợi `jobs` nhận việc bằng `FOR UPDATE SKIP LOCKED`.
- Mongo xoá phiên hết hạn bằng TTL index; MySQL coi phiên quá hạn là không tồn tại và xoá chúng khi tạo phiên mới.
- Cần MySQL 8.0.19 trở lên (upsert dùng alias `INSERT ... AS new`). Lược đồ ở các hàm `xxxSchema()` và được ghi nhận trong bảng
  `schema_migrations`; sửa bảng sau khi phát hành thì thêm một migration mới ở cuối `migrations()`, không sửa cái đã chạy.
- Test tích hợp dùng MySQL thật qua `MYSQL_TEST_DSN` (mỗi test một database riêng) và tự bỏ qua nếu thiếu biến này.

### Nối các domain: `internal/service`

Mọi thứ được nối ở một chỗ, theo cách tự viết constructor (không dùng thư viện DI):

- `service.Init(ctx, cfg, log, store)` trả một `Container` và chạy các hàm khởi tạo theo thứ tự phụ thuộc: `initAuth`, `initDictionary`,
  `initAI`, `initSettings`, `initTopic`, `initLesson`, `initWriting`, `initWorker`, `initReader`, `initVocab`, `initProgress`, `initExport`.
  Mỗi hàm nằm trong file cùng tên domain (`lesson.go`, `vocab.go`...) và chỉ đọc những service đã được tạo trước nó.
- Các **adapter** giữa domain (ví dụ `lessonTopicsPort`, `dailyReviews`) nằm cạnh domain dùng chúng, để hai domain không import nhau.
  Hai domain phụ thuộc vòng (viết bài và luồng học) được nối bằng một adapter nhận service sau (`writingSteps`, gán trong `initProgress`).
- `Container.Handler()` đăng ký toàn bộ route (`routes.go`) và bọc middleware; `StartWorker(ctx)` chạy worker nền; `Close()` đóng từ điển.
  Thêm một domain mới: thêm file `initXxx` và adapter nếu cần, gọi nó trong `Init`, thêm dòng route trong `routes.go`.
- `cmd/server/main.go` giữ phần hạ tầng: đọc cấu hình, mở và đóng CSDL (`factory.Open`), gọi `service.Init`, chạy `http.Server` và tắt an toàn.
  Thứ tự đóng khi tắt: worker dừng, rồi `Container.Close`, rồi CSDL.
- `service_test.go` dựng cả `Container` (trên một store Mongo không kết nối) và kiểm tra mọi route còn được đăng ký.
