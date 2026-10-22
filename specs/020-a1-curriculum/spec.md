# Feature Specification: Giáo trình A1

**Feature Branch**: `020-a1-curriculum`

**Created**: 2026-10-22

**Status**: Implemented

**Input**: mục 3.2 của bản đánh giá hệ thống, bản phác thảo `design/de-xuat-giao-trinh-a1.png` đã được duyệt
("chọn phương án 2": mỗi chủ đề 3 bài chính + 2–3 bài thêm). Đã chốt: có danh sách từ theo cấp CEFR để cảnh báo từ vượt
trình độ; điểm ngữ pháp chưa tới vẫn mở xem được ("Học trước").

## Tóm tắt

A1 có một giáo trình cố định: 12 chủ đề theo thứ tự ngữ pháp, mỗi chủ đề 3 bài chính và 2–3 bài thêm (36 + 29 = 65 bài),
đủ 18/18 điểm ngữ pháp A1. Giáo trình nằm trong mã nguồn (`backend/internal/curriculum/a1.json`), như phần Ngữ pháp.

## Yêu cầu

- **FR-001** Giáo trình: mỗi chủ đề có tên như danh mục chủ đề, câu "Sau chủ đề này bạn có thể…", điểm ngữ pháp, và các ô
  bài (khoá, loại chính/thêm, tên bài, điểm ngữ pháp, ý tưởng tiếng Anh cho AI). 3 ô đầu là bài chính. Kiểm tra khi khởi
  động: điểm ngữ pháp đúng trình độ, khoá và tên bài không trùng.
- **FR-002** Một ô có bài khi lộ trình của chủ đề (trình độ A1) có bài cùng tên (không phân biệt hoa thường, khoảng trắng,
  kiểu dấu nháy). Không thêm cột/trường mới cho bài học.
- **FR-003** Trang quản trị "Giáo trình" (`GET /api/admin/curriculum?level=A1`): từng chủ đề, từng ô (trống, nháp, đã
  đăng), bài khác trong chủ đề, điểm ngữ pháp nào chưa đăng bài ngữ pháp.
- **FR-004** "Áp dụng thứ tự giáo trình" (`POST /api/admin/curriculum/apply`): tạo chủ đề còn thiếu (mô tả = câu "Sau chủ
  đề này…"), đặt thứ tự chủ đề A1 theo giáo trình (chủ đề ngoài giáo trình xếp sau), sắp lộ trình mỗi chủ đề theo ô: bài
  chính, bài thêm, rồi bài khác.
- **FR-005** "Sinh bài này" (`POST /api/admin/curriculum/slots/{key}/generate`): 1 request AI; tên bài = tên ô, điểm
  ngữ pháp = điểm đầu của ô (các điểm sau đưa vào ý tưởng), 8 từ mục tiêu theo kế hoạch từ của chủ đề, 120 từ; lưu thành
  bài nháp ở đúng chỗ trong lộ trình. "Sinh các ô trống" của một chủ đề gọi lần lượt từng ô.
- **FR-005b** "Gỡ" bên mỗi bài (`POST /api/admin/curriculum/lessons/{id}/remove`): gỡ bài khỏi lộ trình để ô trống lại;
  bài nháp bị xoá luôn, bài đã đăng vẫn còn trong danh sách Bài học (như "Gỡ" ở trang Lộ trình). Có hộp xác nhận.
- **FR-006** Kiểm tra bài nháp (`GET /api/admin/curriculum/checks/{lessonId}`, cũng có sẵn trong bảng cho bài nháp): số
  lần dùng điểm ngữ pháp (≥ 3, đếm bằng mẫu cho 18 điểm A1), có đoạn he/she, câu phủ định, câu hỏi, đủ từ mục tiêu, độ dài
  (A1 90–150 từ), hội thoại (≥ 3 lượt hỏi, không chép nguyên câu bài đọc; khi đã có luyện tập), từ vượt trình độ.
- **FR-007** Từ vượt trình độ dùng CEFR-J Wordlist 1.5 (A1–B2, Tono Laboratory, TUFS; dùng miễn phí có ghi nguồn), gói
  `backend/internal/cefr`: tra theo dạng gốc, bỏ tên riêng (chữ hoa giữa câu), từ không có trong danh sách.
- **FR-007b** Sinh bài chỉ dùng từ đúng trình độ (người dùng chọn "cách 3", áp dụng cho cả trang Giáo trình và trang Lộ
  trình): lời nhắc sinh bài yêu cầu từ thông dụng của trình độ; sinh xong, bài nào còn từ vượt trình độ (CEFR-J, trừ từ
  mục tiêu và tên riêng) được gửi lại AI **1 request cho cả lô** (op `lesson_simplify`) để chỉ thay những từ đó. Bản sửa
  chỉ được nhận khi ít từ khó hơn, độ dài trong ±20 % và không mất từ mục tiêu; lỗi AI thì giữ bản gốc. Từ còn lại trả về ở
  `hardWords` của bản nháp và hiện cảnh báo.
- **FR-008** Lộ trình người học (`GET /api/lessons/path`): chủ đề trong giáo trình có `course {number, canDoVi,
  grammarVi}`, bài có `extra`. Học xong 3 bài chính thì có lối sang chủ đề sau; bài thêm vẫn học theo thứ tự nếu không chuyển.
- **FR-009** Ngữ pháp (`GET /api/grammar?level=A1`): điểm xếp theo thứ tự giáo trình, mỗi điểm có `courseNumber`,
  `courseTopic`; "Học tiếp" ưu tiên điểm của chủ đề đang học; điểm của chủ đề sau vẫn mở xem được (nhãn "Học trước").

## Ngoài phạm vi

- Giáo trình A2 trở lên.
- Gợi ý từ dễ hơn cho từ vượt trình độ (cần AI); hiện chỉ liệt kê từ và cấp.
- Sinh nhiều ô trong 1 request AI (mỗi ô 1 request vì mỗi ô có điểm ngữ pháp riêng).
- Admin sửa tên, ý tưởng của ô (sửa trong `a1.json`).
