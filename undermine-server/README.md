# UnderMine Server

Бэкенд-сервис аутентификации, каталога открытых ключей и ретрансляции сообщений в режиме реального времени для криптографического мессенджера UnderMine.

Сервер спроектирован по архитектуре **Zero-Knowledge Relay**: бэкенд не хранит закрытые ключи пользователей и не имеет технической возможности расшифровать полезную нагрузку сообщений. Его задача — верификация сессий, предоставление актуальных открытых ключей собеседников и надёжная маршрутизация зашифрованных пакетов между пирами (включая доставку офлайн-сообщений).

---

## Архитектура

```
undermine-server/
├── cmd/server/main.go              # Точка входа (Composition Root)
├── internal/
│   ├── api/                        # HTTP Transport Layer (REST API)
│   │   └── auth_handler.go         # Обработчики HTTP
│   ├── auth/                       # Бизнес-логика (Service Layer)
│   │   └── service.go              # Сервис аутентификации
│   ├── protocol/                   # Бинарный транспортный протокол
│   │   └── packet.go               # Структуры пакетов
│   ├── storage/                    # Data Access Layer (SQLite)
│   │   ├── sqlite.go               # Инициализация БД
│   │   └── auth.go                 # Запросы к данным
│   └── transport/                  # WebSocket Relay Layer
│       ├── client.go               # Модель активного сокета
│       └── hub.go                  # Менеджер подключений
├── go.mod
├── go.sum
└── README.md
```

---

## Схема данных (SQLite)

База данных работает в режиме **WAL** (Write-Ahead Logging) для высокой конкурентности.

### Таблицы

| Таблица | Описание |
|---------|----------|
| `users` | Пользователи (id, username, email, password_hash) |
| `user_keys` | Открытые ключи Curve25519 пользователей |
| `sessions` | Сессионные токены (SHA-256 хэш) |
| `login_attempts` | Защита от подбора паролей |
| `pending_messages` | Офлайн-очередь сообщений |

---

## REST API

Базовый URL: `http://localhost:8080`

### 1. Регистрация

```
POST /api/v1/auth/register
Content-Type: application/json

{
  "username": "alice",
  "email": "alice@example.com",
  "password": "SecurePassword123",
  "public_key": "64_hex_symbols"
}
```

**Ответ 201 Created:**
```json
{
  "token": "64_hex_symbols",
  "user_id": "32_hex_symbols",
  "username": "alice"
}
```

### 2. Вход

```
POST /api/v1/auth/login
Content-Type: application/json

{
  "username": "alice",
  "password": "SecurePassword123"
}
```

**Ответ 200 OK:**
```json
{
  "token": "64_hex_symbols",
  "user_id": "32_hex_symbols",
  "username": "alice"
}
```

### 3. Поиск пользователя

```
GET /api/v1/users/search?username=bob
Authorization: Bearer <token>
```

**Ответ 200 OK:**
```json
{
  "user_id": "32_hex_symbols",
  "username": "bob",
  "public_key": "64_hex_symbols"
}
```

---

## WebSocket Gateway

URL: `ws://localhost:8080/ws?token=<token>`

### Протокол обмена

| Тип пакета | Код | Описание |
|------------|-----|----------|
| Message | 0x01 | Зашифрованное сообщение |
| MessageAck | 0x02 | Подтверждение доставки |
| Ping | 0x03 | Heartbeat |
| Pong | 0x04 | Ответ на Ping |

---

## Запуск

```bash
# Сборка
go build -o server ./cmd/server/main.go

# Запуск
./server
```

Сервер запустится на `0.0.0.0:8080`.
