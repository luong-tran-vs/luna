# LingoStudy: Màu sắc và chữ

> Phiên bản: v1 (2026-09-29). Đã chốt bảng màu **Oải hương**, font **Lexend** và có **chế độ tối**.
> Tài liệu sản phẩm: [mvp-features.md](mvp-features.md). Code giao diện chỉ dùng màu và font qua các token trong file này, không viết mã màu trực tiếp.

## 1. Nguyên tắc

1. **Mọi màu đi qua token (biến CSS).** Chế độ sáng và tối chỉ khác nhau ở giá trị của token, component không cần biết đang ở chế độ nào.
2. **Font phải miễn phí.** Mọi font dùng giấy phép SIL Open Font License và được đóng gói kèm app, không tải từ máy chủ bên ngoài.
3. **Độ tương phản đạt WCAG AA:** chữ thường tối thiểu 4.5:1, chữ lớn và biểu tượng tối thiểu 3:1, ở cả hai chế độ.
4. **Không truyền đạt thông tin chỉ bằng màu.** Đúng, sai, thiếu còn khác nhau về kiểu gạch chữ; bước học còn khác nhau về biểu tượng (✓, số thứ tự).

## 2. Chế độ sáng và tối

- Trong phần cài đặt có 3 lựa chọn: **Sáng**, **Tối**, **Theo hệ thống** (mặc định).
- "Theo hệ thống" đọc `prefers-color-scheme` của thiết bị. Khi người dùng chọn Sáng hoặc Tối, lựa chọn đó ghi đè thiết lập của hệ thống và được lưu vào tài khoản.
- Gắn `data-theme="light"` hoặc `data-theme="dark"` lên thẻ `<html>`. Khi ở chế độ tối thì đặt thêm `color-scheme: dark` để thanh cuộn và ô nhập liệu của trình duyệt cũng đổi theo.

## 3. Bảng màu Oải hương

### 3.1. Màu nền tảng

| Token | Vai trò | Sáng | Tối |
|---|---|---|---|
| `--color-bg` | Nền trang | `#F6F4FB` | `#13101F` |
| `--color-surface` | Nền thẻ, popup | `#FFFFFF` | `#1D1930` |
| `--color-text` | Chữ chính | `#221B38` | `#E9E5F5` |
| `--color-text-muted` | Chữ phụ, nhãn | `#625B7E` | `#A39DC0` |
| `--color-line` | Viền, đường kẻ | `#E0DCEE` | `#302A4A` |
| `--color-track` | Nền của thanh tiến trình | `#E7E3F2` | `#29233F` |
| `--color-primary` | Nút chính, thanh mục tiêu, bước đang làm | `#5B3FB8` | `#A58CF5` |
| `--color-on-primary` | Chữ trên nền màu chính | `#FFFFFF` | `#140B2E` |
| `--color-primary-soft` | Nền nhạt: nhãn, từ đã lưu trong sổ | `#E9E3FA` | `#2E2552` |
| `--color-primary-ink` | Chữ màu chính trên nền nhạt | `#472F96` | `#C9B9FA` |
| `--color-accent` | Điểm nhấn: biểu tượng streak, cụm từ đang chọn | `#D98E04` | `#F6B93B` |
| `--color-accent-soft` | Nền nhạt của điểm nhấn | `#FCEFD2` | `#3A2C0F` |
| `--color-accent-ink` | Chữ màu điểm nhấn trên nền nhạt | `#855600` | `#F8CD72` |

**Lưu ý:** `--color-accent` (vàng) **không dùng làm màu chữ** trên nền sáng vì không đủ tương phản. Chữ màu vàng luôn dùng `--color-accent-ink`, ví dụ số ngày trong chip streak.

### 3.2. Màu 4 kỹ năng

Dùng cho 4 thanh kỹ năng dưới thanh mục tiêu, và cho nhãn kỹ năng ở các màn hình khác. Các màu được chọn để không trùng với màu chính (tím) và màu điểm nhấn (vàng).

