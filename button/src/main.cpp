// File: src/main.cpp
// ESP32-C3 SuperMini — Wi-Fi "кнопка": отправляет HTTP(S) запросы на сервер квиза.
// Поддерживает открытые сети (SSID без пароля): если WIFI_PASS == "" → WiFi.begin(SSID).
// Триггеры: физическая кнопка (BTN_PIN) и клавиша 's' в Serial.

#include <Arduino.h>
#include <WiFi.h>
#include <HTTPClient.h>
#include <WiFiClient.h>
#include <WiFiClientSecure.h>

#ifndef WIFI_SSID
#define WIFI_SSID "YourSSID"
#endif
#ifndef WIFI_PASS
#define WIFI_PASS "YourPassword"
#endif
#ifndef SERVER_URL
#define SERVER_URL "https://your-server.com"
#endif
#ifndef BUTTON_ID
#define BUTTON_ID "1"
#endif
#ifndef AUTO_SEND_INTERVAL_MS
#define AUTO_SEND_INTERVAL_MS 0ul  // Выключено по умолчанию (только кнопка)
#endif
#ifndef BTN_PIN
#define BTN_PIN 9  // GPIO9 на ESP32-C3 (можно изменить)
#endif
#ifndef USE_TLS_INSECURE
#define USE_TLS_INSECURE 1  // Для самоподписанных сертификатов
#endif

#ifdef AUTH_BEARER
#define HAS_AUTH_BEARER 1
#endif
#ifdef X_API_KEY
#define HAS_X_API_KEY 1
#endif

const unsigned long DEBOUNCE_MS = 50;  // Защита от дребезга (мс) - как в рабочем примере
unsigned long lastAutoSend = 0;
int lastButtonState = HIGH;  // Изменено с lastBtn для совместимости с рабочей логикой
int currentButtonState = HIGH;  // Добавлено для правильной обработки debounce
unsigned long lastDebounceTime = 0;  // Изменено с lastBtnChange для совместимости
unsigned long lastPressTime = 0;
const unsigned long PRESS_COOLDOWN_MS = 500;  // Минимальный интервал между нажатиями

// Получить MAC адрес в формате строки
String getMacAddress() {
  return WiFi.macAddress();
}

// Нормализовать MAC адрес (убрать двоеточия и привести к верхнему регистру)
String normalizeMacAddress(const String& mac) {
  String normalized = mac;
  normalized.toUpperCase();
  normalized.replace(":", "");
  return normalized;
}

static void wifiBeginSmart() {
  // если пароль пустой — подключаемся как к открытой сети
  if (strlen(WIFI_PASS) == 0) {
    WiFi.begin(WIFI_SSID);
  } else {
    WiFi.begin(WIFI_SSID, WIFI_PASS);
  }
}

void connectWiFiBlocking() {
  Serial.printf("Wi-Fi: подключаюсь к \"%s\"%s...\n",
                WIFI_SSID, strlen(WIFI_PASS) ? "" : " (open)");
  WiFi.mode(WIFI_STA);
  WiFi.setAutoReconnect(true);
  WiFi.persistent(false);
  wifiBeginSmart();

  unsigned long t0 = millis();
  const unsigned long TIMEOUT_MS = 15000;
  while (WiFi.status() != WL_CONNECTED && (millis() - t0) < TIMEOUT_MS) {
    delay(250);
    Serial.print(".");
  }
  Serial.println();
  if (WiFi.status() == WL_CONNECTED) {
    Serial.printf("Wi-Fi OK: IP=%s RSSI=%d dBm\n",
                  WiFi.localIP().toString().c_str(), WiFi.RSSI());
  } else {
    Serial.println("Wi-Fi не подключен (таймаут). Буду пытаться в фоне.");
  }
}

bool ensureWiFi() {
  if (WiFi.status() == WL_CONNECTED) return true;
  WiFi.disconnect();
  wifiBeginSmart();
  unsigned long t0 = millis();
  const unsigned long TIMEOUT_MS = 8000;
  while (WiFi.status() != WL_CONNECTED && (millis() - t0) < TIMEOUT_MS) {
    delay(200);
  }
  return WiFi.status() == WL_CONNECTED;
}

