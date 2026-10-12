# Triển khai Luna lên Ubuntu với Nginx

Hướng dẫn đưa Luna lên một VPS Ubuntu: Nginx phục vụ frontend (file tĩnh) và chuyển `/api/` sang backend Go
chạy bằng systemd. Frontend và backend cùng một tên miền, nên bản build production gọi `/api` là đủ, không cần CORS.

Ví dụ dưới đây dùng tên miền `luongtran.site` và mã nguồn nằm ở `/root/luna`. Đổi lại cho đúng máy của bạn.

## 0. Chuẩn bị

- VPS Ubuntu, đăng nhập được bằng SSH.
- Bản ghi DNS `A` của tên miền trỏ về IP của VPS (cần cho bước SSL).
- Trên VPS đã cài **Go 1.25**, **Node 22** và `git`.
- Mã nguồn đã clone về `/root/luna`:

```bash
git clone https://github.com/luong-tran-vs/luna.git /root/luna
```

## 1. Tường lửa (ufw)

Mở cổng SSH **trước** khi bật ufw, nếu không sẽ tự khoá mình ra khỏi máy.

```bash
sudo apt update
sudo apt install ufw -y

sudo ufw allow 22/tcp
sudo ufw enable
sudo ss -tlnp | grep ':22'     # kiểm tra SSH vẫn đang nghe

sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
```

Không mở cổng `8080`: backend chỉ cần nhận request từ Nginx trên chính máy này.

## 2. Cài Nginx

```bash
sudo apt install nginx -y
sudo systemctl enable nginx
sudo systemctl status nginx
```

## 3. Backend

### 3.1. Cấu hình

```bash
cd /root/luna/backend
cp .env.example .env
nano .env
```

Các biến cần xem lại khi chạy production (danh sách đầy đủ ở mục **Cấu hình** trong `README.md`):

| Biến | Giá trị production |
|---|---|
| `MONGO_URI` (hoặc `MYSQL_*`) | Chuỗi kết nối cơ sở dữ liệu thật |
| `HTTP_ADDR` | `127.0.0.1:8080`: chỉ nghe nội bộ, Nginx chuyển request vào |
| `COOKIE_SECURE` | `true`: site chạy HTTPS (bước 5) |
| `CORS_ORIGINS` | Để trống: frontend và API cùng một tên miền |
| `LOG_LEVEL` | `info` |
| `GEMINI_API_KEY` | Khoá Gemini nếu dùng AI |

Backend đọc file `.env` trong thư mục đang chạy, nên service ở bước 3.3 đặt `WorkingDirectory` là `/root/luna/backend`.

Tải từ điển offline cho bước Đọc (một lần, khoảng 180 MB, lưu vào `deploy/data/dictionary/`):

```bash
/root/luna/deploy/fetch-dictionary.sh
```

### 3.2. Build

```bash
cd /root/luna/backend
CGO_ENABLED=0 go build -trimpath -o luna-backend ./cmd/server
```

Phải build cả package `./cmd/server`, **không** build riêng `cmd/server/main.go`: thư mục này còn có
`tzdata.go`, chỉ định một file thì Go sẽ bỏ qua file đó.

### 3.3. Chạy bằng systemd

```bash
sudo nano /etc/systemd/system/luna-backend.service
```

```ini
[Unit]
Description=Luna Go Backend
After=network.target

[Service]
Type=simple
User=root

WorkingDirectory=/root/luna/backend
ExecStart=/root/luna/backend/luna-backend

Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl daemon-reload
sudo systemctl start luna-backend     # khởi động backend
sudo systemctl enable luna-backend    # tự khởi động cùng Ubuntu
sudo systemctl status luna-backend    # kiểm tra trạng thái
```

Xem log: `sudo journalctl -u luna-backend -f`.

## 4. Frontend

Frontend là file tĩnh. Build ra `frontend/dist/luna/browser/`, rồi chép vào thư mục Nginx phục vụ.

```bash
sudo mkdir -p /var/www/luna
```

