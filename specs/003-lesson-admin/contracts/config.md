# Contract: Biến môi trường mới (F2)

| Biến | Bắt buộc | Mặc định | Ý nghĩa |
|---|---|---|---|
| `TTS_URL` | Không | `http://localhost:8880` (compose: `http://kokoro:8880`) | Địa chỉ Kokoro-FastAPI |
| `TTS_VOICE` | Không | `af_heart` | Giọng đọc |
| `AUDIO_DIR` | Không | `./data/audio` (compose: `/data/audio`) | Thư mục lưu mp3 |
| `AI_PROVIDER` | Không | `gemini` | `gemini` \| `none` |
| `GEMINI_API_KEY` | Không | rỗng | Rỗng → chú thích báo "AI chưa được cấu hình" |
| `GEMINI_MODEL` | Không | `gemini-3.5-flash-lite` | Model dùng cho chú thích |

`AI_PROVIDER` khác `gemini|none` → lỗi `config: AI_PROVIDER must be gemini or none`, backend dừng mã 1.
`TTS_URL` không phải URL http(s) hợp lệ → lỗi tương tự.

`GEMINI_API_KEY` là bí mật: chỉ đặt trong `deploy/.env` hoặc `backend/.env` (đều bị git bỏ qua), không bao giờ log.
Cập nhật `backend/.env.example`, `deploy/.env.example`, `deploy/docker-compose.yml` và bảng cấu hình trong README.
