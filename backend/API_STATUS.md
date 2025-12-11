# Статус бэкенда и API эндпоинты

## Текущий статус

**Версия:** 1.0  
**Статус:** ✅ Реализовано и готово к использованию

### Реализованные модули

1. **WebSocket сервис** - Real-time коммуникация для викторин
2. **Hardware Buttons** - Управление физическими кнопками (SQLite)
3. **Presentations** - Интеграция с PowerPoint (JSON + файлы)
4. **PowerPoint API** - Активация/деактивация вопросов

### Хранилища данных

- **SQLite** (`/srv/data/quiz.db`) - Физические кнопки
- **JSON** (`/srv/data/presentations.json`) - Презентации и их привязки
- **Файлы** (`/srv/data/presentations/{docKey}/slides/`) - PNG снапшоты слайдов

---

## API Endpoints

### WebSocket

**`ws://host:port/ws`**

Подключение для real-time коммуникации.

**Query параметры:**
- `room` (string, optional) - Код комнаты (default: "default")
- `role` (string, optional) - Роль: "host", "admin", "viewer" (default: "viewer")

**События:**
- `create_room` - Создание комнаты
- `join` - Присоединение к комнате
- `admin_auth` - Аутентификация администратора
- `create_team` - Создание команды
- `join_team` - Присоединение к команде
- `click` - Нажатие кнопки игроком
- `host_set_state` - Изменение фазы викторины
- `start_question` - Начало вопроса
- `answer_received` - Получен ответ
- `answer_confirmation` - Подтверждение ответа
- `show_answer` - Показать правильный ответ
- `next_question` - Следующий вопрос

---

### PowerPoint интеграция

#### `POST /api/activate-question`

Активирует вопрос для комнаты.

**Request:**
```json
{
  "roomCode": "A1B2",
  "duration": 60
}
```

**Response:**
```json
{
  "success": true,
  "message": "Question activated successfully",
  "roomCode": "A1B2"
}
```

**Коды ответа:**
- `200` - Успешно
- `400` - Неверный запрос
- `404` - Комната не найдена

---

#### `POST /api/deactivate-question`

Деактивирует вопрос для комнаты.

**Request:**
```json
{
  "roomCode": "A1B2"
}
```

**Response:**
```json
{
  "success": true,
  "message": "Question deactivated successfully",
  "roomCode": "A1B2"
}
```

**Коды ответа:**
- `200` - Успешно
- `400` - Неверный запрос
- `404` - Комната не найдена

---

### Физические кнопки (Hardware Buttons)

#### `POST /api/button/press`

Регистрирует нажатие физической кнопки.

**Request:**
```json
{
  "macAddress": "AA:BB:CC:DD:EE:FF",
  "buttonId": "1"
}
```

**Response:**
```json
{
  "success": true,
  "message": "Button press processed successfully",
  "processed": true
}
```

**Особенности:**
- Автоматическая регистрация кнопки, если не существует
- Обрабатывается только если комната в фазе `active` и вопрос активен

---

#### `POST /api/button/register`

Регистрирует новую физическую кнопку.

**Request:**
```json
{
  "macAddress": "AA:BB:CC:DD:EE:FF",
  "buttonId": "1",
  "name": "Кнопка команды 1"
}
```

**Response:**
```json
{
  "id": "btn_EEFF",
  "macAddress": "AABBCCDDEEFF",
  "buttonId": "1",
  "name": "Кнопка команды 1",
  "roomCode": "",
  "teamId": "",
  "isActive": true,
  "pressCount": 0
}
```

---

#### `POST /api/button/assign` или `PUT /api/button/assign`

Привязывает кнопку к команде в комнате.

**Request:**
```json
{
  "macAddress": "AA:BB:CC:DD:EE:FF",
  "roomCode": "A1B2",
  "teamId": "team_1234567890"
}
```

**Response:**
```json
{
  "id": "btn_EEFF",
  "macAddress": "AABBCCDDEEFF",
  "roomCode": "A1B2",
  "teamId": "team_1234567890",
  "teamName": "Команда 1"
}
```

---

#### `POST /api/button/unassign`

Отвязывает кнопку от команды.

