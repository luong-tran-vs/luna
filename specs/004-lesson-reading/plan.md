# Implementation Plan: Đọc

**Branch**: `004-lesson-reading` | **Date**: 2026-09-29 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/004-lesson-reading/spec.md` + khối 2 trong `docs/spec-inputs/f3-doc.md`

**Note**: This template is filled in by the `/speckit-plan` command. See `.specify/templates/plan-template.md` for the execution workflow.

## Summary

Bước Đọc: người học xem bài theo đoạn và câu, chạm từ hoặc chọn cụm để tra, nghe phát âm, lưu vào sổ từ và bấm "Đã đọc
xong" khi đã cuộn hết bài. Backend thêm: `internal/dictionary` (SQLite `dictionary.db` của minhqnd/dictionary, chỉ đọc,
driver Go thuần `modernc.org/sqlite`; hàm đưa về dạng gốc), endpoint học `GET /api/lessons/{id}` và tra cứu
`GET /api/lessons/{id}/lookup` (chú thích → từ điển → 404, không gọi AI), phát âm từ `GET /api/tts/word` (Kokoro, cache
file theo hash), `internal/vocab` (sổ từ tối thiểu: `cards`, unique `(userId, lemma)`). Frontend thêm trang
`/lessons/:id/read` (tách token ở client, chọn cụm, popup định vị bằng CDK Overlay, IntersectionObserver cho nút Đã đọc
xong) và liên kết mở từ trang chi tiết bài của admin.

## Technical Context

**Language/Version**: Go 1.25; TypeScript 5.9 strict + Angular 21

**Primary Dependencies**:
- Backend **mới**: `modernc.org/sqlite` **v1.59.0** (Go thuần, không CGO; v1.60 yêu cầu Go 1.26); `golang.org/x/sync/singleflight` (đã có trong module graph)
- Frontend: `@angular/cdk/overlay` (đã cài ở F2); `@fontsource/noto-sans` (đã cài ở F0, nay import cho IPA)
- Dữ liệu: `dictionary.db` v2.0.0 của minhqnd/dictionary (MIT, 179 MB, 118 nghìn từ tiếng Anh có nghĩa tiếng Việt)

**Storage**: MongoDB: **mới** `cards`. File: `dictionary.db` (chỉ đọc), `{AUDIO_DIR}/words/{sha256}.mp3`.

**Testing**: Go `testing` với repository/dictionary giả + một SQLite nhỏ tạo trong test (schema giống thật); Vitest.

**Target Platform / Project Type**: như F0–F2 (web app, Docker, không GPU)

**Performance Goals**: tra cứu < 300ms từ lúc chạm tới lúc hiện (SC-001); truy vấn SQLite dùng index `(word, lang_code)`.

**Constraints**: không gọi AI khi tra; sổ từ tách theo người học; chạy offline trừ lần tải từ điển; 360px.

**Scale/Scope**: 1 người học; bài ≤ 200 câu; từ điển 118k mục.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

Đối chiếu `.specify/memory/constitution.md` v1.0.0.

- [x] **I. docs/ là nguồn sự thật**: F3 theo `docs/phases/`; cấu trúc `internal/dictionary`, `internal/vocab`,
  `features/lesson/reading`, `shared/components/word-popup`, `shared/utils/tokenize.ts` đúng `docs/architecture.md`;
  spec bao phủ 4 tiêu chí nghiệm thu F3.
- [x] **II. Chi phí 0 đồng**: từ điển MIT, `modernc.org/sqlite` BSD, Kokoro Apache 2.0, font OFL.
- [x] **III. AI là phần bổ sung**: tra cứu không gọi AI; chú thích AI chỉ đọc từ dữ liệu đã lưu; thiếu chú thích vẫn tra
  được bằng từ điển hoặc tự nhập.
- [x] **IV. Mobile-first**: popup tự định vị trong màn hình, không che từ; token có vùng chạm đủ lớn nhờ line-height 30px;
  từ đã lưu có gạch chân ngoài màu; IPA dùng `--font-ipa`.
- [x] **V. Dữ liệu người học**: `cards` luôn lọc theo `userId` lấy từ phiên (không nhận userId từ client); truy cập DB qua
  `vocab.Repository`; từ điển qua interface `lesson.Dictionary`.
- [x] **VI. Kiểm thử**: unit test đưa về dạng gốc, thứ tự tra cứu, lưu trùng, tách token, tính dạng gốc cho bài; handler
  test thành công / lỗi đầu vào / chưa đăng nhập / người khác không thấy sổ từ; test không cần mạng (SQLite nhỏ tạo trong
  test, Kokoro giả).
- [x] **VII. Đơn giản trước**: dạng gốc bằng quy tắc + bảng nhỏ (không thư viện NLP); không SDK; một trang đọc.

**Kết quả**: ĐẠT, không có ngoại lệ.

## Project Structure

### Documentation (this feature)

```text
specs/004-lesson-reading/
├── plan.md, research.md, data-model.md, quickstart.md
├── contracts/
│   ├── reading-api.md       # GET /api/lessons/{id}, lookup, /api/tts/word
│   ├── vocab-api.md         # POST /api/vocab/cards, GET /api/vocab/words
│   └── config.md            # DICTIONARY_PATH, script tải từ điển
└── tasks.md
```

### Source Code (repository root)

```text
backend/
├── cmd/api/main.go                       # + dictionary, vocab, reading routes, word TTS
└── internal/
    ├── dictionary/
    │   ├── sqlite.go (+ _test)           # Open(path) chỉ đọc, Lookup(word) → Entry
    │   ├── lemma.go (+ _test)            # Candidates(word): nguyên dạng → dạng gốc
    │   └── irregular.go                  # bảng động từ/danh từ bất quy tắc
    ├── lesson/
    │   ├── reading.go (+ _test)          # ReadingView, Lookup, LemmaMap (dùng Dictionary port)
    │   ├── reading_handler.go (+ _test)  # GET /api/lessons/{id}, /lookup
    │   └── tokenize.go (+ _test)         # cùng quy tắc token với frontend
    ├── vocab/
    │   ├── model.go, repository.go
    │   ├── service.go (+ _test)          # Save (chống trùng theo lemma), Words
    │   └── handler.go (+ _test)          # POST /api/vocab/cards, GET /api/vocab/words
    ├── tts/wordcache.go (+ _test)        # WordAudio: cache theo sha256, singleflight
    └── storage/mongo/cards.go            # + index unique (userId, lemma)