Tạo script build ở `/root/luna/build.sh`. Script lấy code mới nhất từ GitHub, cài dependency, build rồi chép
sang `/var/www/luna`:

```bash
cd /root/luna
nano build.sh
```

```bash
#!/usr/bin/env bash
set -euo pipefail
cd /root/luna
git pull
cd frontend
npm ci
npm run build
sudo cp -a /root/luna/frontend/dist/luna/browser/. /var/www/luna/
```

```bash
chmod +x build.sh
./build.sh
```

> Nếu `npm ci` báo `Cannot read properties of null (reading 'edgesOut')` (lỗi của npm 10.9), thay bằng
> `npx -y npm@11 ci`.

## 5. Cấu hình Nginx

```bash
sudo nano /etc/nginx/sites-available/luna
```

```nginx
server {
    listen 80;
    listen [::]:80;

    server_name luongtran.site;

    root /var/www/luna;
    index index.html;

    # Angular SPA: đường dẫn không phải file (vd. /review) trả về index.html
    location / {
        try_files $uri $uri/ /index.html;
    }

    # Go backend API
    location /api/ {
        proxy_pass http://127.0.0.1:8080;

        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # Không cache trang chính và các file của service worker (PWA),
    # nếu không người dùng sẽ kẹt ở bản cũ sau khi deploy
    location ~ ^/(index\.html|ngsw\.json|ngsw-worker\.js|safety-worker\.js|manifest\.webmanifest)$ {
        add_header Cache-Control "no-cache";
    }
}
```

Bật site, gỡ site mặc định, kiểm tra cú pháp rồi nạp lại:

```bash
sudo ln -s /etc/nginx/sites-available/luna /etc/nginx/sites-enabled/luna
sudo rm -f /etc/nginx/sites-enabled/default
sudo nginx -t
sudo systemctl reload nginx
```

Lúc này mở `http://luongtran.site` là thấy app.

## 6. Cài SSL (HTTPS)

Bắt buộc để cài app dạng PWA: service worker chỉ chạy trên HTTPS.

```bash
sudo apt update
sudo apt install certbot python3-certbot-nginx -y

sudo certbot --nginx -d luongtran.site
sudo nginx -t
sudo systemctl reload nginx
```

Certbot tự thêm khối `listen 443 ssl` và chuyển hướng HTTP sang HTTPS vào file cấu hình ở bước 5, và tự gia hạn
chứng chỉ (kiểm tra bằng `sudo certbot renew --dry-run`).

Nhớ đặt `COOKIE_SECURE=true` trong `backend/.env` rồi `sudo systemctl restart luna-backend`.

## 7. Cập nhật bản mới

```bash
cd /root/luna
git pull

# Backend (khi có thay đổi trong backend/)
cd backend
CGO_ENABLED=0 go build -trimpath -o luna-backend ./cmd/server
sudo systemctl restart luna-backend

# Frontend (khi có thay đổi trong frontend/)
/root/luna/build.sh
```

Chỉ sửa frontend thì chạy mỗi `/root/luna/build.sh` là đủ, vì script đã tự `git pull`.

Frontend không cần reload Nginx. Người đã cài app sẽ tự nhận bản mới ở lần chuyển trang kế tiếp.

## Xử lý sự cố

| Triệu chứng | Kiểm tra |
|---|---|
| Trang báo lỗi kết nối ở chân trang | `sudo systemctl status luna-backend`, `sudo journalctl -u luna-backend -n 100` |
| `502 Bad Gateway` ở `/api/` | Backend chưa chạy, hoặc `HTTP_ADDR` không phải cổng `8080` |
| F5 ở một trang con ra `404` | Thiếu `try_files ... /index.html` trong cấu hình Nginx |
| Không hiện nút cài app | Site chưa chạy HTTPS (bước 6) |
| Deploy xong vẫn thấy bản cũ | Khối `Cache-Control "no-cache"` ở bước 5; thử Ctrl+F5 |
| Log Nginx | `sudo tail -f /var/log/nginx/error.log` |
