package main

import (
	"encoding/hex"
	"fmt"
	"log"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"

	"undermine.ru/undermine-messenger/internal/api"
	"undermine.ru/undermine-messenger/internal/client"
	"undermine.ru/undermine-messenger/internal/config"
	"undermine.ru/undermine-messenger/internal/domain"
	customtheme "undermine.ru/undermine-messenger/internal/ui/theme"
	"undermine.ru/undermine-messenger/internal/ui/views"
)

// AppBackend связывает UI с бизнес-логикой
type AppBackend struct {
	apiClient    *api.Client
	networkClient *client.NetworkClient
	appState     *client.AppState
	token        string
	userID       [16]byte
	serverURL    string
}

func main() {
	log.Println("[UI] Запуск десктопного интерфейса UnderMine (Fyne v2)...")

	// Загрузка конфигурации из .env файла
	serverURL := config.LoadServerURL()
	if serverURL != "" {
		log.Printf("[Config] Адрес сервера загружен из .env: %s", serverURL)
	} else {
		log.Println("[Config] Адрес сервера не найден в .env, будет запрошен у пользователя")
	}

	fyneApp := app.NewWithID("com.undermine.desktop")
	fyneApp.Settings().SetTheme(customtheme.NewCustomTheme())

	mainWindow := fyneApp.NewWindow("UnderMine")
	mainWindow.Resize(fyne.NewSize(960, 640))
	mainWindow.SetMaster()

	authView := views.NewAuthView()
	chatView := views.NewChatView()

	backend := &AppBackend{}

	// Привязка обработчиков авторизации
	authView.OnLogin = func(username, password string) {
		backend.handleLogin(username, password, mainWindow, authView, chatView)
	}

	authView.OnRegister = func(username, email, password string) {
		backend.handleRegister(username, email, password, mainWindow, authView, chatView)
	}

	// Привязка обработчиков чата
	chatView.OnSendMessage = func(chatID string, text string) {
		backend.handleSendMessage(chatID, text, chatView)
	}

	chatView.OnChatSelected = func(chat domain.Chat) {
		backend.handleChatSelected(chat, chatView)
	}

	chatView.OnSearchChanged = func(query string) {
		log.Printf("[UI Event] Поисковый запрос: query=%s", query)
		backend.handleSearchUser(query, chatView)
	}

	chatView.OnLogout = func() {
		backend.handleLogout(mainWindow, authView)
	}

	// Отображение начального экрана
	mainWindow.SetContent(authView.BuildUI())
	mainWindow.ShowAndRun()
}

func (b *AppBackend) handleLogin(username, password string, window fyne.Window, authView *views.AuthView, chatView *views.ChatView) {
	serverURL := config.LoadServerURL()
	if serverURL == "" {
		authView.SetError("Адрес сервера не найден. Создайте файл .env с SERVER_URL")
		return
	}

	log.Printf("[UI Event] Попытка входа: user=%s server=%s", username, serverURL)

	b.serverURL = serverURL
	b.apiClient = api.NewClient(serverURL)

	resp, err := b.apiClient.Login(username, password)
	if err != nil {
		authView.SetError(err.Error())
		return
	}

	b.token = resp.Token
	userIDBytes, err := hex.DecodeString(resp.UserID)
	if err != nil {
		authView.SetError("Ошибка формата UserID")
		return
	}
	copy(b.userID[:], userIDBytes)

	// Генерация ключей для шифрования
	keyPair, err := client.GenerateKeyPair()
	if err != nil {
		authView.SetError("Ошибка генерации ключей")
		return
	}

	b.appState = client.NewAppState(b.userID, keyPair)

	// Подключение к WebSocket
	b.networkClient = client.NewNetworkClient(b.userID)
	wsURL := fmt.Sprintf("%s/ws?token=%s", serverURL, b.token)

	if err := b.networkClient.Connect(wsURL); err != nil {
		authView.SetError("Ошибка подключения к серверу: " + err.Error())
		return
	}

	// Запуск прослушивания сообщений в горутине
	go b.networkClient.Listen(
		func(msgID [16]byte, senderID [16]byte, text string) {
			b.handleIncomingMessage(msgID, senderID, text, chatView)
		},
		func(msgID [16]byte, status byte) {
			b.handleAck(msgID, status, chatView)
		},
	)

	log.Println("[UI] Успешный вход, переход к чатам")
	window.SetContent(chatView.BuildUI())
}

