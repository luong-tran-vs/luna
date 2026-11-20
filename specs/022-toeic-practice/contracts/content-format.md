# Định dạng nội dung TOEIC (file trong `content/toeic/`)

Claude soạn, admin nhập bằng `POST /api/admin/toeic/import`. Một file = `{ "groups": [ … ] }`. Đề là file riêng trong
`content/toeic/tests/` dùng cho `POST /api/admin/toeic/tests`.

Quy ước: câu hỏi, văn bản, lời nói bằng tiếng Anh; `explainVi`, `translateVi`, `meaningVi` bằng tiếng Việt. Nội dung tự viết,
không chép đề ETS hay sách đã xuất bản. Tên người, công ty, địa chỉ là hư cấu.

## Thư mục

```text
content/toeic/
├── part1/  part2/  part3/  part4/  part5/  part6/  part7/   # mỗi file 5–20 cụm
└── tests/luna-test-01.json, mini-01.json                   # { key, title, kind, order, groups: [...] }
```

## Ví dụ theo Part

```json
{ "groups": [
  { "key": "p1-warehouse-04", "part": 1, "level": 450,
    "scene": "A man in a high-visibility vest stacks cardboard boxes onto a metal shelf in a warehouse aisle.",
    "questions": [ { "no": 1, "options": [
        "He's stacking boxes on a shelf.", "He's opening a window.", "He's sweeping the floor.", "He's loading a truck." ],
      "answer": 0, "tags": ["photo-people"],
      "explainVi": "Người đàn ông đang xếp hộp lên kệ. (B) không có cửa sổ, (C) không quét nhà, (D) không có xe tải.",
      "translateVi": "(A) Anh ấy đang xếp hộp lên kệ. (B) … (C) … (D) …",
      "vocab": [ { "word": "stack", "ipa": "/stæk/", "meaningVi": "chồng, xếp chồng" } ] } ] },

  { "key": "p2-shipment-07", "part": 2, "level": 600,
    "script": [ { "speaker": "w", "accent": "gb", "text": "When is the shipment arriving?" } ],
    "questions": [ { "no": 1, "prompt": "When is the shipment arriving?",
      "options": [ "By Thursday at the latest.", "Yes, it was shipped.", "To the main office." ],
      "answer": 0, "tags": ["q-wh"], "explainVi": "Hỏi When → trả lời thời gian. (B) trả lời Yes/No cho câu hỏi WH; (C) trả lời nơi chốn." } ] },

  { "key": "p3-printshop-02", "part": 3, "level": 600,
    "script": [
      { "speaker": "m", "accent": "us", "text": "Hi, I'd like to pick up the brochures I ordered yesterday.", "translateVi": "…" },
      { "speaker": "w", "accent": "gb", "text": "I'm sorry, but our printer broke down this morning, so they're not ready yet." },
      { "speaker": "w2", "accent": "au", "text": "We could deliver them to your office tomorrow at no charge." } ],
    "graphic": null,
    "questions": [
      { "no": 1, "prompt": "Where most likely are the speakers?", "options": ["At a hotel","At a print shop","At a bank","At a library"],
        "answer": 1, "evidence": [ { "doc": -1, "turn": 0, "text": "pick up the brochures I ordered" } ],
        "tags": ["who-where"], "explainVi": "…" },
      { "no": 2, "prompt": "What problem does the woman mention?", "options": ["…","…","…","…"], "answer": 0,
        "evidence": [ { "doc": -1, "turn": 1, "text": "our printer broke down this morning" } ], "tags": ["detail"], "explainVi": "…" },
      { "no": 3, "prompt": "What does the woman offer to do?", "options": ["…","…","…","…"], "answer": 2, "tags": ["detail"], "explainVi": "…" } ] },

  { "key": "p5-lo-03-01", "part": 5, "level": 600,
    "questions": [ { "no": 1,
      "prompt": "The new software will allow employees to submit their expense reports more ___ than before.",
      "options": ["efficient","efficiently","efficiency","efficiencies"], "answer": 1, "tags": ["word-form"],
      "explainVi": "Sau more và bổ nghĩa cho động từ submit → cần trạng từ (-ly). (A) tính từ; (C), (D) danh từ.",
      "translateVi": "Phần mềm mới sẽ giúp nhân viên nộp báo cáo chi phí hiệu quả hơn trước.",
      "vocab": [ { "word": "expense report", "ipa": "/ɪkˈspens rɪˌpɔːrt/", "meaningVi": "báo cáo chi phí" } ] } ] },

  { "key": "p6-memo-renovation", "part": 6, "level": 750,
    "docs": [ { "kind": "memo", "title": "Office Renovation",
      "meta": [ {"k":"To","v":"All staff"}, {"k":"From","v":"Facilities"}, {"k":"Date","v":"May 3"} ],
      "body": [ { "kind": "p", "text": "The third floor will be [[1]] from May 10 to May 24. [[2]] During this period, …" } ] } ],
    "questions": [ { "no": 1, "options": ["closing","closed","close","closes"], "answer": 1, "tags": ["verb-form"], "explainVi": "…" },
                   { "no": 2, "options": ["…","…","…","…"], "answer": 3, "tags": ["sentence-insert"], "explainVi": "…" },
                   { "no": 3, "…": "…" }, { "no": 4, "…": "…" } ] },

  { "key": "p7-chair-web-03", "part": 7, "level": 750,
    "docs": [
      { "kind": "web", "title": "ErgoPlus Office Chair", "meta": [ {"k":"URL","v":"www.brightdesk-office.com/chairs"} ],
        "imageSlot": "doc-1",
        "body": [ { "kind": "p", "text": "Adjustable height and armrests. **$189** · Free delivery on orders over $150." },
                  { "kind": "table", "table": { "columns": ["Colour","In stock","Ships"],
                    "rows": [["Black","Yes","1–2 days"],["Navy","Pre-order","3 weeks"]] } } ] },
      { "kind": "invoice", "title": "Order #50213", "body": [ … ] },
      { "kind": "email", "meta": [ {"k":"From","v":"Daniel Lee"}, {"k":"Subject","v":"My order"} ], "body": [ … ] } ],
    "questions": [ { "no": 1, "prompt": "Why will Mr. Lee's chair arrive late?", "options": ["…","…","…","…"], "answer": 0,
      "evidence": [ { "doc": 0, "block": 1, "text": "Pre-order" }, { "doc": 1, "block": 0, "text": "Navy" } ],
      "tags": ["cross-reference"], "explainVi": "…" } ] }
] }
```

