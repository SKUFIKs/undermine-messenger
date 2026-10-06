package domain

// AuthUIHandler обрабатывает действия пользователя на экране авторизации
type AuthUIHandler interface {
	OnLogin(serverURL, username, password string)
	OnRegister(serverURL, username, email, password string)
}

// ChatUIHandler обрабатывает действия пользователя в основном окне чата
type ChatUIHandler interface {
	OnSendMessage(chatID string, text string)
	OnChatSelected(chat Chat)
	OnSearchQueryChanged(query string)
	OnLogout()
}
