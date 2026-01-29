# Continuity Ledger

## Goal
- Чистый репозиторий: убраны лишние файлы и скрипт быстрого развёртывания.
- Развёртывание: systemd-сервис для бэкенда (локально порт 3009), nginx для фронтенда и прокси к API/WebSocket.

## Now / Next
- Готово: удалены deploy.sh, debug-websocket.sh, test-connection.html, quiz-server-https, QUICKSTART.md.
- Готово: DEPLOYMENT.md — короткая пошаговая инструкция (systemd + nginx).
- Актуально: развёртывание по DEPLOYMENT.md.

## Open questions
- Нет.

## Decisions
- Бэкенд за nginx: порт 3009, без TLS (TLS на nginx при необходимости).
- Фронтенд: статика из `frontend/build`, раздача через nginx.
- Скрипт deploy.sh удалён; единственный источник — команды в DEPLOYMENT.md.