**Request:**
```json
{
  "macAddress": "AA:BB:CC:DD:EE:FF"
}
```

**Response:** `204 No Content`

---

#### `GET /api/button/list`

Возвращает список всех зарегистрированных кнопок.

**Response:**
```json
[
  {
    "id": "btn_EEFF",
    "macAddress": "AABBCCDDEEFF",
    "buttonId": "1",
    "name": "Кнопка 1",
    "roomCode": "A1B2",
    "teamId": "team_123",
    "teamName": "Команда 1",
    "isActive": true,
    "pressCount": 5,
    "lastPress": "2024-01-01T12:00:00Z"
  }
]
```

---

#### `GET /api/button/room/{roomCode}`

Возвращает все кнопки, привязанные к комнате.

**Response:**
```json
[
  {
    "id": "btn_EEFF",
    "macAddress": "AABBCCDDEEFF",
    "roomCode": "A1B2",
    "teamId": "team_123"
  }
]
```

---

#### `GET /api/button/{macAddress}`

Возвращает информацию о конкретной кнопке.

**Response:**
```json
{
  "id": "btn_EEFF",
  "macAddress": "AABBCCDDEEFF",
  "buttonId": "1",
  "name": "Кнопка 1",
  "roomCode": "A1B2",
  "teamId": "team_123",
  "isActive": true,
  "pressCount": 5
}
```

**Коды ответа:**
- `200` - Успешно
- `400` - Неверный запрос
- `404` - Кнопка не найдена

---

#### `DELETE /api/button/{macAddress}`

Удаляет кнопку из системы.

**Response:** `204 No Content`

**Коды ответа:**
- `204` - Успешно удалено
- `404` - Кнопка не найдена

---

### Презентации (PowerPoint интеграция)

#### `POST /quiz/api/presentation/link`

Связывает презентацию с комнатой.

**Request:**
```json
{
  "docKey": "https://.../presentation.pptx",
  "roomCode": "A1B2"
}
```

**Response:**
```json
{
  "success": true
}
```

**Коды ответа:**
- `200` - Успешно
- `400` - Неверный запрос (отсутствует docKey или roomCode)
- `500` - Ошибка сервера

---

#### `GET /quiz/api/presentation/room?docKey=...`

Получает код комнаты по презентации.

**Query параметры:**
- `docKey` (required) - Ключ документа презентации

**Response (успех):**
```json
{
  "success": true,
  "roomCode": "A1B2"
}
```

**Response (не найдено):**
```json
{
  "success": false
}
```

**Коды ответа:**
- `200` - Успешно (даже если не найдено)
- `400` - Отсутствует параметр docKey

---

#### `POST /quiz/api/presentation/slide-snapshot`

Сохраняет снапшот слайда (PNG изображение).

**Request:**
```json
{
  "docKey": "https://.../presentation.pptx",
  "slideId": "1",
  "imageBase64": "data:image/png;base64,iVBORw0KGgoAAA..."
}
```

**Response:**
```json
{
  "success": true,
  "imagePath": "presentations/docKey_123/slides/1.png"
}
```

**Особенности:**
- Автоматически отрезает префикс `data:image/png;base64,` если присутствует
- Сохраняет PNG в `presentations/{docKey}/slides/{slideId}.png`
- Создает директории автоматически

**Коды ответа:**
- `200` - Успешно
- `400` - Неверный запрос или некорректный base64
- `500` - Ошибка сохранения

---

#### `POST /quiz/api/presentation/slide-config`

Сохраняет конфигурацию слайда (таймер, очки).

**Request:**
```json
{
  "docKey": "https://.../presentation.pptx",
  "slideId": "1",
  "config": {
    "timeLimitSeconds": 30,
    "pointsCorrect": 10,
    "pointsWrong": 0
  }
}
```

**Response:**
```json
{
  "success": true
}
```

**Коды ответа:**
- `200` - Успешно
- `400` - Неверный запрос
- `500` - Ошибка сохранения

---

### Служебные endpoints

#### `GET /health`

Health check endpoint.

**Response:** `OK` (200)

---

#### `GET /*` (любой другой путь)

