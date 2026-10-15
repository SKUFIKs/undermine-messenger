# UnderMine Server

Бэкенд-сервис аутентификации, каталога открытых ключей, групповых чатов и ретрансляции сообщений в режиме реального времени для криптографического мессенджера UnderMine.

Сервер спроектирован по архитектуре **Zero-Knowledge Relay**: бэкенд не хранит закрытые ключи пользователей, не хранит открытые ключи групп и не имеет технической возможности расшифровать полезную нагрузку сообщений. Его задача — верификация сессий, предоставление актуальных открытых ключей собеседников, хранение зашифрованных ключей групп и надёжная маршрутизация зашифрованных пакетов между пирами (включая доставку офлайн-сообщений).

---

## Технологии

- **Go 1.21+**
- **gorilla/websocket** — WebSocket Gateway
- **mattn/go-sqlite3** — хранилище SQLite (режим WAL)
- **golang.org/x/crypto** — криптографические примитивы

---

## Архитектура

```
undermine-server/
├── cmd/server/main.go              # Точка входа (Composition Root), .env, CORS, маршруты
├── internal/
│   ├── api/                        # HTTP Transport Layer (REST API)
│   │   ├── auth_handler.go         # Обработчики аутентификации и пользователей
│   │   └── group_handler.go        # Обработчики групповых чатов
│   ├── auth/                       # Бизнес-логика (Service Layer)
│   │   └── service.go              # Сервис аутентификации, сессий и групп
│   ├── protocol/                   # Бинарный транспортный протокол
│   │   └── packet.go               # Структуры пакетов (личные + групповые)
│   ├── storage/                    # Data Access Layer (SQLite)
│   │   ├── sqlite.go               # Инициализация БД
│   │   ├── auth.go                 # Запросы пользователей и сессий
│   │   └── group.go                # Запросы групп и ключей
│   └── transport/                  # WebSocket Relay Layer
│       ├── client.go               # Модель активного сокета
│       ├── hub.go                  # Менеджер подключений и маршрутизация
│       └── hub_test.go             # Тест гонки регистрации соединений
├── .env / .env.example             # Конфигурация (PORT, HOST)
├── go.mod
└── go.sum
```

---

## Конфигурация

Все параметры задаются через `.env` файл (загружается автоматически; значения по умолчанию — в скобках):

| Переменная | По умолчанию | Описание |
|------------|--------------|----------|
| `PORT` | `8080` | Порт HTTP/WebSocket сервера |
| `HOST` | `0.0.0.0` | Адрес прослушивания |

```bash
cp .env.example .env
```

---

## Схема данных (SQLite)

База данных работает в режиме **WAL** (Write-Ahead Logging) для высокой конкурентности. Файл БД — `messenger.db` в рабочей директории.

### Таблицы

| Таблица | Описание |
|---------|----------|
| `users` | Пользователи (id, username, email, password_hash) |
| `user_keys` | Открытые ключи Curve25519 пользователей |
| `sessions` | Сессионные токены (SHA-256 хэш) |
| `login_attempts` | Защита от подбора паролей |
| `pending_messages` | Офлайн-очередь сообщений |
| `groups` | Групповые чаты (id, name, owner_id) |
| `group_members` | Участники групп (group_id, user_id) |
| `group_keys` | Зашифрованные ключи групп для каждого участника |

---

## REST API

Базовый URL: `http://localhost:8080`

### Аутентификация

#### 1. Регистрация

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

#### 2. Вход

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

#### 3. Обновление открытого ключа

```
POST /api/v1/auth/key
Authorization: Bearer <token>
Content-Type: application/json

{
  "public_key": "64_hex_symbols"
}
```

### Пользователи

#### 4. Поиск пользователя

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

#### 5. Каталог пользователей с ключами

```
GET /api/v1/users/directory
```

**Ответ 200 OK:**
```json
[
  { "user_id": "32_hex_symbols", "username": "bob", "public_key": "64_hex_symbols" }
]
```

### Групповые чаты

#### 6. Создание группы

