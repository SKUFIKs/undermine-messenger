package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
	"undermine.ru/undermine-server/internal/storage"
)

type Storage interface {
	CreateUserWithKey(user *storage.User, keyID [16]byte, pubKey [32]byte) error
	GetUserByUsername(username string) (*storage.User, error)
	GetActivePublicKey(userID [16]byte) ([32]byte, error)
	CreateSession(session *storage.Session, deviceInfo string, ipAddress string) error
	GetSessionByTokenHash(tokenHash string) (*storage.Session, error)
	DeleteSession(tokenHash string) error
	GetLoginAttempt(ipAddress string) (*storage.Login, error)
	RecordFailedLogin(ip string, blockedUntil int64) error
	ResetLoginAttempts(ip string) error
}

type AuthService struct {
	store Storage
}

func NewAuthService(store Storage) *AuthService {
	return &AuthService{store: store}
}

func (s *AuthService) Register(username, email, password string, pubKey [32]byte, ip, deviceInfo string) (string, *storage.User, error) {
	if len(username) < 3 || len(password) < 6 {
		return "", nil, fmt.Errorf("Некорректный логин или пароль")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", nil, err
	}

	var userID, keyID [16]byte
	rand.Read(userID[:])
	rand.Read(keyID[:])

	user := &storage.User{
		ID:           userID,
		Username:     username,
		Email:        email,
		PasswordHash: string(hash),
		CreatedAt:    time.Now().Unix(),
	}

	err = s.store.CreateUserWithKey(user, keyID, pubKey)
	if err != nil {
		return "", nil, err
	}
	token, err := s.createSessionForUser(userID, ip, deviceInfo)
	return token, user, nil
}

func (s *AuthService) Login(username, password, ip, deviceInfo string) (string, *storage.User, error) {
	now := time.Now().Unix()

	attempt, err := s.store.GetLoginAttempt(ip)

	if err == nil && attempt != nil {
		if attempt.BlockedUntil > now {
			remainder := attempt.BlockedUntil - now
			return "", nil, fmt.Errorf("IP заблокирован. Подождите %d сек", remainder)
		}
	}

	user, err := s.store.GetUserByUsername(username)

	hashToCompare := "$2a$10$7EqJtq98hPqEX7fNZaFWoO.8/M8Y.Z8k/eZ5e.VzYy.X5w.w..."

	if user != nil {
		hashToCompare = user.PasswordHash
	}
	errPass := bcrypt.CompareHashAndPassword([]byte(hashToCompare), []byte(password))

	if errPass != nil || user == nil {
		failedCount := 1

		if attempt != nil {
			failedCount = attempt.FailedCount + 1
		}

		blockedUntil := int64(0)
		if failedCount == 5 {
			blockedUntil = now + 15*60
		} else if failedCount == 6 {
			blockedUntil = now + 60*60
		} else if failedCount == 7 {
			blockedUntil = now + 24*60*60
		} else if failedCount >= 8 {
			blockedUntil = now + 14*24*60*60
		}

		s.store.RecordFailedLogin(ip, blockedUntil)

		return "", nil, fmt.Errorf("неверный логин или пароль")
	}

	s.store.ResetLoginAttempts(ip)
	token, err := s.createSessionForUser(user.ID, ip, deviceInfo)

	return token, user, err
}

func (s *AuthService) createSessionForUser(userID [16]byte, ip, deviceInfo string) (string, error) {
	var rawToken [32]byte
	if _, err := rand.Read(rawToken[:]); err != nil {
		return "", fmt.Errorf("ошибка генерации токена: %w", err)
	}

	tokenHex := hex.EncodeToString(rawToken[:])

	tokenHash := sha256.Sum256([]byte(tokenHex))
	tokenHashHex := hex.EncodeToString(tokenHash[:])

	var sessionID [16]byte
	if _, err := rand.Read(sessionID[:]); err != nil {
		return "", err
	}

	session := &storage.Session{
		ID:        sessionID,
		UserID:    userID,
		TokenHash: tokenHashHex,
		ExpiresAt: time.Now().Add(30 * 24 * time.Hour).Unix(), // 30 дней
	}

	if err := s.store.CreateSession(session, deviceInfo, ip); err != nil {
		return "", err
	}

	return tokenHex, nil
}

func (s *AuthService) ValidateToken(rawToken string) ([16]byte, error) {
	tokenHash := sha256.Sum256([]byte(rawToken))
	tokenHashHex := hex.EncodeToString(tokenHash[:])

	session, err := s.store.GetSessionByTokenHash(tokenHashHex)
	if err != nil {
		return [16]byte{}, fmt.Errorf("недействительная сессия или срок действия истек: %w", err)
	}

	return session.UserID, nil
}

func (s *AuthService) SearchUser(username string) (*storage.User, [32]byte, error) {
	user, err := s.store.GetUserByUsername(username)
	if err != nil {
		return nil, [32]byte{}, fmt.Errorf("пользователь @%s не найден", username)
	}

	pubKey, err := s.store.GetActivePublicKey(user.ID)
	if err != nil {
		return nil, [32]byte{}, fmt.Errorf("у пользователя нет активного публичного ключа: %w", err)
	}

	return user, pubKey, nil
}

func (s *AuthService) UpdatePublicKey(userID [16]byte, pubKey [32]byte) error {
	return s.store.UpdatePublicKey(userID, pubKey)
}
