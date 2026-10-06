package views

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"undermine.ru/undermine-server/internal/domain"
	"undermine.ru/undermine-server/internal/ui/components"
	customtheme "undermine.ru/undermine-server/internal/ui/theme"
)

// ChatView управляет главным двухпанельным окном мессенджера (Master-Detail Split View)
type ChatView struct {
	// Обработчики пользовательских действий (внедряются внешней бизнес-логикой)
	OnSendMessage   func(chatID string, text string)
	OnChatSelected  func(chat domain.Chat)
	OnSearchChanged func(query string)
	OnLogout        func()

	// Текущее состояние
	activeChat     *domain.Chat
	chatsList      []domain.Chat
	chatWidgetsMap map[string]*components.ChatItemWidget

	// Виджеты левой панели
	searchEntry *widget.Entry
	chatsBox    *fyne.Container
	chatsScroll *container.Scroll

	// Виджеты правой панели (Активный чат)
	emptyPlaceholder *canvas.Text
	chatHeaderTitle  *canvas.Text
	chatHeaderStatus *canvas.Text
	chatHeaderBox    *fyne.Container
	msgListContainer *fyne.Container
	msgScroll        *container.Scroll
	inputEntry       *widget.Entry
	sendButton       *widget.Button
	chatAreaBorder   *fyne.Container

	rootSplit *container.Split
}

// NewChatView создает чистый двухпанельный интерфейс без дефолтных сообщений и моков
func NewChatView() *ChatView {
	return &ChatView{
		chatWidgetsMap: make(map[string]*components.ChatItemWidget),
	}
}

// BuildUI собирает графический интерфейс сплит-панелей
func (cv *ChatView) BuildUI() fyne.CanvasObject {
	leftPanel := cv.buildLeftPanel()
	rightPanel := cv.buildRightPanel()

	cv.rootSplit = container.NewHSplit(leftPanel, rightPanel)
	cv.rootSplit.Offset = 0.32

	return cv.rootSplit
}

// buildLeftPanel создает боковую панель поиска и списка диалогов
func (cv *ChatView) buildLeftPanel() fyne.CanvasObject {
	cv.searchEntry = widget.NewEntry()
	cv.searchEntry.SetPlaceHolder("Поиск диалогов...")
	cv.searchEntry.OnChanged = func(query string) {
		if cv.OnSearchChanged != nil {
			cv.OnSearchChanged(query)
		}
	}

	logoutBtn := widget.NewButton("Выход", func() {
		if cv.OnLogout != nil {
			cv.OnLogout()
		}
	})
	logoutBtn.Importance = widget.LowImportance

	searchBar := container.NewBorder(nil, nil, nil, logoutBtn, container.NewPadded(cv.searchEntry))

	cv.chatsBox = container.NewVBox()
	cv.chatsScroll = container.NewVScroll(cv.chatsBox)

	sidebarBg := canvas.NewRectangle(customtheme.ColorSidebarBg)
	panelContent := container.NewBorder(searchBar, nil, nil, nil, cv.chatsScroll)

	return container.NewStack(sidebarBg, panelContent)
}

// buildRightPanel создает область активного чата
func (cv *ChatView) buildRightPanel() fyne.CanvasObject {
	// Верхний бар активного диалога
	cv.chatHeaderTitle = canvas.NewText("", customtheme.ColorTextMain)
	cv.chatHeaderTitle.TextSize = 15
	cv.chatHeaderTitle.TextStyle = fyne.TextStyle{Bold: true}

	cv.chatHeaderStatus = canvas.NewText("", customtheme.ColorTextMuted)
	cv.chatHeaderStatus.TextSize = 11

	headerTextContainer := container.NewVBox(cv.chatHeaderTitle, cv.chatHeaderStatus)
	headerBorder := container.NewPadded(headerTextContainer)
	headerBg := canvas.NewRectangle(customtheme.ColorSidebarBg)
	cv.chatHeaderBox = container.NewStack(headerBg, headerBorder)

	// Контейнер пузырей сообщений
	cv.msgListContainer = container.NewVBox()
	cv.msgScroll = container.NewVScroll(container.NewPadded(cv.msgListContainer))

	// Нижняя панель ввода сообщения
	cv.inputEntry = widget.NewEntry()
	cv.inputEntry.SetPlaceHolder("Написать сообщение... (Enter для отправки)")
	cv.inputEntry.OnSubmitted = func(_ string) {
		cv.handleSend()
	}

	cv.sendButton = widget.NewButton("Отправить", cv.handleSend)
	cv.sendButton.Importance = widget.HighImportance

	inputBar := container.NewBorder(
		nil, nil, nil,
		cv.sendButton,
		container.NewPadded(cv.inputEntry),
	)
	inputBg := canvas.NewRectangle(customtheme.ColorSidebarBg)
	inputContainer := container.NewStack(inputBg, inputBar)

	// Область активной переписки
	cv.chatAreaBorder = container.NewBorder(cv.chatHeaderBox, inputContainer, nil, nil, cv.msgScroll)
	cv.chatAreaBorder.Hide()

	// Заглушка, когда диалог не выбран (чистый лейаут)
	cv.emptyPlaceholder = canvas.NewText("Выберите чат, чтобы начать общение", customtheme.ColorTextMuted)
	cv.emptyPlaceholder.TextSize = 14
	cv.emptyPlaceholder.Alignment = fyne.TextAlignCenter
	placeholderCenter := container.NewCenter(cv.emptyPlaceholder)

	windowBg := canvas.NewRectangle(customtheme.ColorWindowBg)
	return container.NewStack(windowBg, placeholderCenter, cv.chatAreaBorder)
}

