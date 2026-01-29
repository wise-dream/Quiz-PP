#include <Arduino.h>
#include <WiFi.h>
#include <HTTPClient.h>
#include <WiFiClientSecure.h>

/* ===== НАСТРОЙКИ ===== */
const char* WIFI_SSID = "iPhone (Алексей)";
const char* WIFI_PASS = "alex0027";

#define SERVER_URL "https://quiz.wise-dream.ru/quiz/api/button/press"
#define BUTTON_ID  "1"

#define BTN_PIN 5          // GPIO кнопки
#define LED_PIN 8          // встроенный LED (если есть)

/* ===== ТАЙМИНГИ ===== */
const unsigned long DEBOUNCE_MS = 30;
const unsigned long COOLDOWN_MS = 300;

/* ===== СОСТОЯНИЯ ===== */
WiFiClientSecure client;
HTTPClient http;

bool httpReady = false;
bool pressQueued = false;

unsigned long lastDebounce = 0;
unsigned long lastSend = 0;
int lastBtnState = HIGH;

/* ===== WIFI ===== */
void connectWiFi() {
  Serial.printf("Connecting to WiFi: [%s]\n", WIFI_SSID);

  WiFi.mode(WIFI_STA);
  WiFi.setSleep(false);
  WiFi.begin(WIFI_SSID, WIFI_PASS);

  unsigned long t0 = millis();
  while (WiFi.status() != WL_CONNECTED) {
    delay(200);
    Serial.print(".");
    if (millis() - t0 > 15000) {
      Serial.println("\nWiFi timeout");
      return;
    }
  }

  Serial.printf("\nWiFi OK | IP=%s | RSSI=%d\n",
                WiFi.localIP().toString().c_str(),
                WiFi.RSSI());
}

/* ===== HTTP INIT ===== */
bool initHttp() {
  if (WiFi.status() != WL_CONNECTED) return false;

  client.setInsecure();      // TLS без проверки сертификата
  client.setNoDelay(true);

  http.setReuse(true);
  http.setTimeout(5000);

  if (!http.begin(client, SERVER_URL)) {
    Serial.println("HTTP begin failed");
    return false;
  }

  http.addHeader("Content-Type", "application/json");
  Serial.println("HTTP keep-alive ready");
  return true;
}

/* ===== SEND ===== */
void sendPress() {
  if (millis() - lastSend < COOLDOWN_MS) return;
  lastSend = millis();

  digitalWrite(LED_PIN, HIGH);

  String payload =
    String("{\"macAddress\":\"") +
    WiFi.macAddress() +
    "\",\"buttonId\":\"" + BUTTON_ID + "\"}";

  unsigned long t0 = millis();
  int code = http.POST(payload);
  unsigned long dt = millis() - t0;

  Serial.printf("POST -> %d (%lums)\n", code, dt);

  if (code < 0) {
    Serial.println("HTTP error, reopening");
    http.end();
    httpReady = initHttp();
  }

  digitalWrite(LED_PIN, LOW);
}

/* ===== SETUP ===== */
void setup() {
  Serial.begin(115200);
  delay(300);

  pinMode(BTN_PIN, INPUT_PULLUP);
  pinMode(LED_PIN, OUTPUT);
  digitalWrite(LED_PIN, LOW);

  Serial.println("\n=== ESP32-C3 QUIZ BUTTON ===");

  connectWiFi();
  httpReady = initHttp();

  if (!httpReady) {
    Serial.println("HTTP NOT READY");
  }
}

void loop() {
  int btn = digitalRead(BTN_PIN);
  unsigned long now = millis();

  if (btn != lastBtnState) {
    lastDebounce = now;
  }

  if ((now - lastDebounce) > DEBOUNCE_MS) {
    if (btn == LOW && lastBtnState == HIGH) {
      Serial.println("BTN pressed");
      pressQueued = true;
    }
  }

  lastBtnState = btn;

  if (pressQueued && httpReady) {
    pressQueued = false;
    sendPress();
  }
}