void addCommonHeaders(HTTPClient& http) {
  http.setUserAgent("ESP32C3-Button/1.0");
  http.addHeader("Connection", "close");
#ifdef HAS_AUTH_BEARER
  http.addHeader("Authorization", String("Bearer ") + AUTH_BEARER);
#endif
#ifdef HAS_X_API_KEY
  http.addHeader("X-API-Key", X_API_KEY);
#endif
}

// Отправка нажатия кнопки на сервер
int sendButtonPress() {
  unsigned long requestStartTime = millis();
  String separator = String("============================================================"); // 60 символов
  
  Serial.println();
  Serial.println(separator);
  Serial.println("[HTTP REQUEST] ========== Начало запроса ==========");
  
  if (!ensureWiFi()) {
    Serial.println("[ERROR] ❌ Нет Wi-Fi подключения, пропускаю отправку.");
    Serial.println(separator);
    Serial.println();
    return -1;
  }

  // Проверка кулдауна между нажатиями
  unsigned long now = millis();
  if (lastPressTime > 0 && (now - lastPressTime) < PRESS_COOLDOWN_MS) {
    Serial.printf("[SKIP] ⏸️  Слишком быстрое нажатие (cooldown: %lu мс), пропускаю.\n", PRESS_COOLDOWN_MS);
    Serial.println(separator);
    Serial.println();
    return -2;
  }
  lastPressTime = now;

  String macAddress = getMacAddress();
  String endpoint = String(SERVER_URL) + "/api/button/press";
  
  // Формируем JSON payload согласно API бэкенда
  String payload = String("{\"macAddress\":\"") + macAddress +
                   "\",\"buttonId\":\"" + String(BUTTON_ID) + "\"}";

  // Подробное логирование запроса
  Serial.println("[REQUEST INFO]");
  Serial.printf("  Method: POST\n");
  Serial.printf("  URL: %s\n", endpoint.c_str());
  Serial.printf("  Protocol: %s\n", endpoint.startsWith("https://") ? "HTTPS" : "HTTP");
  Serial.printf("  MAC Address: %s\n", macAddress.c_str());
  Serial.printf("  Button ID: %s\n", BUTTON_ID);
  Serial.printf("  Timestamp: %lu ms\n", now);
  
  Serial.println("\n[REQUEST HEADERS]");
  Serial.println("  User-Agent: ESP32C3-Button/1.0");
  Serial.println("  Connection: close");
  Serial.println("  Content-Type: application/json");
#ifdef HAS_AUTH_BEARER
  Serial.printf("  Authorization: Bearer %s\n", AUTH_BEARER);
#endif
#ifdef HAS_X_API_KEY
  Serial.printf("  X-API-Key: %s\n", X_API_KEY);
#endif
  
  Serial.println("\n[REQUEST BODY]");
  Serial.printf("  %s\n", payload.c_str());

  int httpCode = -1;
  String response = "";
  unsigned long responseTime = 0;

  Serial.println("\n[NETWORK] Отправка запроса...");
  
  if (endpoint.startsWith("https://")) {
    WiFiClientSecure client;
#if USE_TLS_INSECURE
    client.setInsecure();  // Для самоподписанных сертификатов
    Serial.println("  TLS: Insecure mode (самоподписанный сертификат)");
#endif
    HTTPClient http;
    http.setTimeout(10000);  // 10 секунд таймаут
    if (http.begin(client, endpoint)) {
      addCommonHeaders(http);
      http.addHeader("Content-Type", "application/json");
      
      unsigned long sendStart = millis();
      httpCode = http.POST(payload);
      responseTime = millis() - sendStart;
      response = http.getString();
      
      Serial.println("\n[RESPONSE]");
      Serial.printf("  HTTP Status Code: %d\n", httpCode);
      Serial.printf("  Response Time: %lu ms\n", responseTime);
      Serial.printf("  Response Size: %d bytes\n", response.length());
      
      // Логируем заголовки ответа, если доступны
      int headerCount = http.headers();
      if (headerCount > 0) {
        Serial.println("\n[RESPONSE HEADERS]");
        for (int i = 0; i < headerCount; i++) {
          String headerName = http.headerName(i);
          String headerValue = http.header(i);
          Serial.printf("  %s: %s\n", headerName.c_str(), headerValue.c_str());
        }
      }
      
      Serial.println("\n[RESPONSE BODY]");
      if (response.length() > 0) {
        Serial.printf("  %s\n", response.c_str());
      } else {
        Serial.println("  (пусто)");
      }
      
      http.end();
    } else {
      Serial.println("[ERROR] ❌ HTTP begin() failed (HTTPS).");
      Serial.println(separator);
      Serial.println();
      return -1;
    }
  } else {
    WiFiClient client;
    HTTPClient http;
    http.setTimeout(10000);
    if (http.begin(client, endpoint)) {
      addCommonHeaders(http);
      http.addHeader("Content-Type", "application/json");
      
      unsigned long sendStart = millis();
      httpCode = http.POST(payload);
      responseTime = millis() - sendStart;
      response = http.getString();
      
      Serial.println("\n[RESPONSE]");
      Serial.printf("  HTTP Status Code: %d\n", httpCode);
      Serial.printf("  Response Time: %lu ms\n", responseTime);
      Serial.printf("  Response Size: %d bytes\n", response.length());
      
      // Логируем заголовки ответа, если доступны
      int headerCount = http.headers();
      if (headerCount > 0) {
        Serial.println("\n[RESPONSE HEADERS]");
        for (int i = 0; i < headerCount; i++) {
          String headerName = http.headerName(i);
          String headerValue = http.header(i);
          Serial.printf("  %s: %s\n", headerName.c_str(), headerValue.c_str());
        }
      }
      
      Serial.println("\n[RESPONSE BODY]");
      if (response.length() > 0) {
        Serial.printf("  %s\n", response.c_str());
      } else {
        Serial.println("  (пусто)");
      }
      
      http.end();
    } else {
      Serial.println("[ERROR] ❌ HTTP begin() failed (HTTP).");
      Serial.println(separator);
      Serial.println();
      return -1;
    }
  }

  // Детальная обработка ответа
  Serial.println("\n[RESULT ANALYSIS]");
  unsigned long totalTime = millis() - requestStartTime;
  Serial.printf("  Total Request Time: %lu ms\n", totalTime);
  
  if (response.length() > 0) {
    // Проверяем успешность обработки
    if (httpCode == 200 && response.indexOf("\"processed\":true") > 0) {
      Serial.println("  Status: ✅ SUCCESS - Нажатие обработано успешно!");
    } else if (httpCode == 200 && response.indexOf("\"processed\":false") > 0) {
      Serial.println("  Status: ⚠️  WARNING - Нажатие получено, но не обработано");
      Serial.println("          (вопрос не активен или уже ответили)");
    } else if (httpCode == 400) {
      Serial.println("  Status: ❌ ERROR - Bad Request");
      Serial.println("          Кнопка не найдена или не привязана к команде");
    } else if (httpCode == 401) {
      Serial.println("  Status: ❌ ERROR - Unauthorized");
      Serial.println("          Проблема с аутентификацией");
    } else if (httpCode == 404) {
      Serial.println("  Status: ❌ ERROR - Not Found");
      Serial.println("          Endpoint не найден");
    } else if (httpCode == 500) {
      Serial.println("  Status: ❌ ERROR - Internal Server Error");
      Serial.println("          Ошибка на сервере");
    } else if (httpCode < 0) {
      Serial.printf("  Status: ❌ ERROR - Network error (code: %d)\n", httpCode);
      Serial.println("          Возможные причины: нет подключения, таймаут");
    } else {
      Serial.printf("  Status: ⚠️  UNKNOWN - HTTP %d\n", httpCode);
    }
  } else {
    Serial.println("  Status: ⚠️  WARNING - Пустой ответ от сервера");
  }

  Serial.println(separator);
  Serial.println("[HTTP REQUEST] ========== Конец запроса ==========");
  Serial.println();

  return httpCode;
}

