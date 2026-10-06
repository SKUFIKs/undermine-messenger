package theme

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// Цветовые константы Telegram-стиля в формате RGBA
var (
	// Основной фон приложения (рабочая область, фон окна) #171717
	ColorWindowBg = color.NRGBA{R: 0x17, G: 0x17, B: 0x17, A: 0xFF}

	// Дополнительный фон (боковая панель чатов, поля ввода, карточки) #222222
	ColorSidebarBg = color.NRGBA{R: 0x22, G: 0x22, B: 0x22, A: 0xFF}

	// Акцентный цвет (фон кнопок, бейджи непрочитанных, свои пузыри сообщений) #C76029
	ColorAccent = color.NRGBA{R: 0xC7, G: 0x60, B: 0x29, A: 0xFF}

	// Чужие пузыри сообщений #2A2A2A
	ColorPeerBubble = color.NRGBA{R: 0x2A, G: 0x2A, B: 0x2A, A: 0xFF}

	// Разделители и границы #2D2D2D
	ColorSeparator = color.NRGBA{R: 0x2D, G: 0x2D, B: 0x2D, A: 0xFF}

	// Основной текст #FFFFFF
	ColorTextMain = color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}

	// Второстепенный текст #D1D1D1
	ColorTextMuted = color.NRGBA{R: 0xD1, G: 0xD1, B: 0xD1, A: 0xFF}

	// Ошибки #E05252
	ColorError = color.NRGBA{R: 0xE0, G: 0x52, B: 0x52, A: 0xFF}

	// Онлайн индикатор #4CAF50
	ColorOnline = color.NRGBA{R: 0x4C, G: 0xAF, B: 0x50, A: 0xFF}
)

// CustomTheme реализует интерфейс fyne.Theme со строгим соответствием палитре
type CustomTheme struct{}

var _ fyne.Theme = (*CustomTheme)(nil)

// NewCustomTheme создает экземпляр кастомной темы Fyne
func NewCustomTheme() *CustomTheme {
	return &CustomTheme{}
}

// Color возвращает соответствующий цвет согласно названию токена Fyne Theme
func (t *CustomTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground:
		return ColorWindowBg
	case theme.ColorNameInputBackground:
		return ColorSidebarBg
	case theme.ColorNamePrimary:
		return ColorAccent
	case theme.ColorNameButton:
		return ColorAccent
	case theme.ColorNameForeground:
		return ColorTextMain
	case theme.ColorNamePlaceHolder:
		return ColorTextMuted
	case theme.ColorNameSeparator:
		return ColorSeparator
	case theme.ColorNameShadow:
		return color.NRGBA{R: 0x00, G: 0x00, B: 0x00, A: 0x66}
	case theme.ColorNameError:
		return ColorError
	case theme.ColorNameHover:
		return color.NRGBA{R: 0x2C, G: 0x2C, B: 0x2C, A: 0xFF}
	case theme.ColorNameSelection:
		return color.NRGBA{R: 0xC7, G: 0x60, B: 0x29, A: 0x55}
	default:
		return theme.DefaultTheme().Color(name, theme.VariantDark)
	}
}

// Font делегирует стандартным шрифтам системы с поддержкой масштабирования DPI
func (t *CustomTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

// Icon возвращает стандартные векторные иконки Fyne
func (t *CustomTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

// Size задает геометрию элементов: скругления 8-10px, отступы и размеры шрифтов
func (t *CustomTheme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case theme.SizeNameInputBorder:
		return 1.0
	case theme.SizeNamePadding:
		return 8.0
	case theme.SizeNameInnerPadding:
		return 6.0
	case theme.SizeNameText:
		return 13.0
	case theme.SizeNameHeadingText:
		return 17.0
	case theme.SizeNameSubHeadingText:
		return 14.0
	case theme.SizeNameCaptionText:
		return 11.0
	default:
		return theme.DefaultTheme().Size(name)
	}
}