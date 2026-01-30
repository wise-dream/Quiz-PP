# Развёртывание PowerPoint Quiz (Linux)

Краткая инструкция: бэкенд как systemd-сервис на порту **3009**, фронтенд через **nginx**.

---

## 1. Подготовка сервера

```bash
sudo apt update && sudo apt upgrade -y
sudo apt install -y nginx build-essential
```

---

## 2. Бэкенд (Go, порт 3009)

### Сборка

```bash
cd /path/to/Quiz-PP/backend
go build -o quiz-backend ./cmd/server
```

### Каталог и данные

```bash
sudo mkdir -p /srv/quiz/data
sudo chown "$USER:$USER" /srv/quiz/data
```

### Unit-файл systemd

Создайте `/etc/systemd/system/quiz-backend.service`:

```ini
[Unit]
Description=PowerPoint Quiz Backend
After=network.target

[Service]
Type=simple
User=www-data
Group=www-data
WorkingDirectory=/srv/quiz
ExecStart=/srv/quiz/quiz-backend
Restart=always
RestartSec=5
Environment=PORT=3009
Environment=HOST=127.0.0.1
Environment=TLS_ENABLED=false
Environment=DB_PATH=/srv/quiz/data/quiz.db

[Install]
WantedBy=multi-user.target
```

### Установка и запуск

```bash
sudo cp /path/to/Quiz-PP/backend/quiz-backend /srv/quiz/
sudo chown www-data:www-data /srv/quiz/quiz-backend
sudo chmod +x /srv/quiz/quiz-backend
sudo systemctl daemon-reload
sudo systemctl enable quiz-backend
sudo systemctl start quiz-backend
sudo systemctl status quiz-backend
```

Проверка: `curl -s http://127.0.0.1:3009/health` — должен ответить `healthy`.

---

## 3. Фронтенд (сборка + nginx)

### Сборка React

```bash
cd /path/to/Quiz-PP/frontend
npm ci
npm run build
```

В `frontend/.env.production` задайте URL вашего домена (или IP), например:

```env
REACT_APP_WS_URL=wss://your-domain/ws
REACT_APP_API_URL=https://your-domain/api
```

После изменения `.env.production` пересоберите: `npm run build`.

### Размещение статики

```bash
sudo mkdir -p /var/www/quiz
sudo cp -r /path/to/Quiz-PP/frontend/build/* /var/www/quiz/
sudo chown -R www-data:www-data /var/www/quiz
```

---

## 4. Nginx (прокси к бэкенду и раздача фронта)

Создайте конфиг, например `/etc/nginx/sites-available/quiz`:

```nginx
server {
    listen 80;
    server_name _;

    root /var/www/quiz;
    index index.html;

    location / {
        try_files $uri $uri/ /index.html;
    }

    location /api/ {
        proxy_pass http://127.0.0.1:3009;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # WebSocket: бэкенд слушает только /ws; публичный путь /quiz/ws для фронта и аддина
    location /ws {
        proxy_pass http://127.0.0.1:3009;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_read_timeout 86400;
        proxy_send_timeout 86400;
    }
    location /quiz/ws {
        proxy_pass http://127.0.0.1:3009/ws;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_read_timeout 86400;
        proxy_send_timeout 86400;
    }

    location /addin/ {
        proxy_pass http://127.0.0.1:3009/addin/;
        proxy_set_header Host $host;
        expires 1y;
        add_header Cache-Control "public, immutable";
    }
}
```

Включите сайт и перезагрузите nginx:

```bash
sudo ln -sf /etc/nginx/sites-available/quiz /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx
```

---

## 5. Полезные команды

| Действие | Команда |
|----------|--------|
| Статус бэкенда | `sudo systemctl status quiz-backend` |
| Логи бэкенда | `sudo journalctl -u quiz-backend -f` |
| Перезапуск бэкенда | `sudo systemctl restart quiz-backend` |
| Проверка nginx | `sudo nginx -t` |
| Перезагрузка nginx | `sudo systemctl reload nginx` |
| Обновление фронта | `cd frontend && npm run build && sudo cp -r build/* /var/www/quiz/` |
| Обновление бэкенда | пересобрать бинарник, скопировать в `/srv/quiz/`, `sudo systemctl restart quiz-backend` |

---

## 6. Файрвол (по желанию)

```bash
sudo ufw allow 22/tcp
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw enable
```

---

## Итог

- Бэкенд: systemd-сервис `quiz-backend`, слушает **127.0.0.1:3009**, без TLS.
- Фронтенд: статика в `/var/www/quiz`, раздаётся nginx.
- Nginx: проксирует `/api/`, `/ws`, `/addin/` на `http://127.0.0.1:3009`.
- Доступ: `http://IP-сервера` или `http://your-domain` (при необходимости настройте SSL в nginx и `server_name`).
