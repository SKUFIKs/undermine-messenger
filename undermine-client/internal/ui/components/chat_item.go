package components

import (
	"fmt"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"undermine.ru/undermine-messenger/internal/domain"
	customtheme "undermine.ru/undermine-messenger/internal/ui/theme"
)

// ChatItemWidget представляет элемент списка диалогов в Telegram стиле
type ChatItemWidget struct {
	widget.BaseWidget
	chat       domain.Chat
	isSelected bool
	onSelected func(chat domain.Chat)

	root   *fyne.Container
	bgRect *canvas.Rectangle
}

// NewChatItemWidget создает элемент боковой панели списка диалогов
func NewChatItemWidget(chat domain.Chat, onSelected func(chat domain.Chat)) *ChatItemWidget {
	w := &ChatItemWidget{
		chat:       chat,
		onSelected: onSelected,
	}
	w.ExtendBaseWidget(w)
	w.buildUI()
	return w
}

func (w *ChatItemWidget) buildUI() {
	var bgColor color.Color = color.Transparent
	if w.isSelected {
		bgColor = color.NRGBA{R: 0x2C, G: 0x2C, B: 0x2C, A: 0xFF}
	}
	w.bgRect = canvas.NewRectangle(bgColor)
	w.bgRect.CornerRadius = 8

	avatar := NewAvatarWidget(w.chat.Title, 44)

	titleLabel := widget.NewLabel(w.chat.Title)
	titleLabel.TextStyle = fyne.TextStyle{Bold: true}

	timeStr := ""
	if !w.chat.UpdatedAt.IsZero() {
		timeStr = w.chat.UpdatedAt.Format("15:04")
	}
	timeLabel := canvas.NewText(timeStr, customtheme.ColorTextMuted)
	timeLabel.TextSize = 11

	topRow := container.NewBorder(nil, nil, nil, timeLabel, titleLabel)

	previewText := "Нет сообщений"
	if w.chat.LastMessage != nil {
		previewText = w.chat.LastMessage.Content
		if len(previewText) > 36 {
			previewText = previewText[:33] + "..."
		}
	}
	previewLabel := widget.NewLabel(previewText)
	previewLabel.Truncation = fyne.TextTruncateEllipsis

	var badge fyne.CanvasObject = widget.NewLabel("")
	if w.chat.UnreadCount > 0 {
		badgeBg := canvas.NewRectangle(customtheme.ColorAccent)
		badgeBg.CornerRadius = 9
		badgeBg.Resize(fyne.NewSize(20, 18))

		badgeText := canvas.NewText(fmt.Sprintf("%d", w.chat.UnreadCount), color.White)
		badgeText.TextSize = 10
		badgeText.TextStyle = fyne.TextStyle{Bold: true}
		badge = container.NewStack(badgeBg, container.NewCenter(badgeText))
	}

	bottomRow := container.NewBorder(nil, nil, nil, badge, previewLabel)
	contentBox := container.NewVBox(topRow, bottomRow)

	rowLayout := container.NewBorder(nil, nil, avatar, nil, container.NewPadded(contentBox))

	button := widget.NewButton("", func() {
		if w.onSelected != nil {
			w.onSelected(w.chat)
		}
	})
	button.Importance = widget.LowImportance

	w.root = container.NewStack(w.bgRect, rowLayout, button)
}

// SetSelected переключает визуальное выделение активного чата
func (w *ChatItemWidget) SetSelected(selected bool) {
	w.isSelected = selected
	if selected {
		w.bgRect.FillColor = color.NRGBA{R: 0x2C, G: 0x2C, B: 0x2C, A: 0xFF}
	} else {
		w.bgRect.FillColor = color.Transparent
	}
	w.Refresh()
}

// UpdateData обновляет отображаемые данные диалога
func (w *ChatItemWidget) UpdateData(chat domain.Chat) {
	w.chat = chat
	w.buildUI()
	w.Refresh()
}

// CreateRenderer реализует контракт виджета Fyne
func (w *ChatItemWidget) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(w.root)
}
