# Contract: Dịch vụ ngoài

Cả hai chỉ được gọi qua interface (`tts.Synthesizer`, `ai.Provider`); test dùng bản giả hoặc `httptest.Server`.

## Kokoro-FastAPI (TTS, chạy trên máy)

```http
POST {TTS_URL}/v1/audio/speech
Content-Type: application/json

{"model":"kokoro","input":"He was late!","voice":"af_heart","response_format":"mp3","speed":1.0}
```

- 200: body là mp3 (`audio/mpeg`). Rỗng → lỗi.
- Khác 200 hoặc lỗi mạng/timeout (60 giây) → lỗi có thể thử lại.
- Image: `ghcr.io/remsky/kokoro-fastapi-cpu:<tag ghim>`, cổng 8880. Giấy phép: Apache 2.0 (code và model Kokoro-82M).

## Gemini generateContent (chú thích, gói miễn phí)

```http
POST https://generativelanguage.googleapis.com/v1beta/models/{GEMINI_MODEL}:generateContent
x-goog-api-key: {GEMINI_API_KEY}
Content-Type: application/json

{
  "contents": [{"role": "user", "parts": [{"text": "<prompt: CEFR level + numbered sentences + rules>"}]}],
  "generationConfig": {
    "temperature": 0.2,
    "responseMimeType": "application/json",
    "responseSchema": {
      "type": "ARRAY",
      "items": {
        "type": "OBJECT",
        "properties": {
          "text": {"type": "STRING"},
          "lemma": {"type": "STRING"},
          "meaningVi": {"type": "STRING"},
          "sentenceIndex": {"type": "INTEGER"}
        },
        "required": ["text", "lemma", "meaningVi", "sentenceIndex"]
      }
    }
  }
}
```

- 200: `candidates[0].content.parts[0].text` là chuỗi JSON theo schema → parse thành `[]ai.Annotation`.
- 429 → lỗi "AI hết hạn mức" (thử lại). 400/403 key sai → lỗi "khoá API không hợp lệ" (không thử lại).
  5xx/timeout (90 giây) → thử lại. Không có candidate hoặc JSON sai → lỗi (thử lại).
- Không có key hoặc `AI_PROVIDER=none` → `ai.ErrNotConfigured`, không gọi mạng, không thử lại.
- Log: một dòng `ai request` (model, số câu, thời gian, status) mỗi lần gọi; không log key hay nội dung.
