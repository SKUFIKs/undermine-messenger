# UnderMine Messenger

Криптографический мессенджер с архитектурой **Zero-Knowledge Relay** и сквозным шифрованием **E2EE (Curve25519 ECDH + ChaCha20-Poly1305 AEAD)**.

## Структура проекта

- **`undermine-server/`** — Оригинальный бэкенд на **Go**:
  - `cmd/server/main.go` — Composition root и запуск сервера
  - `internal/api/` — REST API (`/api/v1/auth/register`, `/api/v1/auth/login`, `/api/v1/users/search`)
  - `internal/auth/` — Сервис аутентификации и сессий
  - `internal/storage/` — Хранилище SQLite (WAL режим, таблицы пользователей, ключей и офлайн-очереди)
  - `internal/transport/` — WebSocket Relay Hub и менеджер соединений
  - `internal/protocol/` — Бинарный протокол обмена пакетами

- **`undermine-client/`** — Современный веб-клиент на **React + TypeScript + Tailwind CSS**:
  - `src/crypto/e2ee.ts` — E2EE криптография (Curve25519 ECDH + ChaCha20-Poly1305)
  - `src/components/` — Интерфейс: чаты, аватары, пузыри сообщений, модальные окна
  - `server.ts` — Веб-шлюз и среда разработки на порту 3000

## Запуск

```bash
# Установка зависимостей
npm install

# Запуск клиента в режиме разработки (порт 3000)
npm run dev
```
