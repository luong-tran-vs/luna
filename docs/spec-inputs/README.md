# Thứ tự chạy spec-kit

## Thứ tự

**F0 → F1 → F2 → F3 → F4 → F5 → F14 → L → F6 → F12 → F13**

| Thứ tự | Tính năng | specify | plan |
|---|---|---|---|
| 1 | F0 | `/speckit-specify Tạo spec theo khối 1 trong @docs/spec-inputs/f0-khung-du-an.md` | `/speckit-plan Lập plan theo khối 2 trong @docs/spec-inputs/f0-khung-du-an.md` |
| 2 | F1 | `/speckit-specify Tạo spec theo khối 1 trong @docs/spec-inputs/f1-tai-khoan.md` | `/speckit-plan Lập plan theo khối 2 trong @docs/spec-inputs/f1-tai-khoan.md` |
| 3 | F2 | `/speckit-specify Tạo spec theo khối 1 trong @docs/spec-inputs/f2-quan-tri-bai-hoc.md` | `/speckit-plan Lập plan theo khối 2 trong @docs/spec-inputs/f2-quan-tri-bai-hoc.md` |
| 4 | F3 | `/speckit-specify Tạo spec theo khối 1 trong @docs/spec-inputs/f3-doc.md` | `/speckit-plan Lập plan theo khối 2 trong @docs/spec-inputs/f3-doc.md` |
| 5 | F4 | `/speckit-specify Tạo spec theo khối 1 trong @docs/spec-inputs/f4-nghe.md` | `/speckit-plan Lập plan theo khối 2 trong @docs/spec-inputs/f4-nghe.md` |
| 6 | F5 | `/speckit-specify Tạo spec theo khối 1 trong @docs/spec-inputs/f5-so-tu-on-tap.md` | `/speckit-plan Lập plan theo khối 2 trong @docs/spec-inputs/f5-so-tu-on-tap.md` |
| 7 | F14 | `/speckit-specify Tạo spec theo khối 1 trong @docs/spec-inputs/f14-chu-de.md` | `/speckit-plan Lập plan theo khối 2 trong @docs/spec-inputs/f14-chu-de.md` |
| 8 | L | `/speckit-specify Tạo spec theo khối 1 trong @docs/spec-inputs/l-luong-mot-ngay-hoc.md` | `/speckit-plan Lập plan theo khối 2 trong @docs/spec-inputs/l-luong-mot-ngay-hoc.md` |
| 9 | F6 | `/speckit-specify Tạo spec theo khối 1 trong @docs/spec-inputs/f6-man-hinh-chinh.md` | `/speckit-plan Lập plan theo khối 2 trong @docs/spec-inputs/f6-man-hinh-chinh.md` |
| 10 | F12 | `/speckit-specify Tạo spec theo khối 1 trong @docs/spec-inputs/f12-cai-dat.md` | `/speckit-plan Lập plan theo khối 2 trong @docs/spec-inputs/f12-cai-dat.md` |
| 11 | F13 | `/speckit-specify Tạo spec theo khối 1 trong @docs/spec-inputs/f13-sao-luu.md` | `/speckit-plan Lập plan theo khối 2 trong @docs/spec-inputs/f13-sao-luu.md` |

Mỗi tính năng: **specify → (clarify nếu cần) → plan → tasks → implement → kiểm tra thủ công**, xong mới sang tính năng tiếp theo.

Mỗi file có 2 khối: **Khối 1** cho `/speckit-specify` (làm gì, vì sao), **Khối 2** cho `/speckit-plan` (làm thế nào). Tiêu chí nghiệm thu nằm ở [docs/phases/](../phases/).

## Giai đoạn 1

Cập nhật: 2026-09-30 (lần 2).

| # | Tính năng | File đầu vào | Spec | Trạng thái |
|---|---|---|---|---|
| 0 | Constitution | [constitution.md](constitution.md) | `.specify/memory/constitution.md` | ✅ Xong |
| 1 | F0 Khung dự án | [f0-khung-du-an.md](f0-khung-du-an.md) | `specs/001-project-skeleton` | 🟡 Code xong, còn kiểm tra thủ công |
| 2 | F1 Tài khoản | [f1-tai-khoan.md](f1-tai-khoan.md) | `specs/002-user-accounts` | 🟡 Code xong, còn kiểm tra thủ công |
| 3 | F2 Quản trị bài học | [f2-quan-tri-bai-hoc.md](f2-quan-tri-bai-hoc.md) | `specs/003-lesson-admin` | 🟡 Code xong, còn kiểm tra thủ công |
| 4 | F3 Đọc | [f3-doc.md](f3-doc.md) | `specs/004-lesson-reading` | 🟡 Code xong, còn kiểm tra thủ công |
| 5 | F4 Nghe | [f4-nghe.md](f4-nghe.md) | `specs/005-lesson-listening` | 🟡 Code xong, còn kiểm tra thủ công |
| 6 | F5 Sổ từ và ôn tập | [f5-so-tu-on-tap.md](f5-so-tu-on-tap.md) | `specs/006-vocabulary-review` | 🟡 Code xong, còn kiểm tra thủ công |
| 7 | F14 Chủ đề và lộ trình | [f14-chu-de.md](f14-chu-de.md) | `specs/007-topic-roadmaps` | 🟡 Code xong, còn kiểm tra thủ công |
| 8 | L Luồng một ngày học | [l-luong-mot-ngay-hoc.md](l-luong-mot-ngay-hoc.md) | `specs/008-daily-study-flow` | 🟡 Code xong, còn kiểm tra thủ công |
| 9 | F6 Màn hình chính | [f6-man-hinh-chinh.md](f6-man-hinh-chinh.md) | `specs/009-home-dashboard` | 🟡 Code xong, còn kiểm tra thủ công |
| 10 | F12 Cài đặt | [f12-cai-dat.md](f12-cai-dat.md) | `specs/010-user-settings` | 🟡 Code xong, còn kiểm tra thủ công |
| 11 | F13 Sao lưu | [f13-sao-luu.md](f13-sao-luu.md) | `specs/011-backup-export` | 🟡 Code xong, còn kiểm tra thủ công |