void setupButtonIfAny() {
#if BTN_PIN >= 0
  pinMode(BTN_PIN, INPUT_PULLUP);
  currentButtonState = lastButtonState = digitalRead(BTN_PIN);
#endif
}

void setup() {
  Serial.begin(115200);
  delay(500);

  Serial.println("\n=== ESP32-C3 Quiz Button ===");
  Serial.printf("Server URL: %s\n", SERVER_URL);
  Serial.printf("Button ID: %s\n", BUTTON_ID);
  Serial.printf("MAC Address: %s\n", WiFi.macAddress().c_str());
  Serial.printf("Wi-Fi SSID: %s\n", WIFI_SSID);
  Serial.printf("Wi-Fi режим: %s\n", strlen(WIFI_PASS) ? "WPA/WPA2" : "OPEN (без пароля)");
  
#if BTN_PIN >= 0
  Serial.printf("Button pin: %d (INPUT_PULLUP)\n", BTN_PIN);
  Serial.printf("Начальное состояние кнопки: %s\n", 
                currentButtonState == LOW ? "LOW (нажата)" : "HIGH (отпущена)");
#else
  Serial.println("Button: нет (только Serial 's')");
#endif

  if (AUTO_SEND_INTERVAL_MS > 0) {
    Serial.printf("Auto send: каждые %lu мс\n", (unsigned long)AUTO_SEND_INTERVAL_MS);
  } else {
    Serial.println("Auto send: выключено");
  }

  Serial.println("\nПодключаюсь к Wi-Fi...");
  connectWiFiBlocking();
  setupButtonIfAny();
  
  Serial.println("\n=== Готово ===");
  Serial.println("Команды:");
  Serial.println("  - Нажмите кнопку для отправки нажатия");
  Serial.println("  - 's' в Serial - отправить вручную");
  Serial.println("  - 'r' в Serial - переподключить Wi-Fi");
  Serial.println("========================\n");
}

