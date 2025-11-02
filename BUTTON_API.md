# API для физических кнопок

## Обзор

Система поддерживает физические кнопки (hardware buttons), которые могут отправлять нажатия через REST API. Каждая кнопка идентифицируется по MAC-адресу устройства и может быть привязана к определенной команде в комнате.

## Архитектура

- **REST API** для кнопок (не WebSocket - проще для микроконтроллеров)
- **In-memory хранилище** кнопок (можно заменить на БД при необходимости)
- **Привязка через MAC-адрес** к команде
- **Интеграция с существующей логикой** - нажатия обрабатываются так же, как ответы игроков

## API Endpoints

### 1. Регистрация кнопки
```
POST /api/button/register
Content-Type: application/json

{
  "macAddress": "AA:BB:CC:DD:EE:FF",
  "buttonId": "1",
  "name": "Кнопка команды 1"
}
```

**Ответ:**
```json
{
  "id": "btn_EEFF",
  "macAddress": "AABBCCDDEEFF",
  "buttonId": "1",
  "name": "Кнопка команды 1",
  "roomCode": "",
  "teamId": "",
  "teamName": "",
  "isActive": true,
  "lastPress": "0001-01-01T00:00:00Z",
  "pressCount": 0,
  "createdAt": "2024-01-01T12:00:00Z",
  "updatedAt": "2024-01-01T12:00:00Z"
}
```

### 2. Привязка кнопки к команде
```
POST /api/button/assign
Content-Type: application/json

{
  "macAddress": "AA:BB:CC:DD:EE:FF",
  "roomCode": "ABCD",
  "teamId": "team_1234567890"
}
```

**Ответ:** Обновленная информация о кнопке с заполненными `roomCode`, `teamId`, `teamName`.

### 3. Отправка нажатия кнопки
```
POST /api/button/press
Content-Type: application/json

{
  "macAddress": "AA:BB:CC:DD:EE:FF",
  "buttonId": "1"  // опционально
}
```

**Ответ:**
```json
{
  "success": true,
  "message": "Button press processed successfully",
  "processed": true
}
```

Если вопрос не активен или кто-то уже ответил:
```json
{
  "success": true,
  "message": "question not active in room ABCD",
  "processed": false
}
```

### 4. Список всех кнопок
```
GET /api/button/list
```

### 5. Информация о кнопке
```
GET /api/button/{macAddress}
```

### 6. Отвязка кнопки от команды
```
POST /api/button/unassign
Content-Type: application/json

{
  "macAddress": "AA:BB:CC:DD:EE:FF"
}
```

### 7. Удаление кнопки
```
DELETE /api/button/{macAddress}
```

## Пример кода для ESP32/Arduino

### Основная логика
```cpp
#include <WiFi.h>
#include <HTTPClient.h>
#include <esp_mac.h>

const char* ssid = "YOUR_WIFI_SSID";
const char* password = "YOUR_WIFI_PASSWORD";
const char* serverUrl = "https://your-server.com/api/button/press";

// Получить MAC адрес ESP32
String getMACAddress() {
  uint8_t mac[6];
  esp_read_mac(mac, ESP_MAC_WIFI_STA);
  char macStr[18];
  sprintf(macStr, "%02X:%02X:%02X:%02X:%02X:%02X",
          mac[0], mac[1], mac[2], mac[3], mac[4], mac[5]);
  return String(macStr);
}

void sendButtonPress() {
  HTTPClient http;
  http.begin(serverUrl);
  http.addHeader("Content-Type", "application/json");
  
  String macAddress = getMACAddress();
  String jsonPayload = "{\"macAddress\":\"" + macAddress + "\",\"buttonId\":\"1\"}";
  
  int httpResponseCode = http.POST(jsonPayload);
  
  if (httpResponseCode > 0) {
    String response = http.getString();
    Serial.println("Response: " + response);
  } else {
    Serial.println("Error: " + String(httpResponseCode));
  }
  
  http.end();
}

void setup() {
  Serial.begin(115200);
  
  WiFi.begin(ssid, password);
  while (WiFi.status() != WL_CONNECTED) {
    delay(1000);
    Serial.println("Connecting to WiFi...");
  }
  
  Serial.println("WiFi connected!");
  
  // Регистрация кнопки (выполнить один раз)
  registerButton();
  
  // Привязка к команде (выполнить один раз через админ-панель или API)
  // assignButtonToTeam();
}

void loop() {
  // Ваша логика чтения кнопки
  if (digitalRead(BUTTON_PIN) == LOW) {
    sendButtonPress();
    delay(500); // Защита от дребезга
  }
}
```

### Нумерация кнопок в контроллере

Если у вас несколько кнопок на одном устройстве, можно пронумеровать их в коде:

