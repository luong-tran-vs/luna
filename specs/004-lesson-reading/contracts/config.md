# Contract: Cấu hình và dữ liệu từ điển (F3)

| Biến | Bắt buộc | Mặc định | Ý nghĩa |
|---|---|---|---|
| `DICTIONARY_PATH` | Không | `./data/dictionary/dictionary.db` (compose: `/data/dictionary/dictionary.db`) | File SQLite từ điển |

File không tồn tại → backend log `dictionary unavailable` (warn) và tra cứu chỉ dùng chú thích; không dừng.

## Tải từ điển

```bash
./deploy/fetch-dictionary.sh          # bash (Linux/macOS/WSL/Git Bash)
.\deploy\fetch-dictionary.ps1         # PowerShell
```

Tải `https://github.com/minhqnd/dictionary/releases/download/v2.0.0/dictionary.db` về
`deploy/data/dictionary/dictionary.db`, kiểm tra SHA-256
`9259403f0675b2991a1bd0ef6d0dbc5933afdb135632af095a60662f09bbf1d3`; sai thì xoá file và báo lỗi. Có sẵn file đúng thì
không tải lại.

- `deploy/docker-compose.yml`: backend thêm `DICTIONARY_PATH: /data/dictionary/dictionary.db` và volume
  `./data/dictionary:/data/dictionary:ro`.
- `backend/.env.example`: `DICTIONARY_PATH=../deploy/data/dictionary/dictionary.db`.
- `.gitignore`: `deploy/data/`.
