package storage

import (
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"undermine.ru/undermine-server/internal/domain"
)

func (s *SQLiteStorage) CreateUser(id [16]byte, username, email, passwordHash string, publicKey [32]byte) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(
		"INSERT INTO users (id, username, email, password_hash, created_at) VALUES (?, ?, ?, ?, ?)",
		id[:], username, email, passwordHash, time.Now(),
	)
	if err != nil {
		return fmt.Errorf("ошибка создания пользователя: %w", err)
	}

	_, err = tx.Exec(
		"INSERT INTO user_keys (user_id, public_key, updated_at) VALUES (?, ?, ?)",
		id[:], publicKey[:], time.Now(),
	)
	if err != nil {
		return fmt.Errorf("ошибка сохранения открытого ключа: %w", err)
	}

	return tx.Commit()
}

func (s *SQLiteStorage) GetUserByUsername(username string) (*domain.User, error) {
	row := s.db.QueryRow(
		"SELECT id, username, email, password_hash, created_at FROM users WHERE username = ?",
		username,
	)

	var user domain.User
	var idBytes []byte
	err := row.Scan(&idBytes, &user.Username, &user.Email, &user.PasswordHash, &user.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	copy(user.ID[:], idBytes)
	return &user, nil
}

func (s *SQLiteStorage) GetPublicKey(userID [16]byte) ([32]byte, error) {
	row := s.db.QueryRow("SELECT public_key FROM user_keys WHERE user_id = ?", userID[:])
	var pubKeyBytes []byte
	var pubKey [32]byte

	err := row.Scan(&pubKeyBytes)
	if err != nil {
		return pubKey, err
	}
	copy(pubKey[:], pubKeyBytes)
	return pubKey, nil
}

func (s *SQLiteStorage) UpdatePublicKey(userID [16]byte, publicKey [32]byte) error {
	_, err := s.db.Exec(
		"INSERT OR REPLACE INTO user_keys (user_id, public_key, updated_at) VALUES (?, ?, ?)",
		userID[:], publicKey[:], time.Now(),
	)
	return err
}

func (s *SQLiteStorage) CreateSession(tokenHash [32]byte, userID [16]byte, deviceInfo string) error {
	_, err := s.db.Exec(
		"INSERT INTO sessions (token_hash, user_id, device_info, created_at) VALUES (?, ?, ?, ?)",
		tokenHash[:], userID[:], deviceInfo, time.Now(),
	)
	return err
}

func (s *SQLiteStorage) GetSessionUser(tokenHash [32]byte) ([16]byte, error) {
	row := s.db.QueryRow("SELECT user_id FROM sessions WHERE token_hash = ?", tokenHash[:])
	var idBytes []byte
	var userID [16]byte

	err := row.Scan(&idBytes)
	if err != nil {
		return userID, err
	}
	copy(userID[:], idBytes)
	return userID, nil
}

func (s *SQLiteStorage) SavePendingMessage(messageID [16]byte, recipientID [16]byte, payload []byte) error {
	_, err := s.db.Exec(
		"INSERT INTO pending_messages (message_id, recipient_id, payload, created_at) VALUES (?, ?, ?, ?)",
		messageID[:], recipientID[:], payload, time.Now(),
	)
	return err
}

func (s *SQLiteStorage) GetPendingMessages(recipientID [16]byte, limit int) ([][]byte, error) {
	rows, err := s.db.Query(
		"SELECT message_id, payload FROM pending_messages WHERE recipient_id = ? ORDER BY created_at ASC LIMIT ?",
		recipientID[:], limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result [][]byte
	var messageIDs [][]byte

	for rows.Next() {
		var msgID, payload []byte
		if err := rows.Scan(&msgID, &payload); err != nil {
			continue
		}
		result = append(result, payload)
		messageIDs = append(messageIDs, msgID)
	}

	for _, id := range messageIDs {
		_, _ = s.db.Exec("DELETE FROM pending_messages WHERE message_id = ?", id)
	}

	return result, nil
}

func HashToken(token string) [32]byte {
	return sha256.Sum256([]byte(token))
}
