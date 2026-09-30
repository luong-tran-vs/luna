# Contract: Export API (F13)

## GET /api/export

Cần đăng nhập (cookie phiên F1). Luôn là dữ liệu của người dùng trong phiên; không nhận tham số.

`200 OK`

```http
Content-Type: application/json; charset=utf-8
Content-Disposition: attachment; filename="luna-export-20260930.json"
Cache-Control: no-store
```

Body: object theo `data-model.md` ("File xuất"): `version` (1), `exportedAt` (RFC 3339, múi giờ người học), `account`, `settings`,
`cards`, `reviewLogs`, `goals`, `lessonProgress`, `studyDays`, `dictationResults`, `lessons`. Mọi mảng luôn có (rỗng là `[]`).

Bảo đảm:

- Không có trường `passwordHash`, không có dữ liệu từ `sessions`, không có tài liệu nào của người dùng khác.
- Ngày trong tên file là ngày theo múi giờ trong Cài đặt (F12) lúc xuất.

Lỗi:

| Trường hợp | Mã | Body |
| --- | --- | --- |
| thiếu phiên | 401 | `{"error":"unauthenticated",…}` |
| tài khoản của phiên không còn | 401 | `{"error":"unauthenticated",…}` |
| lỗi đọc DB | 500 | `{"error":"internal_error",…}` |

## Frontend

- Trang `/settings` → nhóm **Dữ liệu** → nút **Xuất dữ liệu**.
- `ExportApiService.download()` → `GET /api/export` (`responseType: 'blob'`, `observe: 'response'`) → `{blob, filename}`; trang tạo
  link tạm để trình duyệt lưu file.
