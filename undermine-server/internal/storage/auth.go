package storage

import (
	"fmt"
	"time"
)

type User struct {
	ID           [16]byte
	Username     string
	Email        string
	PasswordHash string
	CreatedAt    int64
}

type Session struct {
	ID        [16]byte
	UserID    [16]byte
	TokenHash string
	ExpiresAt int64
}

type Login struct {
	IPAddress    string
	FailedCount  int
	BlockedUntil int64
}

func (s *SQLiteStorage) CreateUserWithKey(user *User, keyID [16]byte, pubKey [32]byte) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	queryUser := `INSERT INTO users (id, username, email, password_hash, created_at, updated_at) 
	              VALUES (?, ?, ?, ?, ?, ?)`

	_, err = tx.Exec(queryUser, user.ID[:], user.Username, user.Email, user.PasswordHash, user.CreatedAt, user.CreatedAt)
	if err != nil {
		return err
	}

	queryKey := `INSERT INTO user_keys (key_id, user_id, public_key, is_active, created_at) 
	             VALUES (?, ?, ?, 1, ?)`

	_, err = tx.Exec(queryKey, keyID[:], user.ID[:], pubKey[:], user.CreatedAt)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (s *SQLiteStorage) GetUserByUsername(username string) (*User, error) {
	query := "SELECT id, username, email, password_hash, created_at FROM users WHERE username = ?"

	row := s.db.QueryRow(query, username)

	var user User
	var idBytes []byte

	err := row.Scan(&idBytes, &user.Username, &user.Email, &user.PasswordHash, &user.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("пользователь не найден или ошибка: %w", err)
	}

	copy(user.ID[:], idBytes)
	return &user, nil
}

func (s *SQLiteStorage) CreateSession(session *Session, deviceInfo string, ipAddress string) error {

	qwery := "INSERT INTO sessions (id, user_id, token_hash, device_info, ip_address, created_at, expires_at) VALUES (?,?,?,?,?,?,?)"

	_, err := s.db.Exec(qwery, session.ID[:], session.UserID[:], session.TokenHash, deviceInfo, ipAddress, time.Now().Unix(), session.ExpiresAt)
	if err != nil {
		return fmt.Errorf("ошибка при создании сессии: %w", err)
	}
	return nil
}

func (s *SQLiteStorage) GetSessionByTokenHash(tokenHash string) (*Session, error) {
	query := "SELECT id, user_id, token_hash, expires_at FROM sessions WHERE token_hash = ? AND expires_at > ?"

	row := s.db.QueryRow(query, tokenHash, time.Now().Unix())

	var session Session
	var idBytes, userIDBytes []byte

	err := row.Scan(&idBytes, &userIDBytes, &session.TokenHash, &session.ExpiresAt)
	if err != nil {
		return nil, fmt.Errorf("сессия не найдена или истекла: %w", err)
	}

	copy(session.ID[:], idBytes)
	copy(session.UserID[:], userIDBytes)

	return &session, nil
}

func (s *SQLiteStorage) DeleteSession(tokenHash string) error {
	qwery := "DELETE FROM sessions WHERE token_hash = ?"

	_, err := s.db.Exec(qwery, tokenHash)
	if err != nil {
		return fmt.Errorf("ошибка при удалении сессии: %w", err)
	}
	return nil
}

func (s *SQLiteStorage) GetLoginAttempt(ipAddress string) (*Login, error) {
	query := "SELECT ip_address, failed_count, blocked_until FROM login_attempts WHERE ip_address = ?"
	row := s.db.QueryRow(query, ipAddress)

	var login Login
	err := row.Scan(&login.IPAddress, &login.FailedCount, &login.BlockedUntil)
	if err != nil {
		return nil, fmt.Errorf("попытка входа не найдена: %w", err)
	}
	return &login, nil
}

func (s *SQLiteStorage) RecordFailedLogin(ip string, blockedUntil int64) error {
	query := "INSERT INTO login_attempts (ip_address, failed_count, last_failed_at, blocked_until) VALUES (?,1,?,?) ON CONFLICT (ip_address) DO UPDATE SET failed_count = failed_count +1, last_failed_at = excluded.last_failed_at, blocked_until = excluded.blocked_until"
	_, err := s.db.Exec(query, ip, time.Now().Unix(), blockedUntil)
	if err != nil {
		return fmt.Errorf("ошибка записи неудачной попытки входа: %w", err)
	}
	return nil
}

func (s *SQLiteStorage) ResetLoginAttempts(ip string) error {
	query := "DELETE FROM login_attempts WHERE ip_address = ?"
	_, err := s.db.Exec(query, ip)
	if err != nil {
		return fmt.Errorf("ошибка сброса попыток входа: %w", err)
	}
	return nil
}

func (s *SQLiteStorage) GetActivePublicKey(userID [16]byte) ([32]byte, error) {
	query := "SELECT public_key FROM user_keys WHERE user_id = ? AND is_active = 1 LIMIT 1"
	row := s.db.QueryRow(query, userID[:])

	var keyBytes []byte
	if err := row.Scan(&keyBytes); err != nil {
		return [32]byte{}, fmt.Errorf("активный публичный ключ не найден: %w", err)
	}

	if len(keyBytes) != 32 {
		return [32]byte{}, fmt.Errorf("некорректная длина публичного ключа в БД: ожидается 32 байта")
	}

	var pubKey [32]byte
	copy(pubKey[:], keyBytes)

	return pubKey, nil
}
