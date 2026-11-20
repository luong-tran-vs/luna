# Nội dung Luyện thi TOEIC (F27)

Claude soạn các file JSON ở đây, admin nhập ở **Quản trị → TOEIC → Nhập JSON** (hoặc
`POST /api/admin/toeic/import?dryRun=true` rồi nhập thật). Định dạng: `specs/022-toeic-practice/contracts/content-format.md`.

```text
part1/ … part7/   # mỗi file { "groups": [ … ] }, 5–30 cụm
tests/            # mỗi file một đề { key, title, kind, order, groups } (Quản trị → TOEIC → Tạo đề)
```

## Quy tắc soạn

- Tự viết mới hoàn toàn: **không chép đề ETS** hay sách đã xuất bản. Tên người, công ty, địa chỉ, website là hư cấu.
- Câu hỏi, văn bản, lời nói bằng tiếng Anh; `explainVi`, `translateVi`, `meaningVi` bằng tiếng Việt.
- Mỗi câu **đúng một đáp án**; phương án nhiễu hợp lý (từ loại khác, sai thì, từ nghe giống, từ lặp lại trong bài) nhưng không mơ hồ.
- `explainVi` nêu vì sao đúng, vì sao từng phương án kia sai và bẫy thường gặp.
- `evidence.text` trích **nguyên văn** từ khối (`doc`, `block`) hoặc lượt nói (`doc: -1`, `turn`).
- `key`: chữ thường, số, gạch ngang, duy nhất toàn hệ thống (ví dụ `p5-s01-07`, `p7-s01-double-01`).
- Dạng câu (`tags`) lấy trong danh sách cố định ở `backend/internal/toeic/tags.go`, đúng Part.

## Kiểm tra trước khi nhập

```powershell
cd backend; go test ./internal/toeic -run TestContentFiles
```

Test này chạy đúng các kiểm tra của lần nhập chạy thử trên mọi file ở đây, và kiểm tra mỗi file đề có đủ câu từng Part.

## Cụm do AI soạn

Quản trị → TOEIC → Kho cụm câu → **Nhờ AI soạn**: AI viết cụm theo đúng định dạng này, cụm qua kiểm tra lưu thành nháp
(mã `p5-ai-…`, nhãn AI). Các cụm này không nằm trong thư mục này; luôn xem trước kỹ đáp án và bản dịch trước khi đăng.
