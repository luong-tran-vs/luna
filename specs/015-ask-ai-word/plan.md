# Implementation Plan: Hỏi AI về từ

**Branch**: `015-ask-ai-word` | **Date**: 2026-10-01 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/015-ask-ai-word/spec.md` + khối 2 trong `docs/spec-inputs/f9-hoi-ai.md`

## Summary

**Backend.** `ai.Provider.Explain` gửi 1 request Gemini, schema gồm `{lemma, meaningVi, noteVi}`. Route
`POST /api/lessons/{id}/ask` dùng chung guard L với F3:

1. Kiểm tra cụm từ có trong câu, theo giới hạn của F3.
2. Tìm trong `ai_lookups` (dùng chung, khoá unique `(lessonId, revision, sentenceIndex, textLower)`). Có thì trả ngay.
3. Chưa có thì gọi AI qua `singleflight` theo khoá: nhiều yêu cầu cùng lúc chỉ gọi 1 lần. Context tách khỏi request, timeout 12 giây.
4. Lưu kết quả rồi trả về.

**Lỗi AI**: 503 khi chưa cấu hình hoặc khoá sai, 429 khi hết lượt, 502 với các lỗi khác. Lỗi không bao giờ được lưu vào `ai_lookups`.

**Tra từ (F3).** `GET /lookup` trả kết quả đã lưu (nguồn `ai`, có `note`) trước khi tra từ điển.

**Frontend.** Popup tra từ có nút **Hỏi AI** khi nguồn không phải AI, kèm trạng thái đang hỏi, phần giải thích và chỗ hiện lỗi. Trang
Đọc gọi API rồi thay kết quả trong popup. Lưu vào sổ từ dùng sẵn kết quả đang hiện, nên ra đúng nghĩa AI.

## Technical Context

**Language/Version**: Go 1.25.1 (`GOTOOLCHAIN=local`); TypeScript 5.9 strict + Angular 21

**Primary Dependencies**:
- `golang.org/x/sync/singleflight` (module đã có trong `go.mod`; chỉ dùng thêm gói con).
- Gemini `generateContent`.

**Storage**: collection mới `ai_lookups`, 1 index unique. Không đổi collection nào khác.

**Testing**:
- Go: `lesson` (Ask có cache, gọi đồng thời, revision, kiểm tra đầu vào, lỗi; Lookup ưu tiên kết quả đã lưu; handler), `ai/gemini`
  (Explain), `storage/mongo` (chuyển đổi doc).
- Vitest: `word-popup` (nút, trạng thái, note, lỗi), `reading` (ask → lưu thẻ dùng nghĩa AI, lỗi, bỏ kết quả cũ), `reading-api`
  (ask).

**Target Platform / Project Type**: web app như các phần trước

**Performance Goals**:
- Cache hit < 1 giây (SC-002).
- Lần hỏi đầu < 10 giây (SC-004). Timeout 12 giây nằm dưới 15 giây của server và nginx.

**Constraints**:
- Chỉ gọi AI khi người học bấm nút (FR-002).
- 1 lần gọi cho mỗi khoá (FR-008).
- AI lỗi không chặn việc lưu thẻ (FR-010).

**Scale/Scope**: vài người học. Mỗi bài có tối đa vài chục kết quả hỏi AI.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

Đối chiếu `.specify/memory/constitution.md` v1.0.0.

- [x] **I. docs/ là nguồn sự thật**: F9 theo `docs/phases/giai-doan-2.md` (2 tiêu chí nghiệm thu). Code nằm trong `internal/ai`,
  `internal/lesson` và `shared/components/word-popup`.
- [x] **II. Chi phí 0 đồng**: gói AI miễn phí. Kết quả được lưu dùng chung, nên mỗi khoá chỉ tốn 1 lần gọi.
- [ ] **III. AI là phần bổ sung**:
  - Đạt:
    - gọi qua `ai.Provider`;
    - AI lỗi không ảnh hưởng tra từ hay lưu thẻ;
    - có trạng thái đang hỏi / lỗi, bấm lại được;
    - kết quả được lưu để không gọi lặp.
  - **Ngoại lệ**: gọi đồng bộ trong request, không qua job nền. Giải trình ở Complexity Tracking.
- [x] **IV. Mobile-first**:
  - nút ≥ 44px;
  - trạng thái có `role="status"`, lỗi có `role="alert"`;
  - không đổi cách định vị popup.
- [x] **V. Dữ liệu người học**:
  - `ai_lookups` không có `userId`;
  - AI chỉ nhận câu của bài và cụm từ, không nhận dữ liệu người học;
  - log không ghi câu.
- [x] **VI. Kiểm thử**: cache, gọi đồng thời, revision, lỗi, kiểm tra đầu vào và giao diện đều có test (R7).
- [x] **VII. Đơn giản trước**:
  - một collection, một route;
  - singleflight trong tiến trình, không có khoá phân tán;
  - không sửa luồng lưu thẻ.

**Kết quả**: ĐẠT, có một ngoại lệ (nguyên tắc III, chạy nền) đã giải trình.

## Project Structure

### Documentation (this feature)

```text
specs/015-ask-ai-word/
├── plan.md, research.md, data-model.md, quickstart.md
├── contracts/ask-api.md
├── checklists/requirements.md
└── tasks.md                                     # /speckit-tasks
```

### Source Code (repository root)

```text
backend/
├── cmd/api/main.go                              # NewReader(..., mongo.NewAILookups(db), aiProvider)
└── internal/
    ├── ai/ai.go, ai/gemini/gemini.go (+ _test)  # ExplainRequest, Explanation, Explain
    ├── lesson/
    │   ├── ask.go (+ _test)                     # MỚI: AskKey, AskResult, AskRepository, Reader.Ask, singleflight
    │   ├── ask_handler.go (+ _test)             # MỚI: POST /api/lessons/{id}/ask, ánh xạ lỗi
    │   ├── reading.go, reading_handler.go       # Reader deps; Lookup dùng ai_lookups; lookupJSON + note
    │   └── fake_test.go                         # fake asks, fake AI Explain
    └── storage/mongo/
        ├── ai_lookups.go (+ _test)              # MỚI: Get, Put (trùng khoá → bản đã có)
        └── indexes.go                           # unique index ai_lookups

