package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"
	"undermine.ru/undermine-server/internal/domain"
	"undermine.ru/undermine-server/internal/storage"
)

type AuthService struct {
	storage *storage.SQLiteStorage
}

func NewAuthService(storage *storage.SQLiteStorage) *AuthService {
	return &AuthService{storage: storage}
}

func (s *AuthService) Register(username, email, password string, publicKey [32]byte, ip, deviceInfo string) (string, *domain.User, error) {
	existing, err := s.storage.GetUserByUsername(username)
	if err != nil {
		return "", nil, err
	}
	if existing != nil {
		return "", nil, errors.New("пользователь с таким именем уже существует")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", nil, fmt.Errorf("ошибка хеширования: %w", err)
	}

	var userID [16]byte
	if _, err := rand.Read(userID[:]); err != nil {
		return "", nil, err
	}

	err = s.storage.CreateUser(userID, username, email, string(hash), publicKey)
	if err != nil {
		return "", nil, err
	}

	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", nil, err
	}
	token := hex.EncodeToString(tokenBytes)
	tokenHash := storage.HashToken(token)

	err = s.storage.CreateSession(tokenHash, userID, deviceInfo)
	if err != nil {
		return "", nil, err
	}

	user := &domain.User{
		ID:       userID,
		Username: username,
		Email:    email,
	}

	return token, user, nil
}

func (s *AuthService) Login(username, password, ip, deviceInfo string) (string, *domain.User, error) {
	user, err := s.storage.GetUserByUsername(username)
	if err != nil {
		return "", nil, err
	}
	if user == nil {
		// Timing attack protection
		_ = bcrypt.CompareHashAndPassword([]byte("$2a$10$7EqJtq98hPqEX7fNZaFWoO"), []byte(password))
		return "", nil, errors.New("неверное имя пользователя или пароль")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", nil, errors.New("неверное имя пользователя или пароль")
	}

	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", nil, err
	}
	token := hex.EncodeToString(tokenBytes)
	tokenHash := storage.HashToken(token)

	err = s.storage.CreateSession(tokenHash, user.ID, deviceInfo)
	if err != nil {
		return "", nil, err
	}

	return token, user, nil
}

func (s *AuthService) ValidateToken(token string) ([16]byte, error) {
	tokenHash := storage.HashToken(token)
	userID, err := s.storage.GetSessionUser(tokenHash)
	if err != nil {
		return [16]byte{}, errors.New("сессия не найдена или истекла")
	}
	return userID, nil
}

func (s *AuthService) SearchUser(username string) (*domain.User, [32]byte, error) {
	user, err := s.storage.GetUserByUsername(username)
	if err != nil {
		return nil, [32]byte{}, err
	}
	if user == nil {
		return nil, [32]byte{}, errors.New("пользователь не найден")
	}

	pubKey, err := s.storage.GetPublicKey(user.ID)
	if err != nil {
		return nil, [32]byte{}, errors.New("ключ пользователя не найден")
	}

	return user, pubKey, nil
}

func (s *AuthService) UpdatePublicKey(userID [16]byte, publicKey [32]byte) error {
	return s.storage.UpdatePublicKey(userID, publicKey)
}

func (s *AuthService) Directory() ([]storage.DirectoryEntry, error) {
	return s.storage.ListUsersWithKeys()
}

func (s *AuthService) CreateGroup(name string, ownerID [16]byte) (*storage.GroupEntry, error) {
	var groupID [16]byte
	if _, err := rand.Read(groupID[:]); err != nil {
		return nil, err
	}

	err := s.storage.CreateGroup(groupID, name, ownerID)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания группы: %w", err)
	}

	// Owner is automatically a member
	err = s.storage.AddGroupMember(groupID, ownerID)
	if err != nil {
		return nil, fmt.Errorf("ошибка добавления владельца в группу: %w", err)
	}

	return &storage.GroupEntry{
		ID:      groupID,
		Name:    name,
		OwnerID: ownerID,
	}, nil
}

func (s *AuthService) ListGroups(userID [16]byte) ([]storage.GroupEntry, error) {
	return s.storage.GetGroupsByMember(userID)
}

func (s *AuthService) AddGroupMember(groupID, userID [16]byte) error {
	return s.storage.AddGroupMember(groupID, userID)
}

func (s *AuthService) RemoveGroupMember(groupID, userID [16]byte) error {
	return s.storage.RemoveGroupMember(groupID, userID)
}

func (s *AuthService) ListGroupMembers(groupID [16]byte) ([]storage.GroupMemberEntry, error) {
	return s.storage.GetGroupMembers(groupID)
}

func (s *AuthService) IsGroup(id [16]byte) (bool, error) {
	return s.storage.IsGroup(id)
}

func (s *AuthService) SaveGroupKey(groupID, userID [16]byte, encryptedKey []byte) error {
	return s.storage.SaveGroupKey(groupID, userID, encryptedKey)
}

func (s *AuthService) GetGroupKey(groupID, userID [16]byte) ([]byte, error) {
	return s.storage.GetGroupKey(groupID, userID)
}

func (s *AuthService) DeleteGroup(groupID [16]byte) error {
	return s.storage.DeleteGroup(groupID)
}