`graphic` (Part 3/4) ví dụ: `{ "kind": "floor-plan", "title": "Riverside Conference Center · Floor 2", "cells": [
{"label":"Room A","row":0,"col":0}, …, {"label":"Hallway","row":1,"col":0,"span":3} ] }`;
`{ "kind": "price-list", "columns": ["Item","Price"], "rows": [["Basic plan","$20"],…] }`;
`{ "kind": "bar-chart", "title": "Monthly visitors", "bars": [ {"label":"Jan","value":120}, … ] }`.

## File đề

```json
{ "key": "luna-test-01", "title": "Luna Test 01", "kind": "full", "order": 1,
  "groups": [ "p1-warehouse-04", "…", "p7-chair-web-03" ] }
```

## Quy tắc soạn (kiểm tra ở chạy thử, xem data-model)

- Mỗi Part đúng số câu và số phương án; Part 3 dùng 2–3 người nói với giọng khác nhau; câu "Look at the graphic" có
  `graphic`.
- Phương án nhiễu hợp lý (bẫy phổ biến: từ nghe giống, từ lặp lại trong bài, sai thì), không mơ hồ; mỗi câu đúng một đáp án.
- `evidence.text` trích nguyên văn; `explainVi` nêu vì sao đúng, vì sao từng phương án sai, bẫy (nếu có).
- Phần Nghe của một đề đầy đủ ước tính 40–50 phút (chạy thử báo `listeningMinutes`).
