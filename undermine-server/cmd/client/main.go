package main

import (
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"

	approuter "undermine.ru/undermine-server/internal/app"
	"undermine.ru/undermine-server/internal/domain"
	customtheme "undermine.ru/undermine-server/internal/ui/theme"
	"undermine.ru/undermine-server/internal/ui/views"
)

func main() {
	log.Println("[UI] Запуск десктопного интерфейса UnderMine (Fyne v2)...")

	// 1. Инициализация Fyne приложения с кастомной темой
	fyneApp := app.NewWithID("com.undermine.desktop")
	fyneApp.Settings().SetTheme(customtheme.NewCustomTheme())

	// 2. Настройка главного окна (минимальный размер 850x550)
	mainWindow := fyneApp.NewWindow("UnderMine")
	mainWindow.Resize(fyne.NewSize(960, 640))
	mainWindow.SetMaster()

	// 3. Создание экранов пользовательского интерфейса
	authView := views.NewAuthView()
	chatView := views.NewChatView()

	// 4. Инициализация навигатора сцен (Router)
	router := approuter.NewRouter(mainWindow, authView, chatView)

	// 5. Привязка обработчиков интерфейса к вашим готовым методам
	authView.OnLogin = func(serverURL, username, password string) {
		log.Printf("[UI Event] Попытка входа: user=%s server=%s", username, serverURL)
		// Подключите здесь вызов вашего готового AuthService.Login
		// При успехе:
		router.ShowChat()
	}

	authView.OnRegister = func(serverURL, username, email, password string) {
		log.Printf("[UI Event] Попытка регистрации: user=%s email=%s", username, email)
		// Подключите здесь вызов вашего готового AuthService.Register
		// При успехе:
		router.ShowChat()
	}

	chatView.OnSendMessage = func(chatID string, text string) {
		log.Printf("[UI Event] Отправка сообщения: chat=%s text=%s", chatID, text)
		// Подключите здесь вызов вашего готового ChatService.SendMessage
	}

	chatView.OnChatSelected = func(chat domain.Chat) {
		log.Printf("[UI Event] Выбран диалог: id=%s title=%s", chat.ID, chat.Title)
		// Подключите здесь загрузку истории сообщений через ChatService.GetHistory
	}

	chatView.OnSearchChanged = func(query string) {
		log.Printf("[UI Event] Поисковый запрос: query=%s", query)
		// Подключите здесь фильтрацию контактов
	}

	chatView.OnLogout = func() {
		log.Println("[UI Event] Выход из аккаунта")
		router.ShowAuth()
	}

	// 6. Отображение начального экрана и старт оконного цикла Fyne
	router.ShowAuth()
	mainWindow.ShowAndRun()
}
