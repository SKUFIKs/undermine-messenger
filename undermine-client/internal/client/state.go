package client

import (
	"fmt"
	"sync"
	"time"
)

// ChatMessage представляет сообщение в чате
type ChatMessage struct {
	ID        [16]byte
	IsOut     bool
	Text      string
	Timestamp time.Time
	Delivered bool
}

// Contact представляет контакт пользователя
type Contact struct {
	ID        [16]byte
	Name      string
	PublicKey [32]byte
	SharedKey [32]byte
	Messages  []ChatMessage
}

// AppState представляет состояние приложения
type AppState struct {
	mu            sync.RWMutex
	MyID          [16]byte
	MyKeys        *KeyPair
	Contacts      map[[16]byte]*Contact
	ActiveContact [16]byte
	OnUpdate      func()
}

// NewAppState создает новое состояние приложения
func NewAppState(myID [16]byte, keys *KeyPair) *AppState {
	return &AppState{
		MyID:     myID,
		MyKeys:   keys,
		Contacts: make(map[[16]byte]*Contact),
	}
}

// AddContact добавляет новый контакт
func (s *AppState) AddContact(name string, contactID [16]byte, pubKey [32]byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	sharedKey, err := ComputeSharedKey(s.MyKeys.PrivateKey, pubKey)
	if err != nil {
		return fmt.Errorf("ошибка вычисления ключа: %w", err)
	}

	s.Contacts[contactID] = &Contact{
		ID:        contactID,
		Name:      name,
		PublicKey: pubKey,
		SharedKey: sharedKey,
		Messages:  make([]ChatMessage, 0),
	}

	if s.ActiveContact == [16]byte{} {
		s.ActiveContact = contactID
	}

	if s.OnUpdate != nil {
		s.OnUpdate()
	}
	return nil
}

// AddMessage добавляет сообщение в чат
func (s *AppState) AddMessage(contactID [16]byte, isOut bool, text string, msgID [16]byte) {
	s.mu.Lock()
	defer s.mu.Unlock()

	contact, ok := s.Contacts[contactID]
	if !ok {
		return
	}

	contact.Messages = append(contact.Messages, ChatMessage{
		ID:        msgID,
		IsOut:     isOut,
		Text:      text,
		Timestamp: time.Now(),
		Delivered: isOut,
	})

	if s.OnUpdate != nil {
		s.OnUpdate()
	}
}

// GetActiveContact возвращает активный контакт
func (s *AppState) GetActiveContact() *Contact {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Contacts[s.ActiveContact]
}

// GetContact возвращает контакт по ID
func (s *AppState) GetContact(contactID [16]byte) *Contact {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Contacts[contactID]
}

// SetActiveContact устанавливает активный контакт
func (s *AppState) SetActiveContact(contactID [16]byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ActiveContact = contactID
}

// GetAllContacts возвращает все контакты
func (s *AppState) GetAllContacts() []*Contact {
	s.mu.RLock()
	defer s.mu.RUnlock()

	contacts := make([]*Contact, 0, len(s.Contacts))
	for _, c := range s.Contacts {
		contacts = append(contacts, c)
	}
	return contacts
}
