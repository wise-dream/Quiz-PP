# Инструкция для физических кнопок

## Запрос при нажатии кнопки

### Метод и URL
```
POST https://your-server.com/api/button/press
```

**Важно:** 
- Замените `your-server.com` на адрес вашего сервера
- В production: используйте адрес сервера напрямую
- В development: `http://localhost:443/api/button/press`

### Заголовки
```
Content-Type: application/json
```

### Тело запроса (JSON)
```json
{
  "macAddress": "AA:BB:CC:DD:EE:FF",
  "buttonId": "1"
}
```

**Поля:**
- `macAddress` (обязательно) - MAC адрес устройства кнопки (можно с двоеточиями или без)
- `buttonId` (опционально) - номер кнопки, если на устройстве несколько кнопок

### Примеры MAC адресов
```
"AA:BB:CC:DD:EE:FF"  ✅ (с двоеточиями)
"AABBCCDDEEFF"       ✅ (без разделителей)
"aa:bb:cc:dd:ee:ff" ✅ (в любом регистре)
```

### Успешный ответ

**Если кнопка привязана и вопрос активен:**
```json
{
  "success": true,
  "message": "Button press processed successfully",
  "processed": true
}
```

**Если кнопка не привязана к команде:**
```json
{
  "success": true,
  "message": "Button not assigned to a team",
  "processed": false
}
```

**Если вопрос не активен или кто-то уже ответил:**
```json
{
  "success": true,
  "message": "question not active in room ABCD",
  "processed": false
}
```

### Ошибки

**Кнопка не найдена:**
```json
{
  "success": false,
  "message": "button not found: AABBCCDDEEFF",
  "processed": false
}
```
HTTP статус: `400 Bad Request`

## Пример кода для ESP32/Arduino

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
    Serial.print("Response code: ");
    Serial.println(httpResponseCode);
    Serial.print("Response: ");
    Serial.println(response);
    
    // Парсим ответ
    if (response.indexOf("\"processed\":true") > 0) {
      Serial.println("✅ Button press processed!");
    } else {
      Serial.println("⚠️ Button press not processed (question inactive or already answered)");
    }
  } else {
    Serial.print("❌ Error: ");
    Serial.println(httpResponseCode);
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
  Serial.print("IP: ");
  Serial.println(WiFi.localIP());
}

void loop() {
  // Ваша логика чтения кнопки
  if (digitalRead(BUTTON_PIN) == LOW) {
    sendButtonPress();
    delay(500); // Защита от дребезга
  }
}
```

## Пример через curl

```bash
curl -X POST https://your-server.com/api/button/press \
  -H "Content-Type: application/json" \
  -d '{"macAddress": "AA:BB:CC:DD:EE:FF", "buttonId": "1"}'
```

## Важные моменты

1. **MAC адрес должен быть зарегистрирован** в системе (через админ-панель или API `/api/button/register`)

2. **Кнопка должна быть привязана к команде** (через админ-панель или API `/api/button/assign`)

3. **Вопрос должен быть активен** - админ должен активировать вопрос перед нажатием кнопки

4. **Только первое нажатие засчитывается** - если кто-то уже ответил, остальные нажатия игнорируются

5. **HTTPS рекомендуется** для production окружения

6. **Защита от дребезга** - добавьте задержку между нажатиями в коде контроллера

## Полный workflow

1. ✅ Регистрация кнопки (один раз) - через админ-панель или API
2. ✅ Привязка к команде (перед игрой) - через админ-панель
3. ✅ Активация вопроса - админ активирует вопрос
4. ✅ Нажатие кнопки - кнопка отправляет POST запрос
5. ✅ Ответ засчитывается - если кнопка первая и вопрос активен

