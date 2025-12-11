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
#define SERVER_URL "https://wise-dream.ru"
#endif
#ifndef BUTTON_ID
#define BUTTON_ID "1"
#endif
#ifndef AUTO_SEND_INTERVAL_MS
#define AUTO_SEND_INTERVAL_MS 0ul  // Выключено по умолчанию (только кнопка)
#endif
#ifndef BTN_PIN
#define BTN_PIN 5  // GPIO5 на ESP32-C3 (можно изменить)
#endif
#ifndef USE_TLS_INSECURE
#define USE_TLS_INSECURE 1  // Для самоподписанных сертификатов
#endif

#ifdef AUTH_BEARER
#define HAS_AUTH_BEARER 0
#endif
#ifdef X_API_KEY
#define HAS_X_API_KEY 0
#endif

const unsigned long DEBOUNCE_MS = 50;  // Защита от дребезга (мс) - как в рабочем примере
unsigned long lastAutoSend = 0;
int lastButtonState = HIGH;  // Изменено с lastBtn для совместимости с рабочей логикой
int currentButtonState = HIGH;  // Добавлено для правильной обработки debounce
unsigned long lastDebounceTime = 0;  // Изменено с lastBtnChange для совместимости
unsigned long lastPressTime = 0;
const unsigned long PRESS_COOLDOWN_MS = 500;  // Минимальный интервал между нажатиями

// Глобальные объекты для переиспользования соединения (keep-alive)
WiFiClientSecure* secureClient = nullptr;
WiFiClient* plainClient = nullptr;
HTTPClient* httpClient = nullptr;
bool connectionInitialized = false;
String endpointUrl = "";

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
  
  // Отключаем сон Wi-Fi для минимальной задержки
  WiFi.setSleep(false);
  Serial.println("Wi-Fi: режим сна отключен (WIFI_PS_NONE)");
  
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
  // УБРАЛИ Connection: close - используем keep-alive для переиспользования соединения
#ifdef HAS_AUTH_BEARER
  http.addHeader("Authorization", String("Bearer ") + AUTH_BEARER);
#endif
#ifdef HAS_X_API_KEY
  http.addHeader("X-API-Key", X_API_KEY);
#endif
}

// Инициализация HTTP-соединения один раз (keep-alive)
bool initializeHttpConnection() {
  if (connectionInitialized && httpClient != nullptr) {
    return true;  // Уже инициализировано
  }

  // Освобождаем старые объекты, если есть
  if (httpClient) {
    httpClient->end();
    delete httpClient;
    httpClient = nullptr;
  }
  if (secureClient) {
    delete secureClient;
    secureClient = nullptr;
  }
  if (plainClient) {
    delete plainClient;
    plainClient = nullptr;
  }

  if (!ensureWiFi()) {
    Serial.println("[ERROR] ❌ Нет Wi-Fi подключения для инициализации соединения.");
    return false;
  }

  endpointUrl = String(SERVER_URL) + "/quiz/api/button/press";
  bool isHttps = endpointUrl.startsWith("https://");

  httpClient = new HTTPClient();
  httpClient->setTimeout(10000);
  httpClient->setReuse(true);  // Включаем переиспользование соединения

  bool beginSuccess = false;
  if (isHttps) {
    secureClient = new WiFiClientSecure();
#if USE_TLS_INSECURE
    secureClient->setInsecure();
#endif
    secureClient->setNoDelay(true);  // Отключаем Nagle для минимальной задержки
    beginSuccess = httpClient->begin(*secureClient, endpointUrl);
    Serial.println("[HTTP] Инициализация HTTPS соединения с keep-alive...");
  } else {
    plainClient = new WiFiClient();
    plainClient->setNoDelay(true);  // Отключаем Nagle для минимальной задержки
    beginSuccess = httpClient->begin(*plainClient, endpointUrl);
    Serial.println("[HTTP] Инициализация HTTP соединения с keep-alive...");
  }

  if (!beginSuccess) {
    Serial.println("[ERROR] ❌ HTTP begin() failed при инициализации.");
    delete httpClient;
    httpClient = nullptr;
    if (secureClient) {
      delete secureClient;
      secureClient = nullptr;
    }
    if (plainClient) {
      delete plainClient;
      plainClient = nullptr;
    }
    return false;
  }

  // Устанавливаем заголовки один раз
  addCommonHeaders(*httpClient);
  httpClient->addHeader("Content-Type", "application/json");

  connectionInitialized = true;
  Serial.println("[HTTP] ✅ Соединение инициализировано с keep-alive. TLS handshake выполнен один раз.");
  return true;
}

// Проверка и переподключение при необходимости
bool ensureHttpConnection() {
  if (!connectionInitialized) {
    return initializeHttpConnection();
  }

  // Проверяем, что соединение всё ещё живое
  if (httpClient == nullptr) {
    connectionInitialized = false;
    return initializeHttpConnection();
  }

  return true;
}

