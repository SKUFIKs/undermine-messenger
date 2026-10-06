package components

import (
	"fmt"
	"image/color"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// Цветовые палитры для генерации аватаров собеседников
var avatarPalette = []color.NRGBA{
	{R: 0xC7, G: 0x60, B: 0x29, A: 0xFF}, // Фирменный терракотовый
	{R: 0x3E, G: 0x7E, B: 0xB5, A: 0xFF}, // Синий
	{R: 0x3B, G: 0x97, B: 0x6C, A: 0xFF}, // Зеленый
	{R: 0x8E, G: 0x54, B: 0x9F, A: 0xFF}, // Фиолетовый
	{R: 0xD3, G: 0x47, B: 0x47, A: 0xFF}, // Красный
	{R: 0xC8, G: 0x88, B: 0x20, A: 0xFF}, // Янтарный
}

// AvatarWidget представляет круглый аватар с инициалами пользователя
type AvatarWidget struct {
	widget.BaseWidget
	size     float32
	title    string
	circle   *canvas.Circle
	label    *canvas.Text
	layout   *fyne.Container
}

// NewAvatarWidget создает круглый аватар фиксированного диаметра (например, 42px)
func NewAvatarWidget(title string, size float32) *AvatarWidget {
	a := &AvatarWidget{
		size:  size,
		title: title,
	}
	a.ExtendBaseWidget(a)
	a.setupUI()
	return a
}

func (a *AvatarWidget) setupUI() {
	bgColor := a.pickColor(a.title)

	a.circle = canvas.NewCircle(bgColor)
	a.circle.Resize(fyne.NewSize(a.size, a.size))

	initials := a.getInitials(a.title)
	a.label = canvas.NewText(initials, color.White)
	a.label.TextSize = a.size * 0.42
	a.label.TextStyle = fyne.TextStyle{Bold: true}
	a.label.Alignment = fyne.TextAlignCenter

	a.layout = container.NewStack(a.circle, container.NewCenter(a.label))
}

// SetTitle обновляет имя и перерисовывает аватар
func (a *AvatarWidget) SetTitle(title string) {
	a.title = title
	a.circle.FillColor = a.pickColor(title)
	a.label.Text = a.getInitials(title)
	a.Refresh()
}

func (a *AvatarWidget) getInitials(name string) string {
	parts := strings.Fields(strings.TrimSpace(name))
	if len(parts) == 0 {
		return "?"
	}
	if len(parts) == 1 {
		runes := []rune(parts[0])
		if len(runes) >= 2 {
			return strings.ToUpper(string(runes[:2]))
		}
		return strings.ToUpper(string(runes[:1]))
	}
	return strings.ToUpper(fmt.Sprintf("%c%c", []rune(parts[0])[0], []rune(parts[1])[0]))
}

func (a *AvatarWidget) pickColor(name string) color.NRGBA {
	var hash uint
	for _, b := range []byte(name) {
		hash = hash*31 + uint(b)
	}
	return avatarPalette[hash%uint(len(avatarPalette))]
}

// CreateRenderer реализует контракт виджета Fyne
func (a *AvatarWidget) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(a.layout)
}

// MinSize возвращает размер круга
func (a *AvatarWidget) MinSize() fyne.Size {
	return fyne.NewSize(a.size, a.size)
}