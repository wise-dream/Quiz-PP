# ESP32-C3 Quiz Button

Физическая кнопка для отправки ответов в системе квизов PowerPoint Quiz.

## Настройка

### 1. Настройка в `platformio.ini`

Отредактируйте параметры в секции `[env:quiz_button_prod]`:

```ini
-D WIFI_SSID=\"YourWiFiName\"      # Имя Wi-Fi сети
-D WIFI_PASS=\"YourPassword\"      # Пароль (пусто для открытой сети)
-D SERVER_URL=\"https://server.com\" # URL сервера квиза
-D BUTTON_ID=\"1\"                  # ID кнопки (если несколько на устройстве)
-D BTN_PIN=9                        # GPIO пин для кнопки (GPIO9 на ESP32-C3)
```

### 2. Компиляция и загрузка

```bash
# Установка зависимостей (если нужно)
pio pkg install

# Компиляция и загрузка
pio run -e quiz_button_prod -t upload

# Мониторинг Serial
pio device monitor
```

## Использование

### Перед использованием

1. **Зарегистрируйте кнопку в админ-панели:**
   - Откройте админ-панель квиза
   - Перейдите в раздел "Физические кнопки"
   - Нажмите "+" и введите MAC адрес кнопки (выводится в Serial при запуске)
   - Введите название кнопки (опционально)

2. **Привяжите кнопку к команде:**
   - В списке кнопок выберите команду из выпадающего списка
   - Кнопка будет привязана к выбранной команде

### Работа кнопки

- **При нажатии физической кнопки** - отправляется POST запрос на сервер
- **Защита от дребезга** - минимальный интервал 500мс между нажатиями
- **Автоматическое определение MAC адреса** - из Wi-Fi модуля ESP32

### Команды Serial

- `s` - Отправить нажатие вручную
- `r` - Переподключиться к Wi-Fi
- `m` - Показать информацию (MAC, IP, статус Wi-Fi)

## Формат запроса

Кнопка отправляет на сервер:

```json
{
  "macAddress": "AA:BB:CC:DD:EE:FF",
  "buttonId": "1"
}
```

Endpoint: `POST /api/button/press`

## Ответ сервера

**Успех:**
```json
{
  "success": true,
  "message": "Button press processed successfully",
  "processed": true
}
```

**Не обработано (вопрос не активен):**
```json
{
  "success": true,
  "message": "question not active in room ABCD",
  "processed": false
}
```

## Структура проекта

```
button/
├── platformio.ini      # Конфигурация PlatformIO
├── src/
│   └── main.cpp        # Основной код кнопки
└── README.md           # Эта документация
```

## Требования

- PlatformIO IDE или CLI
- ESP32-C3 DevKit
- Физическая кнопка для GPIO9 (или другой пин, указанный в конфиге)

## Troubleshooting

**Кнопка не отправляет запросы:**
- Проверьте подключение к Wi-Fi
- Убедитесь что MAC адрес зарегистрирован в админ-панели
- Проверьте URL сервера в `platformio.ini`

**Ошибка SSL/TLS:**
- Установите `USE_TLS_INSECURE=1` для самоподписанных сертификатов (только для теста!)

**Кнопка не реагирует:**
- Проверьте подключение кнопки к правильному GPIO пину
- Проверьте логи в Serial мониторе

## Пример логов

```
=== ESP32-C3 Quiz Button ===
Server URL: https://quiz.wise-dream.ru
Button ID: 1
MAC Address: AA:BB:CC:DD:EE:FF
Wi-Fi SSID: WiseDream
Wi-Fi режим: OPEN (без пароля)
Button pin: 9 (INPUT_PULLUP)

Подключаюсь к Wi-Fi...
Wi-Fi OK: IP=192.168.1.100 RSSI=-45 dBm

=== Готово ===
Команды:
  - Нажмите кнопку для отправки нажатия
  - 's' в Serial - отправить вручную
  - 'r' в Serial - переподключить Wi-Fi
========================

[BUTTON] 🔴 Нажатие кнопки -> отправка на сервер...
[SEND] MAC: AA:BB:CC:DD:EE:FF, ButtonID: 1
[SEND] Endpoint: https://quiz.wise-dream.ru/api/button/press
[RESPONSE] HTTP Code: 200
[RESPONSE] Body: {"success":true,"message":"Button press processed successfully","processed":true}
[SUCCESS] ✅ Нажатие обработано успешно!
```