void loop() {
  // Автоматическая отправка (если включена)
  if (AUTO_SEND_INTERVAL_MS > 0) {
    unsigned long now = millis();
    if (now - lastAutoSend >= AUTO_SEND_INTERVAL_MS) {
      lastAutoSend = now;
      Serial.println("[AUTO] Автоматическая отправка...");
      sendButtonPress();
    }
  }

  // Обработка команд из Serial
  if (Serial.available()) {
    int c = Serial.read();
    if (c == 's' || c == 'S') {
      Serial.println("[SERIAL] Ручная отправка...");
      sendButtonPress();
    } else if (c == 'r' || c == 'R') {
      Serial.println("[SERIAL] Переподключаю Wi-Fi...");
      WiFi.disconnect();
      connectWiFiBlocking();
    } else if (c == 'm' || c == 'M') {
      Serial.printf("[INFO] MAC Address: %s\n", WiFi.macAddress().c_str());
      Serial.printf("[INFO] IP Address: %s\n", WiFi.localIP().toString().c_str());
      Serial.printf("[INFO] Wi-Fi Status: %s\n", 
                    WiFi.status() == WL_CONNECTED ? "Connected" : "Disconnected");
    }
  }

  // Обработка физической кнопки (логика из рабочего примера)
#if BTN_PIN >= 0
  int reading = digitalRead(BTN_PIN);
  unsigned long now = millis();
  
  // Если состояние изменилось, запускаем таймер debounce
  if (reading != currentButtonState) {
    lastDebounceTime = now;
  }
  
  // Если состояние стабильно достаточно долго
  if ((now - lastDebounceTime) > DEBOUNCE_MS) {
    // Если произошло реальное изменение
    if (reading != lastButtonState) {
      // Сохраняем новое состояние
      lastButtonState = reading;
      
      // Если произошло нажатие (LOW, так как кнопка подключена к земле)
      if (reading == LOW) {
        Serial.println("\n[BUTTON STATE] 🔴 GPIO" + String(BTN_PIN) + " = LOW (нажата)");
        Serial.println("[BUTTON] Кнопка нажата! -> отправка HTTP запроса на сервер...");
        sendButtonPress();
      } else {
        Serial.println("[BUTTON STATE] 🟢 GPIO" + String(BTN_PIN) + " = HIGH (отпущена)");
        Serial.println("[BUTTON] Кнопка отпущена");
      }
    }
  }
  
  // Обновляем текущее состояние для следующей итерации
  currentButtonState = reading;
#endif

  delay(10);  // Небольшая задержка для стабильности
}