// Отправка нажатия кнопки на сервер (с переиспользованием соединения)
int sendButtonPress() {
  unsigned long requestStartTime = millis();
  String separator = String("============================================================"); // 60 символов
  
  Serial.println();
  Serial.println(separator);
  Serial.println("[HTTP REQUEST] ========== Начало запроса ==========");
  
  // Проверка кулдауна между нажатиями
  unsigned long now = millis();
  if (lastPressTime > 0 && (now - lastPressTime) < PRESS_COOLDOWN_MS) {
    Serial.printf("[SKIP] ⏸️  Слишком быстрое нажатие (cooldown: %lu мс), пропускаю.\n", PRESS_COOLDOWN_MS);
    Serial.println(separator);
    Serial.println();
    return -2;
  }
  lastPressTime = now;

  // Обеспечиваем наличие переиспользуемого соединения
  if (!ensureHttpConnection()) {
    Serial.println("[ERROR] ❌ Не удалось инициализировать/поддерживать HTTP соединение.");
    Serial.println(separator);
    Serial.println();
    return -1;
  }

  String macAddress = getMacAddress();
  
  // Формируем JSON payload согласно API бэкенда
  String payload = String("{\"macAddress\":\"") + macAddress +
                   "\",\"buttonId\":\"" + String(BUTTON_ID) + "\"}";

  // Подробное логирование запроса
  Serial.println("[REQUEST INFO]");
  Serial.printf("  Method: POST\n");
  Serial.printf("  URL: %s\n", endpointUrl.c_str());
  Serial.printf("  Protocol: %s\n", endpointUrl.startsWith("https://") ? "HTTPS" : "HTTP");
  Serial.printf("  Connection: keep-alive (переиспользуется)\n");
  Serial.printf("  MAC Address: %s\n", macAddress.c_str());
  Serial.printf("  Button ID: %s\n", BUTTON_ID);
  Serial.printf("  Timestamp: %lu ms\n", now);
  
  Serial.println("\n[REQUEST HEADERS]");
  Serial.println("  User-Agent: ESP32C3-Button/1.0");
  Serial.println("  Connection: keep-alive");
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

  Serial.println("\n[NETWORK] Отправка запроса через переиспользуемое соединение...");
  
  // Используем уже инициализированное соединение - НЕ создаём новое!
  unsigned long sendStart = millis();
  httpCode = httpClient->POST(payload);
  responseTime = millis() - sendStart;
  response = httpClient->getString();
  
  // НЕ вызываем httpClient->end() - соединение остаётся открытым для следующего запроса!
  
  Serial.println("\n[RESPONSE]");
  Serial.printf("  HTTP Status Code: %d\n", httpCode);
  Serial.printf("  Response Time: %lu ms (только POST, без TLS handshake)\n", responseTime);
  Serial.printf("  Response Size: %d bytes\n", response.length());
  
  // Логируем заголовки ответа, если доступны
  int headerCount = httpClient->headers();
  if (headerCount > 0) {
    Serial.println("\n[RESPONSE HEADERS]");
    for (int i = 0; i < headerCount; i++) {
      String headerName = httpClient->headerName(i);
      String headerValue = httpClient->header(i);
      Serial.printf("  %s: %s\n", headerName.c_str(), headerValue.c_str());
    }
  }
  
  Serial.println("\n[RESPONSE BODY]");
  if (response.length() > 0) {
    Serial.printf("  %s\n", response.c_str());
  } else {
    Serial.println("  (пусто)");
  }

  // Если получили ошибку сети - сбрасываем соединение для переподключения
  if (httpCode < 0) {
    Serial.println("[WARNING] ⚠️  Обнаружена ошибка сети, переподключаю соединение...");
    connectionInitialized = false;
    if (httpClient) {
      httpClient->end();
      delete httpClient;
      httpClient = nullptr;
    }
    if (secureClient) {
      delete secureClient;
      secureClient = nullptr;
    }
    if (plainClient) {
      delete plainClient;
      plainClient = nullptr;
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
      Serial.println("          Соединение будет переподключено при следующем запросе");
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
  
  // Инициализируем HTTP-соединение один раз с keep-alive
  Serial.println("\nИнициализация HTTP-соединения с keep-alive...");
  if (initializeHttpConnection()) {
    Serial.println("✅ HTTP-соединение готово. TLS handshake выполнен.");
  } else {
    Serial.println("⚠️  Предупреждение: не удалось инициализировать соединение сейчас.");
    Serial.println("    Оно будет создано при первом нажатии кнопки.");
  }
  
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
      // Переинициализируем HTTP-соединение после переподключения Wi-Fi
      connectionInitialized = false;
      if (initializeHttpConnection()) {
        Serial.println("[SERIAL] ✅ HTTP-соединение переинициализировано.");
      } else {
        Serial.println("[SERIAL] ⚠️  Не удалось переинициализировать HTTP-соединение.");
      }
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
