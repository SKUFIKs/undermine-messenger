package storage

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

type SQLiteStorage struct {
	db *sql.DB
}

func New(dbPath string) (*SQLiteStorage, error) {

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("ошибка инициализации драйвера: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ошибка открытия файла БД: %w", err)
	}

	if _, err := db.Exec("PRAGMA journal_mode=WAL; PRAGMA synchronous=NORMAL;"); err != nil {
		return nil, fmt.Errorf("ошибка настройки PRAGMA WAL: %w", err)
	}

	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id         BLOB PRIMARY KEY,
		username   TEXT UNIQUE NOT NULL,
		email TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		created_at INTEGER NOT NULL,
		updated_at INTEGER NOT NULL
	);

	CREATE TABLE IF NOT EXISTS user_keys (
		key_id BLOB PRIMARY KEY,
		user_id BLOB NOT NULL REFERENCES user(id) ON DELETE CASCADE,
		public_key BLOB NOT NULL,
		is_active BLOB DEFAULT 1,
		created_at INTEGER NOT NULL
	);

	CREATE TABLE IF NOT EXISTS sessions (
		id BLOB PRIMARY KEY,
		user_id BLOB NOT NULL REFERENCES user(id) ON DELETE CASCADE,
		token_hash TEXT UNIQUE NOT NULL,
		device_info TEXT NULL,
		ip_address TEXT NOT NULL,
		created_at INTEGER NOT NULL,
		expires_at INTEGER NOT NULL
	);

	CREATE TABLE IF NOT EXISTS login_attempts (
		ip_address TEXT PRIMARY KEY,
		failed_count INTEGER DEFAULT 0,
		last_failed_at INTEGER NOT NULL,
		blocked_until INTEGER DEFAULT 0	
	);

	CREATE TABLE IF NOT EXISTS pending_messages (
		id           BLOB PRIMARY KEY,
		recipient_id BLOB NOT NULL,
		raw_payload  BLOB NOT NULL,
		created_at   INTEGER NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_pending_recipient 
	ON pending_messages (recipient_id, id ASC);
	`

	if _, err := db.Exec(schema); err != nil {
		return nil, fmt.Errorf("ошибка создания таблиц БД: %w", err)
	}

	return &SQLiteStorage{db: db}, nil

}

func (s *SQLiteStorage) SavePendingMessage(msgID [16]byte, recipientID [16]byte, rawPayload []byte) error {

	realTime := time.Now().Unix()

	schema := "INSERT INTO pending_messages (id, recipient_id, raw_payload, created_at) VALUES (?, ?, ?, ?) ON CONFLICT(id) DO NOTHING"

	_, err := s.db.Exec(schema, msgID[:], recipientID[:], rawPayload[:], realTime)

	if err != nil {
		return fmt.Errorf("Ошибка сохранения сообщения: %w", err)
	}

	return nil

}

func (s *SQLiteStorage) GetPendingMessages(recipientID [16]byte, limit int) ([][]byte, error) {
	query := `
		SELECT raw_payload 
		FROM pending_messages 
		WHERE recipient_id = ? 
		ORDER BY id ASC 
		LIMIT ?
	`

	// 1. Выполняем SELECT через s.db.Query
	rows, err := s.db.Query(query, recipientID[:], limit)
	if err != nil {
		return nil, fmt.Errorf("ошибка выполнения SELECT: %w", err)
	}
	defer rows.Close()

	var messages [][]byte

	for rows.Next() {
		var payload []byte
		if err := rows.Scan(&payload); err != nil {
			return nil, fmt.Errorf("ошибка чтения строки: %w", err)
		}
		messages = append(messages, payload)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ошибка курсора строк: %w", err)
	}

	return messages, nil
}

func (s *SQLiteStorage) DeletePendingMessage(msgID [16]byte) error {

	schema := "DELETE FROM pending_messages WHERE id = ?"

	_, err := s.db.Exec(schema, msgID[:])

	if err != nil {
		return fmt.Errorf("ошибка удаления сообщения: %w", err)
	}

	return nil
}