// SetChats обновляет список диалогов в боковой панели
func (cv *ChatView) SetChats(chats []domain.Chat) {
	cv.chatsList = chats
	cv.chatsBox.Objects = nil
	cv.chatWidgetsMap = make(map[string]*components.ChatItemWidget)

	for _, c := range chats {
		chat := c
		item := components.NewChatItemWidget(chat, func(selected domain.Chat) {
			cv.SelectChat(selected)
			if cv.OnChatSelected != nil {
				cv.OnChatSelected(selected)
			}
		})
		if cv.activeChat != nil && cv.activeChat.ID == chat.ID {
			item.SetSelected(true)
		}
		cv.chatWidgetsMap[chat.ID] = item
		cv.chatsBox.Add(item)
	}
	cv.chatsBox.Refresh()
}

// SelectChat переключает активный диалог и обновляет шапку
func (cv *ChatView) SelectChat(chat domain.Chat) {
	cv.activeChat = &chat

	// Обновляем подсветку активного элемента
	for id, w := range cv.chatWidgetsMap {
		w.SetSelected(id == chat.ID)
	}

	// Показываем область чата, скрываем плейсхолдер
	cv.emptyPlaceholder.Hide()
	cv.chatAreaBorder.Show()

	// Устанавливаем заголовок и онлайн-статус
	cv.chatHeaderTitle.Text = chat.Title
	if chat.PeerUser.IsOnline {
		cv.chatHeaderStatus.Text = "● в сети"
		cv.chatHeaderStatus.Color = customtheme.ColorOnline
	} else {
		cv.chatHeaderStatus.Text = "не в сети"
		cv.chatHeaderStatus.Color = customtheme.ColorTextMuted
	}

	cv.chatHeaderBox.Refresh()
	cv.chatAreaBorder.Refresh()
}

// SetMessages очищает и заполняет область сообщений новым списком
func (cv *ChatView) SetMessages(messages []domain.Message) {
	cv.msgListContainer.Objects = nil
	for _, m := range messages {
		bubble := components.NewMessageBubbleWidget(m)
		cv.msgListContainer.Add(bubble)
	}
	cv.msgListContainer.Refresh()
	cv.msgScroll.ScrollToBottom()
}

// AppendMessage добавляет единичное сообщение в конец ленты
func (cv *ChatView) AppendMessage(msg domain.Message) {
	bubble := components.NewMessageBubbleWidget(msg)
	cv.msgListContainer.Add(bubble)
	cv.msgListContainer.Refresh()
	cv.msgScroll.ScrollToBottom()
}

// SetPeerStatus обновляет текстовый статус в шапке (например: "в сети", "печатает...")
func (cv *ChatView) SetPeerStatus(statusText string, isOnline bool) {
	cv.chatHeaderStatus.Text = statusText
	if isOnline {
		cv.chatHeaderStatus.Color = customtheme.ColorOnline
	} else {
		cv.chatHeaderStatus.Color = customtheme.ColorTextMuted
	}
	cv.chatHeaderBox.Refresh()
}

// ClearInput очищает поле ввода сообщения
func (cv *ChatView) ClearInput() {
	cv.inputEntry.SetText("")
}

// GetActiveChat возвращает текущий выбранный чат
func (cv *ChatView) GetActiveChat() *domain.Chat {
	return cv.activeChat
}

func (cv *ChatView) handleSend() {
	text := strings.TrimSpace(cv.inputEntry.Text)
	if text == "" || cv.activeChat == nil {
		return
	}

	cv.ClearInput()
	if cv.OnSendMessage != nil {
		cv.OnSendMessage(cv.activeChat.ID, text)
	}
}
