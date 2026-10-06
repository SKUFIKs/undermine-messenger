package views

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	customtheme "undermine.ru/undermine-messenger/internal/ui/theme"
)

// AuthView реализует презентационный слой экрана входа и регистрации
type AuthView struct {
	OnLogin    func(username, password string)
	OnRegister func(username, email, password string)

	isRegister bool
	userEntry  *widget.Entry
	emailEntry *widget.Entry
	passEntry  *widget.Entry
	errorLabel *widget.Label
	actionBtn  *widget.Button
	toggleBtn  *widget.Button
}

// NewAuthView создает форму авторизации без хардкода и дефолтных значений
func NewAuthView() *AuthView {
	return &AuthView{
		isRegister: false,
	}
}

// BuildUI собирает графический интерфейс экрана
func (v *AuthView) BuildUI() fyne.CanvasObject {
	v.userEntry = widget.NewEntry()
	v.userEntry.SetPlaceHolder("Имя пользователя (username)")

	v.emailEntry = widget.NewEntry()
	v.emailEntry.SetPlaceHolder("Электронная почта (email)")
	v.emailEntry.Hide()

	v.passEntry = widget.NewPasswordEntry()
	v.passEntry.SetPlaceHolder("Пароль")
	v.passEntry.OnSubmitted = func(_ string) {
		v.handleSubmit()
	}

	v.errorLabel = widget.NewLabel("")
	v.errorLabel.Wrapping = fyne.TextWrapWord
	v.errorLabel.Hide()

	v.actionBtn = widget.NewButton("Войти", v.handleSubmit)
	v.actionBtn.Importance = widget.HighImportance

	v.toggleBtn = widget.NewButton("Создать аккаунт", v.toggleMode)
	v.toggleBtn.Importance = widget.LowImportance

	title := canvas.NewText("UnderMine", customtheme.ColorTextMain)
	title.TextSize = 22
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.Alignment = fyne.TextAlignCenter

	subtitle := canvas.NewText("Авторизация", customtheme.ColorTextMuted)
	subtitle.TextSize = 13
	subtitle.Alignment = fyne.TextAlignCenter

	cardBg := canvas.NewRectangle(customtheme.ColorSidebarBg)
	cardBg.CornerRadius = 12

	formContent := container.NewVBox(
		container.NewCenter(title),
		container.NewCenter(subtitle),
		widget.NewSeparator(),
		v.userEntry,
		v.emailEntry,
		v.passEntry,
		v.errorLabel,
		container.NewPadded(v.actionBtn),
		v.toggleBtn,
	)

	paddedCard := container.NewPadded(formContent)
	cardStack := container.NewStack(cardBg, paddedCard)

	cardContainer := container.NewCenter(
		container.NewGridWrap(fyne.NewSize(380, 420), cardStack),
	)

	windowBg := canvas.NewRectangle(customtheme.ColorWindowBg)
	return container.NewStack(windowBg, cardContainer)
}

// SetError отображает ошибку в интерфейсе
func (v *AuthView) SetError(msg string) {
	if msg == "" {
		v.ClearError()
		return
	}
	v.errorLabel.SetText("⚠️ " + msg)
	v.errorLabel.Show()
	v.actionBtn.Enable()
	v.errorLabel.Refresh()
}

// ClearError скрывает блок ошибок
func (v *AuthView) ClearError() {
	v.errorLabel.SetText("")
	v.errorLabel.Hide()
	v.errorLabel.Refresh()
}

// SetLoading блокирует кнопку действия при отправке запроса
func (v *AuthView) SetLoading(loading bool) {
	if loading {
		v.actionBtn.Disable()
	} else {
		v.actionBtn.Enable()
	}
}

func (v *AuthView) toggleMode() {
	v.isRegister = !v.isRegister
	v.ClearError()

	if v.isRegister {
		v.emailEntry.Show()
		v.actionBtn.SetText("Зарегистрироваться")
		v.toggleBtn.SetText("Уже есть аккаунт? Войти")
	} else {
		v.emailEntry.Hide()
		v.actionBtn.SetText("Войти")
		v.toggleBtn.SetText("Создать аккаунт")
	}
}

func (v *AuthView) handleSubmit() {
	user := strings.TrimSpace(v.userEntry.Text)
	pass := strings.TrimSpace(v.passEntry.Text)
	email := strings.TrimSpace(v.emailEntry.Text)

	if user == "" || pass == "" {
		v.SetError("Заполните обязательные поля")
		return
	}
	if v.isRegister && email == "" {
		v.SetError("Укажите email для регистрации")
		return
	}

	v.ClearError()
	v.SetLoading(true)

	if v.isRegister {
		if v.OnRegister != nil {
			v.OnRegister(user, email, pass)
		}
	} else {
		if v.OnLogin != nil {
			v.OnLogin(user, pass)
		}
	}
}
