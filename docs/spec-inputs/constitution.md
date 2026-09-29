# Đầu vào cho /speckit.constitution

Dán toàn bộ khối bên dưới vào sau lệnh `/speckit.constitution`.

---

```text
Tạo constitution cho dự án Luna: web app học tiếng Anh cá nhân, luyện 4 kỹ năng Nghe, Nói, Đọc, Viết quanh một bài học mỗi ngày. Viết bằng tiếng Việt. Phiên bản 1.0.0, phê chuẩn ngày 2026-09-29.

NGUYÊN TẮC CỐT LÕI

I. Tài liệu trong docs/ là nguồn sự thật
- docs/mvp-features.md, docs/phases/*.md, docs/design-system.md và docs/architecture.md quyết định sản phẩm làm gì và cấu trúc mã nguồn.
- Spec, plan, task mâu thuẫn với docs/ thì sửa spec, plan, task; muốn đổi sản phẩm thì sửa docs/ trước.
- Mỗi spec ghi rõ mã tính năng (F1, F2..., L) và bao phủ đủ tiêu chí nghiệm thu của tính năng đó.
- Chỉ làm tính năng của giai đoạn đang thực hiện.

II. Chi phí 0 đồng (KHÔNG THƯƠNG LƯỢNG)
- Không phụ thuộc dịch vụ trả phí. AI dùng gói miễn phí hoặc mô hình chạy trên máy.
- Thư viện, font, dữ liệu phải có giấy phép miễn phí.

III. AI là phần bổ sung
- AI lỗi, chậm hoặc hết hạn mức thì mọi tính năng không cần AI vẫn chạy bình thường.
- Mọi lời gọi AI đi qua một interface; đổi nhà cung cấp chỉ bằng cấu hình.
- Tác vụ AI chạy nền, có trạng thái (đang chạy, xong, lỗi), chạy lại được, và kết quả được lưu để không gọi lặp.

IV. Mobile-first và dễ tiếp cận
- Dùng tốt từ chiều rộng 360px, không cuộn ngang.
- Có chế độ sáng và tối. Màu, font, cỡ chữ chỉ dùng token trong docs/design-system.md.
- Độ tương phản đạt WCAG AA; không truyền đạt thông tin chỉ bằng màu.

V. Bảo vệ dữ liệu người học
- Người dùng chỉ thấy dữ liệu của mình; quyền được kiểm tra ở API, không chỉ ở giao diện.
- Truy cập cơ sở dữ liệu qua interface repository để đổi hệ cơ sở dữ liệu mà không sửa logic nghiệp vụ.
- Mật khẩu lưu dạng băm; bí mật chỉ nằm trong biến môi trường, không commit vào repo.

VI. Kiểm thử logic nghiệp vụ
- Logic nghiệp vụ có unit test (ví dụ: lịch ôn FSRS, quy tắc một ngày học, so sánh chép chính tả).
- Mỗi endpoint có test cho trường hợp thành công, lỗi đầu vào và sai quyền.
- Test không cần mạng: AI, TTS, STT, từ điển được giả lập.
- Mỗi tiêu chí nghiệm thu có ít nhất một test hoặc một bước kiểm tra thủ công ghi trong tasks.

VII. Đơn giản trước
- Một người phát triển, một người dùng: chọn giải pháp đơn giản nhất đáp ứng tiêu chí nghiệm thu.
- Không thêm thư viện hay lớp trừu tượng khi chưa cần, trừ interface ở nguyên tắc III và V.

RÀNG BUỘC CÔNG NGHỆ
- Frontend Angular (standalone, signals, TypeScript strict). Backend Go, REST JSON. Cơ sở dữ liệu MongoDB (tạm thời).
- Chạy bằng Docker Compose trên máy không có GPU.
- Code Go tuân theo bộ skill samber/cc-skills-golang trong .claude/skills.
- Tên trong code, API và commit message bằng tiếng Anh; tài liệu, spec và chữ trên giao diện bằng tiếng Việt.

QUY TRÌNH PHÁT TRIỂN
- Mỗi tính năng trên một nhánh riêng: specify → clarify (nếu cần) → plan → tasks → implement.
- Thứ tự làm theo docs/phases/.
- Xong khi: đạt mọi tiêu chí nghiệm thu, test qua, lint không lỗi, đã kiểm tra ở 360px cả chế độ sáng và tối.

QUẢN TRỊ
- Constitution cao hơn mọi quy ước khác trong dự án.
- Sửa constitution phải tăng phiên bản theo semver (MAJOR: bỏ hoặc đổi nghĩa nguyên tắc; MINOR: thêm nguyên tắc; PATCH: sửa câu chữ).
- Plan phải có mục kiểm tra tuân thủ constitution; mọi ngoại lệ phải ghi lý do.
```