```cpp
const int BUTTON_PIN_1 = 2;
const int BUTTON_PIN_2 = 4;
const int BUTTON_PIN_3 = 5;

String sendButtonPress(int buttonNumber) {
  HTTPClient http;
  http.begin(serverUrl);
  http.addHeader("Content-Type", "application/json");
  
  String macAddress = getMACAddress();
  String buttonId = String(buttonNumber);
  String jsonPayload = "{\"macAddress\":\"" + macAddress + 
                       "\",\"buttonId\":\"" + buttonId + "\"}";
  
  int httpResponseCode = http.POST(jsonPayload);
  // ... обработка ответа
}

void loop() {
  if (digitalRead(BUTTON_PIN_1) == LOW) {
    sendButtonPress(1);
    delay(500);
  }
  if (digitalRead(BUTTON_PIN_2) == LOW) {
    sendButtonPress(2);
    delay(500);
  }
  // и т.д.
}
```

## Workflow использования

1. **Регистрация кнопок:**
   - При первом запуске каждой кнопки она автоматически регистрируется
   - Или можно зарегистрировать через API `/api/button/register`

2. **Привязка к командам:**
   - Администратор через админ-панель привязывает кнопки к командам
   - Или через API `/api/button/assign`
   - Требуется: MAC-адрес, код комнаты, ID команды

3. **Во время игры:**
   - Кнопка отправляет POST запрос на `/api/button/press` при нажатии
   - Сервер проверяет:
     - Зарегистрирована ли кнопка
     - Привязана ли к команде
     - Активен ли вопрос
     - Не ответил ли кто-то раньше
   - Если все ОК - засчитывается ответ команды

## Сущности в системе

### HardwareButton (модель)
- `ID` - уникальный идентификатор кнопки
- `MACAddress` - MAC-адрес устройства
- `ButtonID` - номер кнопки в устройстве (если несколько кнопок на устройстве)
- `Name` - человеко-читаемое имя
- `RoomCode` - код комнаты
- `TeamID` - ID команды
- `TeamName` - имя команды (кешируется)
- `IsActive` - активна ли кнопка
- `PressCount` - количество нажатий
- `LastPress` - время последнего нажатия

## Преимущества REST API над WebSocket

1. **Простота реализации** - не нужно поддерживать постоянное соединение
2. **Меньше потребление ресурсов** - для одноразовых событий
3. **Надежность** - не зависит от качества соединения
4. **Удобство отладки** - легко тестировать через curl/Postman
5. **Совместимость** - работает с любыми устройствами, поддерживающими HTTP

## Защита от гонок (race conditions)

Как и для обычных игроков, защита от одновременных нажатий реализована через:
- Мьютексы на уровне комнаты
- Проверку `QuestionActive` и `FirstAnswerer`
- Только первое нажатие обрабатывается, остальные игнорируются

## База данных

Система использует SQLite для постоянного хранения данных о кнопках:
- База данных по умолчанию: `./data/quiz.db`
- Можно изменить через переменную окружения `DB_PATH`
- Файл БД включен в git для простого обновления на сервере

### Структура БД

Таблица `hardware_buttons` содержит:
- `id` - уникальный идентификатор кнопки
- `mac_address` - MAC адрес (уникальный)
- `button_id` - номер кнопки в устройстве
- `name` - название кнопки
- `room_code` - код комнаты
- `team_id` - ID команды
- `team_name` - имя команды
- `is_active` - активна ли кнопка
- `press_count` - количество нажатий
- `last_press` - время последнего нажатия
- `created_at` - время создания
- `updated_at` - время обновления

### Админ-панель

В админ-панели добавлен раздел "Физические кнопки" для:
- Просмотра всех зарегистрированных кнопок
- Регистрации новых кнопок (по MAC адресу)
- Привязки кнопок к командам в комнате
- Отвязки кнопок от команд
- Удаления кнопок

## Пример работы с кнопками через API

```bash
# 1. Регистрация кнопки
curl -X POST https://your-server.com/api/button/register \
  -H "Content-Type: application/json" \
  -d '{"macAddress": "AA:BB:CC:DD:EE:FF", "buttonId": "1", "name": "Кнопка команды 1"}'

# 2. Привязка к команде (нужен roomCode и teamId из комнаты)
curl -X POST https://your-server.com/api/button/assign \
  -H "Content-Type: application/json" \
  -d '{"macAddress": "AA:BB:CC:DD:EE:FF", "roomCode": "ABCD", "teamId": "team_1234567890"}'

# 3. Отправка нажатия (из кода контроллера)
curl -X POST https://your-server.com/api/button/press \
  -H "Content-Type: application/json" \
  -d '{"macAddress": "AA:BB:CC:DD:EE:FF", "buttonId": "1"}'
```