```
POST /api/v1/groups
Authorization: Bearer <token>
Content-Type: application/json

{
  "name": "Команда проекта",
  "members": ["32_hex_symbols", "32_hex_symbols"]
}
```

**Ответ 201 Created:**
```json
{
  "id": "32_hex_symbols",
  "name": "Команда проекта"
}
```

#### 7. Список моих групп

```
GET /api/v1/groups
Authorization: Bearer <token>
```

**Ответ 200 OK:**
```json
[
  { "id": "32_hex_symbols", "name": "Команда проекта", "owner_id": "32_hex_symbols" }
]
```

#### 8. Список участников группы

```
GET /api/v1/groups/:id/members
Authorization: Bearer <token>
```

**Ответ 200 OK:**
```json
[
  { "user_id": "32_hex_symbols", "username": "bob", "public_key": "64_hex_symbols" }
]
```

#### 9. Добавление участника

```
POST /api/v1/groups/:id/members
Authorization: Bearer <token>
Content-Type: application/json

{
  "user_id": "32_hex_symbols"
}
```

#### 10. Удаление участника

```
DELETE /api/v1/groups/:id/members/:userId
Authorization: Bearer <token>
```

#### 11. Загрузка зашифрованного ключа группы для участника

```
POST /api/v1/groups/:id/keys/:userId
Authorization: Bearer <token>
Content-Type: application/json

{
  "encrypted_key": "hex"
}
```

#### 12. Получение своего зашифрованного ключа группы

```
GET /api/v1/groups/:id/keys
Authorization: Bearer <token>
```

**Ответ 200 OK:**
```json
{
  "encrypted_key": "hex"
}
```

#### 13. Удаление группы (владелец) / выход из группы (участник)

```
DELETE /api/v1/groups/:id
Authorization: Bearer <token>
```

---

## WebSocket Gateway

URL: `ws://localhost:8080/ws?token=<token>`

### Протокол обмена

| Тип пакета | Код | Описание |
|------------|-----|----------|
| Message | 0x01 | Зашифрованное личное сообщение |
| MessageAck | 0x02 | Подтверждение доставки |
| Ping | 0x03 | Heartbeat |
| Pong | 0x04 | Ответ на Ping |
| GroupMessage | 0x05 | Зашифрованное групповое сообщение |

### Формат пакетов

**Message (0x01):** `type(1) + msgId(16) + recipientId(16) + nonce(12) + payload`

**GroupMessage (0x05):** `type(1) + senderId(16) + groupId(16) + msgId(16) + nonce(12) + payload`

### Доставка сообщений

- **Личные сообщения:** если получатель онлайн — пакет маршрутизируется немедленно, отправителю возвращается ACK `0x02` (delivered). Если офлайн — пакет сохраняется в `pending_messages`, отправителю возвращается ACK `0x01` (sent). При следующем подключении получателя очередь автоматически доставляется.
- **Групповые сообщения:** пакет рассылается всем онлайн-участникам группы (кроме отправителя), офлайн-участникам сохраняется в офлайн-очередь.

---

## Запуск

Требуется **Go 1.21+**.

```bash
# Сборка
go build -o server ./cmd/server/main.go

# Запуск
./server
```

Или без сборки:

```bash
go run ./cmd/server
```

Сервер запустится на `0.0.0.0:8080` (или на адресе из `.env`).

### Проверка работоспособности

```bash
curl http://localhost:8080/api/v1/users/directory
```

---

## Безопасность

- Пароли хранятся только в виде хэшей (bcrypt).
- Сессионные токены хранятся на сервере в виде SHA-256 хэшей.
- Защита от подбора паролей — таблица `login_attempts` с ограничением частоты.
- CORS открыт (`Access-Control-Allow-Origin: *`) — для продакшена рекомендуется ограничить список доменов.
- Сервер **не видит** содержимого сообщений: ключи E2EE генерируются и хранятся только на клиентах.
- Сервер **не видит** открытых ключей групп: в `group_keys` хранятся только ключи, зашифрованные pairwise-ключами участников.