Статические файлы:
- `content-addin/*`, `taskpane-addin/*`, `shared/*` → `addin/`
- Остальные → `web/`

---

## Общие характеристики

### CORS

Все endpoints поддерживают CORS:
- `Access-Control-Allow-Origin: *`
- `Access-Control-Allow-Methods: GET, POST, PUT, DELETE, OPTIONS`
- `Access-Control-Allow-Headers: Content-Type, Authorization`

### Формат ответов

**Успешные ответы:**
- JSON с полем `success: true`
- HTTP статус `200` или `204`

**Ошибки:**
- HTTP статус `400` (Bad Request) - неверный запрос
- HTTP статус `404` (Not Found) - ресурс не найден
- HTTP статус `500` (Internal Server Error) - ошибка сервера
- Текстовое сообщение об ошибке в теле ответа

### Аутентификация

- WebSocket: через `admin_auth` событие с паролем комнаты
- HTTP API: без аутентификации (все endpoints публичные)

---

## Конфигурация

### Переменные окружения

**Сервер:**
- `PORT` (default: "443") - Порт сервера
- `HOST` (default: "0.0.0.0") - Хост сервера

**База данных:**
- `DB_PATH` (default: "/srv/data/quiz.db") - Путь к SQLite БД

**TLS:**
- `TLS_ENABLED` (default: true) - Включить TLS
- `TLS_CERT_FILE` (default: "cert.pem") - Сертификат
- `TLS_KEY_FILE` (default: "key.pem") - Приватный ключ
- `TLS_MIN_VERSION` (default: "1.2") - Минимальная версия TLS

**WebSocket:**
- `WS_READ_LIMIT` (default: 512) - Лимит размера сообщения (байты)
- `WS_READ_TIMEOUT` (default: 60) - Таймаут чтения (секунды)
- `WS_WRITE_TIMEOUT` (default: 10) - Таймаут записи (секунды)
- `WS_PING_PERIOD` (default: 54) - Период ping (секунды)
- `WS_PONG_WAIT` (default: 60) - Ожидание pong (секунды)
- `WS_MAX_MESSAGE_SIZE` (default: 512) - Максимальный размер сообщения (байты)
- `WS_ALLOWED_ORIGINS` - Разрешенные origins (через запятую)

---

## Статистика API

**Всего endpoints:** 18

**По типам:**
- WebSocket: 1
- PowerPoint интеграция: 2
- Физические кнопки: 8
- Презентации: 4
- Служебные: 2
- Статические файлы: 1

**По методам:**
- GET: 5
- POST: 10
- PUT: 1
- DELETE: 1
- WebSocket: 1

---

## Примеры использования

### Создание комнаты и привязка презентации

```bash
# 1. Создать комнату через WebSocket (получить roomCode)
# 2. Связать презентацию с комнатой
curl -X POST http://localhost:8081/quiz/api/presentation/link \
  -H "Content-Type: application/json" \
  -d '{
    "docKey": "https://example.com/presentation.pptx",
    "roomCode": "A1B2"
  }'
```

### Получение комнаты по презентации

```bash
curl "http://localhost:8081/quiz/api/presentation/room?docKey=https://example.com/presentation.pptx"
```

### Сохранение снапшота слайда

```bash
curl -X POST http://localhost:8081/quiz/api/presentation/slide-snapshot \
  -H "Content-Type: application/json" \
  -d '{
    "docKey": "https://example.com/presentation.pptx",
    "slideId": "1",
    "imageBase64": "data:image/png;base64,iVBORw0KGgoAAA..."
  }'
```

### Регистрация физической кнопки

```bash
curl -X POST http://localhost:8081/api/button/register \
  -H "Content-Type: application/json" \
  -d '{
    "macAddress": "AA:BB:CC:DD:EE:FF",
    "buttonId": "1",
    "name": "Кнопка команды 1"
  }'
```

---

## Примечания

- Все пути к данным используют директорию из `DB_PATH` (по умолчанию `/srv/data`)
- Презентации хранятся в JSON-файле, снапшоты - в поддиректориях
- Физические кнопки хранятся в SQLite БД
- Комнаты викторин хранятся в памяти (не персистентны)

