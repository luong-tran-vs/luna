# Data Model: Tài khoản (F1)

## 1. users (MongoDB)

| Trường | Kiểu | Ràng buộc |
|---|---|---|
| `_id` | ObjectID | tự sinh |
| `email` | string | chữ thường, đã trim, ≤ 254, **unique index** |
| `passwordHash` | string | PHC argon2id (research R1), không bao giờ trả ra API |
| `role` | string | `admin` \| `learner` |
| `timezone` | string | tên IANA hợp lệ, mặc định `Asia/Ho_Chi_Minh` |
| `createdAt` | Date | UTC |

Go: `auth.User{ID string; Email string; PasswordHash string; Role Role; Timezone string; CreatedAt time.Time}`.
ID trong domain là chuỗi hex; `storage/mongo` chuyển đổi ObjectID ↔ string.

Quy tắc: bản ghi đầu tiên có `role = admin`; không có API sửa hay xoá ở F1.

## 2. sessions (MongoDB)

| Trường | Kiểu | Ràng buộc |
|---|---|---|
| `_id` | ObjectID | tự sinh |
| `tokenHash` | string | SHA-256 hex của token, **unique index** |
| `userId` | ObjectID | tham chiếu `users._id` |
| `expiresAt` | Date | `createdAt + 30 ngày`, **TTL index** `expireAfterSeconds: 0` |
| `createdAt` | Date | UTC |

Vòng đời: tạo khi đăng ký/đăng nhập → hợp lệ khi `expiresAt > now` → kết thúc khi đăng xuất (xoá) hoặc
hết hạn (TTL xoá, `Authenticate` từ chối ngay cả trước khi TTL chạy).

## 3. Login attempts (bộ nhớ backend, không lưu DB)

| Trường | Kiểu | Ý nghĩa |
|---|---|---|
| key | string | email đã chuẩn hoá (kể cả email chưa đăng ký) |
| `failures` | int | số lần sai liên tiếp chưa bị khoá |
| `lockedUntil` | time.Time | zero nếu không khoá |

Chuyển trạng thái: bình thường —sai (<5)→ đếm tăng —sai lần 5→ khoá 15 phút (đếm về 0) —hết giờ→ bình
thường. Đăng nhập đúng khi chưa khoá → xoá mục.

## 4. Repository (interface trong `internal/auth/repository.go`)

```go
type UserRepository interface {
    Create(ctx context.Context, u User) (User, error)          // ErrEmailTaken khi trùng
    FindByEmail(ctx context.Context, email string) (User, error) // ErrNotFound
    FindByID(ctx context.Context, id string) (User, error)       // ErrNotFound
    Count(ctx context.Context) (int64, error)
}

type SessionRepository interface {
    Create(ctx context.Context, s Session) error
    FindByTokenHash(ctx context.Context, hash string) (Session, error) // ErrNotFound
    Delete(ctx context.Context, hash string) error                     // không lỗi nếu không có
}
```

## 5. Kiểu trả ra API / frontend

```ts
// core/models/user.ts
export type Role = 'admin' | 'learner';
export interface User { id: string; email: string; role: Role; timezone: string; }
```

Cùng hình dạng với JSON `user` ở `contracts/auth-api.md`. Không có `passwordHash`.
