# Continuity Ledger

## Goal
- Чистый репозиторий: убраны лишние файлы и скрипт быстрого развёртывания.
- Развёртывание: systemd-сервис для бэкенда (локально порт 3009), nginx для фронтенда и прокси к API/WebSocket.

## Now / Next
- Готово: удалены deploy.sh, debug-websocket.sh, test-connection.html, quiz-server-https, QUICKSTART.md.
- Готово: DEPLOYMENT.md — короткая пошаговая инструкция (systemd + nginx).
- Готово: исправлена утечка памяти по комнатам и двойное закрытие client.Send (см. Decisions).
- Готово: все адреса запросов переведены на https://quiz.wise-dream.ru (addin, button, frontend .env.production, config.env, docker-compose).
- Актуально: развёртывание по DEPLOYMENT.md.

## Open questions
- Нет.

## Decisions
- Бэкенд за nginx: порт 3009, без TLS (TLS на nginx при необходимости).
- Фронтенд: статика из `frontend/build`, раздача через nginx.
- Скрипт deploy.sh удалён; единственный источник — команды в DEPLOYMENT.md.
- Комнаты: очистка неактивных комнат каждые 30 мин (1 ч без активности); LastActivity обновляется при действиях в комнате; при create_room комната создаётся только в handleCreateRoom (без дубликата по roomID).
- Клиенты: закрытие канала Send только через Client.CloseSend() (sync.Once), чтобы избежать panic при двойном закрытии.
- Продакшен-домен: https://quiz.wise-dream.ru (не wise-dream.ru); WebSocket wss://quiz.wise-dream.ru/quiz/ws, API https://quiz.wise-dream.ru/quiz/api.
