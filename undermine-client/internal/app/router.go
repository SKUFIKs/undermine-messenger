package app

import (
	"fyne.io/fyne/v2"

	"undermine.ru/undermine-messenger/internal/ui/views"
)

// Router управляет отображением экранов в едином окне
type Router struct {
	window   fyne.Window
	AuthView *views.AuthView
	ChatView *views.ChatView
}

// NewRouter создает навигатор экранов
func NewRouter(window fyne.Window, authView *views.AuthView, chatView *views.ChatView) *Router {
	return &Router{
		window:   window,
		AuthView: authView,
		ChatView: chatView,
	}
}

// ShowAuth переключает сцену на экран авторизации
func (r *Router) ShowAuth() {
	r.window.SetContent(r.AuthView.BuildUI())
}

// ShowChat переключает сцену на главный экран чатов
func (r *Router) ShowChat() {
	r.window.SetContent(r.ChatView.BuildUI())
}