frontend/src/app/
├── core/models/reading.ts, vocab.ts
├── shared/utils/tokenize.ts (+ spec)
├── shared/components/word-popup/ (+ spec)
├── features/lesson/reading/ (+ spec)     # trang đọc, chọn cụm, tô màu, Đã đọc xong
├── features/lesson/reading-api.service.ts (+ spec)
├── features/lesson/lesson.routes.ts      # 'lessons/:id/read'
└── features/admin/lesson-detail/         # + nút "Mở bước Đọc"

deploy/
├── docker-compose.yml                    # + bind mount ./data/dictionary:/data/dictionary:ro
├── fetch-dictionary.sh, fetch-dictionary.ps1
└── data/                                 # bị git bỏ qua
```

**Structure Decision**: Đọc là phần của domain `lesson` (dữ liệu bài) + `vocab` (sổ từ) + `dictionary` (hạ tầng tra từ).
`lesson` chỉ phụ thuộc interface `Dictionary` và `Lemmatizer` nhỏ, hiện thực ở `internal/dictionary`.

## Post-Design Constitution Check

Thêm 1 thư viện Go (SQLite thuần Go, bắt buộc để đọc từ điển offline), 2 interface (`lesson.Dictionary`,
`vocab.Repository`), 1 collection. Không thêm lớp trừu tượng khác. **Vẫn ĐẠT.**

## Complexity Tracking

Không có vi phạm cần giải trình.
