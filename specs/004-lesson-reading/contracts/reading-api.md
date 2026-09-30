# Contract: Reading API

Tất cả cần đăng nhập (`RequireAuth`, mọi vai trò). Lỗi theo dạng chung `{error, message}`.

## GET /api/lessons/{id}

200:

```json
{ "lesson": {
  "id": "…", "title": "Park", "level": "B1", "topic": "Daily",
  "sentences": [ { "index": 0, "text": "We went to the park.", "audioUrl": "/api/audio/…/1/0" } ],
  "paragraphs": [[0, 1], [2]],
  "lemmas": { "went": "go", "studies": "study", "park": "park" },
  "phrases": [ { "text": "gave up", "lemma": "give up" } ]
} }
```

Không có `annotations`, `revision`, trạng thái hay lỗi việc nền. 404 `not_found`; 401.

## GET /api/lessons/{id}/lookup?q={text}&sentence={N}

- `q`: 1–100 ký tự sau khi trim, tối đa 6 từ. `sentence`: số nguyên ≥ 0 (tuỳ chọn; ngoài phạm vi thì bỏ qua).
- 200:

  ```json
  { "source": "ai", "text": "went", "lemma": "go", "ipa": "/ˈɡəʊ/",
    "meanings": [ { "pos": "", "text": "đã đi" } ] }
  ```

  `source: "dictionary"` → tối đa 3 nghĩa, `pos` là mã từ điển (`N`, `V`, `A`, …).
- 404 `not_found` ("Chưa có nghĩa") khi không có ở chú thích và từ điển; 404 `lesson_not_found` khi bài không tồn tại.
- 400 `validation_failed` khi `q` rỗng, quá dài, hơn 6 từ, hoặc `sentence` không phải số.
- Không gọi AI trong mọi trường hợp.

## GET /api/tts/word?text={text}

- `text` sau chuẩn hoá: 1–60 ký tự, chỉ chữ cái, khoảng trắng, `'`, `-`; khác → 400.
- 200 `audio/mpeg`, `Cache-Control: private, max-age=31536000, immutable`.
- 503 `tts_unavailable` ("Chưa phát được âm thanh") khi Kokoro lỗi. 401 khi chưa đăng nhập.

## Test hợp đồng

- lesson: 200 không lộ `annotations`; 404; 401.
- lookup: chú thích trước từ điển; ưu tiên câu `N`; dạng biến đổi (*studies* → *study*); cụm theo chú thích; 404 khi không
  có; 400 các lỗi đầu vào; dictionary giả ghi nhận không có lời gọi AI (Provider không được truyền vào).
- tts word: file có sẵn không gọi Synthesizer; 2 yêu cầu song song cùng từ chỉ gọi 1 lần; 400 ký tự lạ; 503 khi lỗi.