| Token | Kỹ năng | Sáng | Tối |
|---|---|---|---|
| `--color-skill-listen` | Nghe | `#2F7FD8` | `#6AAAF0` |
| `--color-skill-speak` | Nói | `#D14D7E` | `#F07BA6` |
| `--color-skill-read` | Đọc | `#2A9D63` | `#5CC58F` |
| `--color-skill-write` | Viết | `#A0622A` | `#D39A5E` |

### 3.3. Màu trạng thái

| Token | Dùng cho | Sáng | Tối | Dấu hiệu đi kèm |
|---|---|---|---|---|
| `--color-ok` | Đúng | `#23874F` | `#5CC58F` | Chữ bình thường |
| `--color-bad` | Sai, lỗi | `#C93636` | `#F07575` | Gạch ngang chữ sai, hiện chữ đúng ngay bên cạnh |
| `--color-warn` | Thiếu, cảnh báo | `#B97A00` | `#F0B54A` | Gạch chân chấm |

## 4. Font chữ

| Vai trò | Font | Gói npm | Ghi chú |
|---|---|---|---|
| Giao diện và bài đọc | **Lexend** | `@fontsource-variable/lexend` | Một font cho mọi thứ; có bộ ký tự tiếng Việt |
| Phiên âm IPA | **Noto Sans** | `@fontsource/noto-sans` | Lexend thiếu một số ký tự IPA như ʊ, ə, ʃ, ː. Chỉ dùng cho phần phiên âm |

```css
--font-ui:  "Lexend Variable", "Lexend", system-ui, "Segoe UI", sans-serif;
--font-ipa: "Noto Sans", "Segoe UI", sans-serif;
```

Chỉ nạp các bộ ký tự `latin`, `latin-ext`, `vietnamese` (và phần IPA của Noto Sans) để file font nhẹ.

### 4.1. Cỡ chữ

| Token | Cỡ / dòng | Độ đậm | Dùng cho |
|---|---|---|---|
| `--text-xs` | 12px / 16px | 500 | Nhãn viết hoa, chú thích |
| `--text-sm` | 14px / 20px | 400 | Chữ phụ, nút nhỏ |
| `--text-base` | 16px / 24px | 400 | Chữ giao diện mặc định |
| `--text-reading` | 18px / 30px | 400 | **Bài đọc tiếng Anh** và chép chính tả |
| `--text-lg` | 20px / 28px | 600 | Tiêu đề thẻ, tên bài học |
| `--text-xl` | 24px / 32px | 700 | Số lớn (12/30 bài), tiêu đề màn hình |
| `--text-2xl` | 30px / 38px | 700 | Tiêu đề trang (dùng ít) |

- Độ đậm dùng: 400 (thường), 500 (nhãn), 600 (tiêu đề nhỏ, nút), 700 (tiêu đề, số lớn).
- Bài đọc giới hạn khoảng 65 ký tự mỗi dòng trên màn hình rộng.
- Các con số trong tiến độ và thống kê dùng `font-variant-numeric: tabular-nums` để thẳng cột.

## 5. Hình khối

| Token | Giá trị | Dùng cho |
|---|---|---|
| `--radius-sm` | 6px | Nhãn (tag) |
| `--radius-md` | 12px | Nút, ô nhập |
| `--radius-lg` | 16px | Thẻ, popup |
| `--radius-full` | 999px | Thanh tiến trình, chip streak |
| `--space-*` | 4, 8, 12, 16, 24, 32px | Khoảng cách, theo bội số của 4 |

## 6. Cách dùng trong Angular

- Khai báo toàn bộ token trong `src/styles/tokens.css`: bộ màu sáng đặt trong `:root`, bộ màu tối đặt trong `[data-theme="dark"]` và trong `@media (prefers-color-scheme: dark)` khi chưa chọn chế độ.
- Component chỉ dùng `var(--color-...)`, `var(--text-...)`, không viết mã màu hay cỡ chữ trực tiếp.
- Một `ThemeService` đọc lựa chọn của người dùng (Sáng, Tối, Theo hệ thống) và gắn `data-theme` lên `<html>`.