func (b *AppBackend) handleRegister(username, email, password string, window fyne.Window, authView *views.AuthView, chatView *views.ChatView) {
	serverURL := config.LoadServerURL()
	if serverURL == "" {
		authView.SetError("Адрес сервера не найден. Создайте файл .env с SERVER_URL")
		return
	}

	log.Printf("[UI Event] Попытка регистрации: user=%s email=%s", username, email)

	b.serverURL = serverURL
	b.apiClient = api.NewClient(serverURL)

	// Генерация ключей для регистрации
	keyPair, err := client.GenerateKeyPair()
	if err != nil {
		authView.SetError("Ошибка генерации ключей")
		return
	}

	resp, err := b.apiClient.Register(username, email, password, keyPair.PublicKey)
	if err != nil {
		authView.SetError(err.Error())
		return
	}

	b.token = resp.Token
	userIDBytes, err := hex.DecodeString(resp.UserID)
	if err != nil {
		authView.SetError("Ошибка формата UserID")
		return
	}
	copy(b.userID[:], userIDBytes)

	b.appState = client.NewAppState(b.userID, keyPair)

	// Подключение к WebSocket
	b.networkClient = client.NewNetworkClient(b.userID)
	wsURL := fmt.Sprintf("%s/ws?token=%s", serverURL, b.token)

	if err := b.networkClient.Connect(wsURL); err != nil {
		authView.SetError("Ошибка подключения к серверу: " + err.Error())
		return
	}

	go b.networkClient.Listen(
		func(msgID [16]byte, senderID [16]byte, text string) {
			b.handleIncomingMessage(msgID, senderID, text, chatView)
		},
		func(msgID [16]byte, status byte) {
			b.handleAck(msgID, status, chatView)
		},
	)

	log.Println("[UI] Успешная регистрация, переход к чатам")
	window.SetContent(chatView.BuildUI())
}

func (b *AppBackend) handleSendMessage(chatID string, text string, chatView *views.ChatView) {
	log.Printf("[UI Event] Отправка сообщения: chat=%s text=%s", chatID, text)

	if b.networkClient == nil || b.appState == nil {
		log.Println("[Ошибка] Нет подключения к серверу")
		return
	}

	// Находим контакт по chatID
	contact := b.findContactByID(chatID)
	if contact == nil {
		log.Printf("[Ошибка] Контакт не найден: %s", chatID)
		return
	}

	// Проверяем, что не отправляем самому себе
	if contact.ID == b.userID {
		log.Println("[Ошибка] Нельзя отправить сообщение самому себе")
		return
	}

	// Отправляем сообщение
	msgID, err := b.networkClient.SendTextMessage(contact.ID, text)
	if err != nil {
		log.Printf("[Ошибка] Не удалось отправить сообщение: %v", err)
		return
	}

	log.Printf("[OK] Сообщение отправлено: msgID=%x", msgID)

	// Добавляем сообщение в локальную историю
	b.appState.AddMessage(contact.ID, true, text, msgID)

	// Обновляем UI
	msg := domain.Message{
		ID:         hex.EncodeToString(msgID[:]),
		ChatID:     chatID,
		SenderID:   hex.EncodeToString(b.userID[:]),
		Content:    text,
		Status:     domain.StatusSending,
		IsOutgoing: true,
		CreatedAt:  time.Now(),
	}
	chatView.AppendMessage(msg)
}

func (b *AppBackend) handleChatSelected(chat domain.Chat, chatView *views.ChatView) {
	log.Printf("[UI Event] Выбран диалог: id=%s title=%s", chat.ID, chat.Title)

	if b.appState == nil {
		return
	}

	// Находим контакт по chatID
	contact := b.findContactByID(chat.ID)
	if contact == nil {
		log.Printf("[Ошибка] Контакт не найден: %s", chat.ID)
		return
	}

	// Устанавливаем активный контакт
	b.appState.SetActiveContact(contact.ID)

	// Загружаем историю сообщений
	messages := make([]domain.Message, 0, len(contact.Messages))
	for _, msg := range contact.Messages {
		senderID := hex.EncodeToString(b.userID[:])
		if !msg.IsOut {
			senderID = hex.EncodeToString(contact.ID[:])
		}

		messages = append(messages, domain.Message{
			ID:         hex.EncodeToString(msg.ID[:]),
			ChatID:     chat.ID,
			SenderID:   senderID,
			Content:    msg.Text,
			Status:     domain.StatusDelivered,
			IsOutgoing: msg.IsOut,
			CreatedAt:  msg.Timestamp,
		})
	}

	chatView.SetMessages(messages)
}

func (b *AppBackend) handleIncomingMessage(msgID [16]byte, senderID [16]byte, text string, chatView *views.ChatView) {
	log.Printf("[UI Event] Входящее сообщение от %x: %s", senderID, text)

	if b.appState == nil {
		return
	}

	// Находим контакт по ID отправителя
	contact := b.appState.GetContact(senderID)
	if contact == nil {
		log.Printf("[Ошибка] Контакт не найден: %x", senderID)
		return
	}

	// Добавляем сообщение в локальную историю
	b.appState.AddMessage(contact.ID, false, text, msgID)

	// Обновляем UI из главного потока
	fyne.DoAndWait(func() {
		msg := domain.Message{
			ID:         hex.EncodeToString(msgID[:]),
			ChatID:     hex.EncodeToString(contact.ID[:]),
			SenderID:   hex.EncodeToString(contact.ID[:]),
			Content:    text,
			Status:     domain.StatusDelivered,
			IsOutgoing: false,
			CreatedAt:  time.Now(),
		}
		chatView.AppendMessage(msg)
	})
}

