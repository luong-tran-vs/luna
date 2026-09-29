# Đầu vào cho /speckit.constitution

Dán toàn bộ khối bên dưới vào sau lệnh `/speckit.constitution`.

---

```text
Tạo constitution cho dự án LingoStudy: web app học tiếng Anh cá nhân, luyện 4 kỹ năng Nghe, Nói, Đọc, Viết quanh một bài học mỗi ngày. Viết constitution bằng tiếng Việt. Phiên bản 1.0.0, ngày phê chuẩn 2026-09-29.

NGUYÊN TẮC CỐT LÕI

I. Tài liệu sản phẩm là nguồn sự thật
- docs/mvp-features.md, docs/phases/*.md và docs/design-system.md quyết định sản phẩm làm gì.
- Spec, plan hay task mâu thuẫn với các tài liệu này thì sửa spec/plan/task, không tự ý đổi sản phẩm.
- Mỗi spec phải dẫn chiếu tính năng tương ứng (F1, F2, ... hoặc L) và bao phủ đủ tiêu chí nghiệm thu của nó.
- Chỉ làm tính năng thuộc giai đoạn đang thực hiện; không làm trước tính năng của giai đoạn sau.

II. Chi phí 0 đồng (KHÔNG THƯƠNG LƯỢNG)
- Không tính năng nào được phụ thuộc vào dịch vụ trả phí.
- AI dùng gói miễn phí (Gemini) hoặc mô hình chạy trên máy (Ollama, whisper.cpp). TTS chạy trên máy (Piper hoặc Kokoro). Từ điển là file SQLite offline.
- Font chữ và thư viện phải có giấy phép miễn phí (font: SIL OFL).

III. AI là phần bổ sung, không phải điều kiện để app chạy
- Khi AI lỗi, chậm hoặc hết hạn mức, các tính năng Đọc, Nghe, Ôn tập và quản trị bài học vẫn hoạt động.
- Mọi lời gọi AI đi qua một lớp trung gian (interface) để đổi nhà cung cấp chỉ bằng cấu hình.
- Tác vụ AI chạy nền, có trạng thái (đang chạy, xong, lỗi) và cho phép chạy lại.
- Kết quả AI được lưu lại; không gọi AI lặp lại cho cùng một đầu vào. Không gọi AI khi người dùng chỉ tra từ.

IV. Mobile-first và tiếp cận được
- Mọi màn hình dùng tốt từ chiều rộng 360px, không cuộn ngang.
- Có chế độ sáng và tối; màu, font, cỡ chữ chỉ dùng qua token trong design-system.md, không viết mã màu trực tiếp trong component.
- Độ tương phản đạt WCAG AA. Không truyền đạt thông tin chỉ bằng màu.
- Toàn bộ chữ trên giao diện bằng tiếng Việt.

V. Dữ liệu người học được bảo vệ
- Mọi dữ liệu cá nhân gắn với tài khoản; người dùng không bao giờ thấy dữ liệu của người khác.
- Backend chỉ truy cập cơ sở dữ liệu qua lớp repository, để đổi từ MongoDB sang hệ khác mà không sửa logic nghiệp vụ.
- Mật khẩu lưu dạng băm (argon2id hoặc bcrypt). Bí mật (API key, chuỗi kết nối) chỉ nằm trong biến môi trường, không commit vào repo.
- Quyền quản trị được kiểm tra ở API, không chỉ ở giao diện.
- Có sao lưu tự động mỗi ngày và cho phép xuất dữ liệu.

VI. Kiểm thử logic nghiệp vụ
- Bắt buộc có unit test cho logic nghiệp vụ: lịch ôn FSRS, quy tắc một ngày học (mở bài, streak, giới hạn thẻ, sang ngày theo múi giờ), so sánh chép chính tả, thứ tự lấy nghĩa khi tra từ.
- Mỗi endpoint API có test tích hợp cho trường hợp thành công, lỗi đầu vào và sai quyền.
- Lời gọi AI, TTS và từ điển được giả lập (mock) trong test; test không cần mạng.
- Mỗi tiêu chí nghiệm thu trong tài liệu giai đoạn phải có ít nhất một test hoặc một bước kiểm tra thủ công ghi rõ trong tasks.

VII. Đơn giản trước
- Một người phát triển, một người dùng: chọn giải pháp đơn giản nhất đáp ứng tiêu chí nghiệm thu.
- Backend là một service Go duy nhất; các dịch vụ AI (TTS, STT, LLM) chạy thành container riêng và được gọi qua HTTP.
- Không thêm thư viện hay lớp trừu tượng khi chưa có nhu cầu thật, trừ hai lớp bắt buộc ở nguyên tắc III và V.

RÀNG BUỘC CÔNG NGHỆ
- Frontend: Angular (standalone components), TypeScript strict mode.
- Backend: Go, REST API trả JSON, dùng context cho timeout và huỷ request.
- Cơ sở dữ liệu: MongoDB (tạm thời). File audio lưu trên ổ đĩa, không lưu trong cơ sở dữ liệu.
- AI: Gemini gói miễn phí (ưu tiên Flash-Lite), dự phòng Groq, OpenRouter hoặc Ollama. STT: whisper.cpp. TTS: Piper hoặc Kokoro.
- Từ điển Anh–Việt: minhqnd/dictionary (SQLite, CC BY-SA 4.0, phải ghi nguồn).
- Chạy toàn bộ bằng Docker Compose trên máy không có GPU.
- Cấu trúc mã nguồn theo docs/architecture.md. Frontend: chia theo tính năng (core, shared, features), interceptor và guard dạng hàm. Backend: tuân theo bộ skill samber/cc-skills-golang; chia theo domain + ports; dependency injection bằng constructor viết tay, không dùng thư viện DI.
- Tên biến, hàm, API và commit message bằng tiếng Anh; tài liệu, spec và chữ trên giao diện bằng tiếng Việt.

QUY TRÌNH PHÁT TRIỂN
- Mỗi tính năng đi theo thứ tự: specify → clarify (nếu cần) → plan → tasks → implement, trên một nhánh riêng.
- Làm theo thứ tự giai đoạn 1 → 2 → 3; trong giai đoạn 1 theo thứ tự F1 → F2 → F3 → F4 → F5 → L → F6 → F12 → F13.
- Một tính năng chỉ được coi là xong khi: đạt mọi tiêu chí nghiệm thu, toàn bộ test qua, lint (golangci-lint, ESLint) không lỗi, và đã kiểm tra trên màn hình 360px ở cả chế độ sáng và tối.

QUẢN TRỊ
- Constitution có hiệu lực cao hơn mọi quy ước khác trong dự án.
- Sửa constitution phải tăng phiên bản theo semver (MAJOR: bỏ hoặc đổi nghĩa nguyên tắc; MINOR: thêm nguyên tắc hoặc mục; PATCH: sửa câu chữ) và ghi ngày sửa.
- Plan phải có mục kiểm tra tuân thủ constitution; vi phạm nào cũng phải ghi lý do và được chấp nhận rõ ràng.
```