**Nên làm trước L:** kiểm tra thủ công F0–F5 theo `quickstart.md` của từng spec (360px, sáng/tối), rồi tích tiêu chí trong `docs/phases/giai-doan-1.md`.

## Lệnh cho mỗi tính năng

Thay `<file>` bằng tên file đầu vào ở bảng trên.

```
1. /speckit-specify   Tạo spec theo khối 1 trong @docs/spec-inputs/<file>.md
2. /speckit-clarify   (chỉ khi spec còn chỗ [NEEDS CLARIFICATION])
3. /speckit-plan      Lập plan theo khối 2 trong @docs/spec-inputs/<file>.md
4. /speckit-tasks
5. /speckit-analyze   (không bắt buộc)
6. /speckit-implement
7. Kiểm tra thủ công theo quickstart.md, tích tiêu chí trong docs/phases/giai-doan-1.md
```

Xong bước 7 của tính năng này rồi mới sang tính năng tiếp theo.

## Giai đoạn 2

**F7 → F15 → F8 → F9**

| Thứ tự | Tính năng | specify | plan |
|---|---|---|---|
| 1 | F7 AI sinh bài | `/speckit-specify Tạo spec theo khối 1 trong @docs/spec-inputs/f7-ai-sinh-bai.md` | `/speckit-plan Lập plan theo khối 2 trong @docs/spec-inputs/f7-ai-sinh-bai.md` |
| 2 | F15 Hiểu bài và ngữ pháp | `/speckit-specify Tạo spec theo khối 1 trong @docs/spec-inputs/f15-hieu-bai-ngu-phap.md` | `/speckit-plan Lập plan theo khối 2 trong @docs/spec-inputs/f15-hieu-bai-ngu-phap.md` |
| 3 | F8 Viết | `/speckit-specify Tạo spec theo khối 1 trong @docs/spec-inputs/f8-viet.md` | `/speckit-plan Lập plan theo khối 2 trong @docs/spec-inputs/f8-viet.md` |
| 4 | F9 Hỏi AI về từ | `/speckit-specify Tạo spec theo khối 1 trong @docs/spec-inputs/f9-hoi-ai.md` | `/speckit-plan Lập plan theo khối 2 trong @docs/spec-inputs/f9-hoi-ai.md` |

F15 phải xong trước F8, vì đề viết được sinh ở F15.

## Giai đoạn 3

**F16 → F10 → F11**

| Thứ tự | Tính năng | specify | plan |
|---|---|---|---|
| 1 | F16 Vận hành | `/speckit-specify Tạo spec theo khối 1 trong @docs/spec-inputs/f16-van-hanh.md` | `/speckit-plan Lập plan theo khối 2 trong @docs/spec-inputs/f16-van-hanh.md` |
| 2 | F10 Nói | `/speckit-specify Tạo spec theo khối 1 trong @docs/spec-inputs/f10-noi.md` | `/speckit-plan Lập plan theo khối 2 trong @docs/spec-inputs/f10-noi.md` |
| 3 | F11 Hội thoại nhập vai | `/speckit-specify Tạo spec theo khối 1 trong @docs/spec-inputs/f11-hoi-thoai.md` | `/speckit-plan Lập plan theo khối 2 trong @docs/spec-inputs/f11-hoi-thoai.md` |
| 4 | F17 Chi tiết bài và luyện tập từ vựng | `/speckit-specify Tạo spec theo khối 1 trong @docs/spec-inputs/f17-luyen-tap-tu-vung.md` | `/speckit-plan Lập plan theo khối 2 trong @docs/spec-inputs/f17-luyen-tap-tu-vung.md` |
| 5 | F18 Từ vựng theo chủ đề | `/speckit-specify Tạo spec theo khối 1 trong @docs/spec-inputs/f18-tu-vung-chu-de.md` | `/speckit-plan Lập plan theo khối 2 trong @docs/spec-inputs/f18-tu-vung-chu-de.md` |

F16 phải xong trước F10: trình duyệt điện thoại chỉ cho dùng micro trên HTTPS.