func (b *AppBackend) handleAck(msgID [16]byte, status byte, chatView *views.ChatView) {
	log.Printf("[UI Event] ACK сообщения: msgID=%x status=%d", msgID, status)

	if b.appState == nil {
		return
	}

	// Находим сообщение по msgID и обновляем статус
	contacts := b.appState.GetAllContacts()
	for _, contact := range contacts {
		for i, msg := range contact.Messages {
			if msg.ID == msgID {
				// Обновляем статус сообщения
				switch status {
				case 0x01:
					contact.Messages[i].Delivered = true
					log.Printf("[OK] Сообщение сохранено на сервере")
				case 0x02:
					contact.Messages[i].Delivered = true
					log.Printf("[OK] Сообщение доставлено получателю")
				}

				// Обновляем UI из главного потока — обновляем статус, а не добавляем новое
				fyne.DoAndWait(func() {
					// Находим и обновляем существующее пузырь сообщения
					// Пока просто обновим список сообщений
					b.refreshChatView(contact.ID, chatView)
				})
				return
			}
		}
	}
}

func (b *AppBackend) refreshChatView(contactID [16]byte, chatView *views.ChatView) {
	if b.appState == nil {
		return
	}

	contact := b.appState.GetContact(contactID)
	if contact == nil {
		return
	}

	messages := make([]domain.Message, 0, len(contact.Messages))
	for _, msg := range contact.Messages {
		senderID := hex.EncodeToString(b.userID[:])
		if !msg.IsOut {
			senderID = hex.EncodeToString(contact.ID[:])
		}

		messages = append(messages, domain.Message{
			ID:         hex.EncodeToString(msg.ID[:]),
			ChatID:     hex.EncodeToString(contact.ID[:]),
			SenderID:   senderID,
			Content:    msg.Text,
			Status:     domain.StatusDelivered,
			IsOutgoing: msg.IsOut,
			CreatedAt:  msg.Timestamp,
		})
	}

	chatView.SetMessages(messages)
}

func (b *AppBackend) handleLogout(window fyne.Window, authView *views.AuthView) {
	log.Println("[UI Event] Выход из аккаунта")

	if b.networkClient != nil {
		b.networkClient.Close()
	}

	b.token = ""
	b.userID = [16]byte{}
	b.appState = nil
	b.networkClient = nil

	window.SetContent(authView.BuildUI())
}

func (b *AppBackend) handleSearchUser(query string, chatView *views.ChatView) {
	if b.apiClient == nil || b.token == "" {
		return
	}

	query = strings.TrimSpace(query)
	if query == "" {
		return
	}

	log.Printf("[API] Поиск пользователя: %s", query)

	resp, err := b.apiClient.SearchUser(b.token, query)
	if err != nil {
		log.Printf("[API] Ошибка поиска: %v", err)
		return
	}

	log.Printf("[API] Найден пользователь: %s (%s)", resp.Username, resp.UserID)

	// Добавляем пользователя в контакты
	userIDBytes, err := hex.DecodeString(resp.UserID)
	if err != nil {
		log.Printf("[API] Ошибка формата UserID: %v", err)
		return
	}

	var userID [16]byte
	copy(userID[:], userIDBytes)

	pubKeyBytes, err := hex.DecodeString(resp.PublicKey)
	if err != nil {
		log.Printf("[API] Ошибка формата PublicKey: %v", err)
		return
	}

	var pubKey [32]byte
	copy(pubKey[:], pubKeyBytes)

	if b.appState != nil {
		err := b.appState.AddContact(resp.Username, userID, pubKey)
		if err != nil {
			log.Printf("[API] Ошибка добавления контакта: %v", err)
			return
		}

		// Устанавливаем SharedKey для сетевого клиента
		if b.networkClient != nil {
			contact := b.appState.GetContact(userID)
			if contact != nil {
				b.networkClient.SetContactSharedKey(userID, contact.SharedKey)
				log.Printf("[OK] SharedKey установлен для контакта: %s", resp.Username)
			}
		}

		// Обновляем список чатов в UI
		contacts := b.appState.GetAllContacts()
		chats := make([]domain.Chat, 0, len(contacts))
		for _, c := range contacts {
			chats = append(chats, domain.Chat{
				ID:    hex.EncodeToString(c.ID[:]),
				Title: c.Name,
				PeerUser: domain.User{
					ID:       hex.EncodeToString(c.ID[:]),
					Username: c.Name,
				},
			})
		}
		chatView.SetChats(chats)
	}
}

// Вспомогательная функция для получения контактов
func (b *AppBackend) getContacts() []*client.Contact {
	if b.appState == nil {
		return nil
	}
	return b.appState.GetAllContacts()
}

// Вспомогательная функция для поиска контакта по ID
func (b *AppBackend) findContactByID(chatID string) *client.Contact {
	contacts := b.getContacts()
	for _, c := range contacts {
		if hex.EncodeToString(c.ID[:]) == chatID {
			return c
		}
	}
	return nil
}

// Вспомогательная функция для нормализации URL
func normalizeURL(url string) string {
	url = strings.TrimSpace(url)
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "http://" + url
	}
	return url
}