frontend/src/app/
├── core/models/reading.ts                       # note, AskResult
├── features/lesson/reading-api.service.ts (+ spec)   # ask()
├── features/lesson/reading/reading.ts/.html (+ spec) # xử lý ask
└── shared/components/word-popup/ (+ spec)       # nút Hỏi AI, đang hỏi, note, lỗi

README.md                                        # mục bước Đọc: Hỏi AI
docs/phases/giai-doan-2.md                       # tick tiêu chí F9 khi xong
```

**Structure Decision**:
- Hỏi AI là một phần của việc tra từ ở bước Đọc, nên nằm trong `lesson.Reader`. `Reader` đã có sẵn bài, câu, từ điển và guard L.
- Kết quả là dữ liệu của bài, dùng chung cho mọi người học, nên không thuộc gói `vocab` (sổ từ riêng của từng người).

## Post-Design Constitution Check

- Thêm 1 method trên Provider, 1 collection, 1 route, 1 nút.
- Không thêm thư viện mới: `x/sync` đã có.
- Ngoại lệ ở nguyên tắc III giữ nguyên.

**Vẫn ĐẠT.**

## Complexity Tracking

| Điểm | Vì sao cần | Phương án đơn giản hơn / đúng nguyên tắc hơn bị loại vì |
| --- | --- | --- |
| Nguyên tắc III: Hỏi AI gọi đồng bộ trong request (≤ 12 giây), không qua job nền | Người học đang đọc và chờ nghĩa ngay trong popup. Kết quả vẫn được lưu (không gọi lặp), có trạng thái đang hỏi / lỗi, bấm lại được, và AI lỗi không chặn gì khác | Job nền kèm poll hoặc thông báo sẽ khiến popup phải chờ và tự cập nhật sau vài giây, trong khi người học có thể đã đọc sang câu khác. Phải thêm job type, trạng thái chờ và poll cho một thao tác chỉ mất vài giây |
