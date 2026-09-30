<!--
Sync Impact Report
- Thay đổi phiên bản: (template chưa điền) → 1.0.0
- Nguyên tắc: thay 5 placeholder của template bằng 7 nguyên tắc mới
  I. Tài liệu trong docs/ là nguồn sự thật
  II. Chi phí 0 đồng (KHÔNG THƯƠNG LƯỢNG)
  III. AI là phần bổ sung
  IV. Mobile-first và dễ tiếp cận
  V. Bảo vệ dữ liệu người học
  VI. Kiểm thử logic nghiệp vụ
  VII. Đơn giản trước
- Mục thêm: Ràng buộc công nghệ, Quy trình phát triển, Quản trị
- Mục bỏ: không có
- Template:
  ✅ .specify/templates/plan-template.md — Constitution Check liệt kê cổng I–VII
  ✅ .specify/templates/spec-template.md — thêm dòng Mã tính năng và Giai đoạn
  ✅ .specify/templates/tasks-template.md — test không còn là tùy chọn (nguyên tắc VI),
     thêm bước kiểm tra 360px sáng/tối và lint
  ✅ .specify/templates/checklist-template.md — không cần đổi
  ✅ .claude/skills/speckit-* — không có tham chiếu lỗi thời cần sửa
- TODO hoãn lại: không có
-->

# Luna Constitution

Luna là web app học tiếng Anh cá nhân, luyện 4 kỹ năng Nghe, Nói, Đọc, Viết quanh một bài
học mỗi ngày.

## Core Principles

### I. Tài liệu trong docs/ là nguồn sự thật

- `docs/mvp-features.md`, `docs/phases/*.md`, `docs/design-system.md` và `docs/architecture.md`
  quyết định sản phẩm làm gì và cấu trúc mã nguồn.
- Spec, plan, task mâu thuẫn với `docs/` thì PHẢI sửa spec, plan, task; muốn đổi sản phẩm thì
  PHẢI sửa `docs/` trước.
- Mỗi spec PHẢI ghi rõ mã tính năng (F1, F2..., L) và bao phủ đủ tiêu chí nghiệm thu của tính
  năng đó.
- Chỉ làm tính năng của giai đoạn đang thực hiện.

Lý do: một nguồn sự thật duy nhất giúp spec, plan và code không trôi khỏi ý định sản phẩm.

### II. Chi phí 0 đồng (KHÔNG THƯƠNG LƯỢNG)

- KHÔNG phụ thuộc dịch vụ trả phí. AI dùng gói miễn phí hoặc mô hình chạy trên máy.
- Thư viện, font, dữ liệu PHẢI có giấy phép miễn phí.

Lý do: đây là dự án cá nhân; mọi chi phí định kỳ đều là rủi ro khiến dự án dừng lại.

### III. AI là phần bổ sung

- AI lỗi, chậm hoặc hết hạn mức thì mọi tính năng không cần AI vẫn PHẢI chạy bình thường.
- Mọi lời gọi AI PHẢI đi qua một interface; đổi nhà cung cấp chỉ bằng cấu hình.
- Tác vụ AI PHẢI chạy nền, có trạng thái (đang chạy, xong, lỗi), chạy lại được, và kết quả
  được lưu để không gọi lặp.

Lý do: gói AI miễn phí không ổn định và hạn mức thấp; việc học hằng ngày không được phụ thuộc
vào chúng.

### IV. Mobile-first và dễ tiếp cận

- Giao diện PHẢI dùng tốt từ chiều rộng 360px, không cuộn ngang.
- PHẢI có chế độ sáng và tối. Màu, font, cỡ chữ chỉ dùng token trong `docs/design-system.md`.
- Độ tương phản PHẢI đạt WCAG AA; KHÔNG truyền đạt thông tin chỉ bằng màu.

Lý do: người học dùng chủ yếu trên điện thoại, nhiều lúc trong điều kiện ánh sáng khác nhau.

### V. Bảo vệ dữ liệu người học

- Người dùng chỉ thấy dữ liệu của mình; quyền PHẢI được kiểm tra ở API, không chỉ ở giao diện.
- Truy cập cơ sở dữ liệu PHẢI qua interface repository để đổi hệ cơ sở dữ liệu mà không sửa
  logic nghiệp vụ.
- Mật khẩu PHẢI lưu dạng băm; bí mật chỉ nằm trong biến môi trường, KHÔNG commit vào repo.

Lý do: dữ liệu học tập là dữ liệu cá nhân; MongoDB chỉ là lựa chọn tạm thời.

### VI. Kiểm thử logic nghiệp vụ

- Logic nghiệp vụ PHẢI có unit test (ví dụ: lịch ôn FSRS, quy tắc một ngày học, so sánh chép
  chính tả).
- Mỗi endpoint PHẢI có test cho trường hợp thành công, lỗi đầu vào và sai quyền.
- Test KHÔNG cần mạng: AI, TTS, STT, từ điển được giả lập.
- Mỗi tiêu chí nghiệm thu PHẢI có ít nhất một test hoặc một bước kiểm tra thủ công ghi trong
  tasks.

Lý do: logic ôn tập và chấm điểm sai âm thầm sẽ làm hỏng việc học mà khó phát hiện bằng mắt.

### VII. Đơn giản trước

- Một người phát triển, một người dùng: PHẢI chọn giải pháp đơn giản nhất đáp ứng tiêu chí
  nghiệm thu.
- KHÔNG thêm thư viện hay lớp trừu tượng khi chưa cần, trừ interface ở nguyên tắc III và V.

Lý do: mỗi lớp phức tạp thêm là chi phí bảo trì mà một người phải gánh một mình.

## Ràng buộc công nghệ

- Frontend Angular (standalone, signals, TypeScript strict). Backend Go, REST JSON. Cơ sở dữ
  liệu MongoDB (tạm thời).
- Chạy bằng Docker Compose trên máy không có GPU.
- Code Go tuân theo bộ skill samber/cc-skills-golang trong `.claude/skills`.
- Tên trong code, API và commit message bằng tiếng Anh; tài liệu, spec và chữ trên giao diện
  bằng tiếng Việt.

## Quy trình phát triển

- Mỗi tính năng trên một nhánh riêng: specify → clarify (nếu cần) → plan → tasks → implement.
- Thứ tự làm theo `docs/phases/`.
- Một tính năng chỉ được coi là xong khi: đạt mọi tiêu chí nghiệm thu, test qua, lint không
  lỗi, đã kiểm tra ở 360px cả chế độ sáng và tối.

## Governance

- Constitution cao hơn mọi quy ước khác trong dự án.
- Sửa constitution PHẢI tăng phiên bản theo semver: MAJOR khi bỏ hoặc đổi nghĩa nguyên tắc;
  MINOR khi thêm nguyên tắc hoặc mở rộng đáng kể; PATCH khi sửa câu chữ.
- Plan PHẢI có mục kiểm tra tuân thủ constitution; mọi ngoại lệ PHẢI ghi lý do trong mục
  Complexity Tracking của plan.

**Version**: 1.0.0 | **Ratified**: 2026-09-29 | **Last Amended**: 2026-09-29
