package components

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"undermine.ru/undermine-server/internal/domain"
	customtheme "undermine.ru/undermine-server/internal/ui/theme"
)

// MessageBubbleWidget отображает пузырь сообщения в Telegram-стиле
type MessageBubbleWidget struct {
	widget.BaseWidget
	msg  domain.Message
	root *fyne.Container
}

// NewMessageBubbleWidget создает пузырь сообщения со скругленными углами и временем
func NewMessageBubbleWidget(msg domain.Message) *MessageBubbleWidget {
	b := &MessageBubbleWidget{
		msg: msg,
	}
	b.ExtendBaseWidget(b)
	b.buildUI()
	return b
}

func (b *MessageBubbleWidget) buildUI() {
	bgColor := customtheme.ColorPeerBubble
	if b.msg.IsOutgoing {
		bgColor = customtheme.ColorAccent
	}

	bgRect := canvas.NewRectangle(bgColor)
	bgRect.CornerRadius = 10

	contentLabel := widget.NewLabel(b.msg.Content)
	contentLabel.Wrapping = fyne.TextWrapWord
	contentLabel.TextStyle = fyne.TextStyle{}

	timeStr := b.msg.CreatedAt.Format("15:04")
	if b.msg.IsOutgoing {
		switch b.msg.Status {
		case domain.StatusSending:
			timeStr += " 🕒"
		case domain.StatusSent:
			timeStr += " ✓"
		case domain.StatusDelivered, domain.StatusRead:
			timeStr += " ✓✓"
		case domain.StatusFailed:
			timeStr += " ⚠️"
		}
	}

	timeLabel := canvas.NewText(timeStr, color.NRGBA{R: 0xE0, G: 0xE0, B: 0xE0, A: 0xCC})
	timeLabel.TextSize = 10
	timeLabel.Alignment = fyne.TextAlignTrailing

	inner := container.NewVBox(
		contentLabel,
		container.NewHBox(
			widget.NewLabel(""),
			timeLabel,
		),
	)

	padded := container.NewPadded(inner)
	bubble := container.NewStack(bgRect, padded)

	if b.msg.IsOutgoing {
		b.root = container.NewHBox(
			widget.NewLabel(""),
			bubble,
		)
	} else {
		b.root = container.NewHBox(
			bubble,
			widget.NewLabel(""),
		)
	}
}

// CreateRenderer реализует интерфейс виджета Fyne
func (b *MessageBubbleWidget) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(b.root)
}

// UpdateStatus обновляет статус доставки
func (b *MessageBubbleWidget) UpdateStatus(status domain.MessageStatus) {
	b.msg.Status = status
	b.buildUI()
	b.Refresh()
}
