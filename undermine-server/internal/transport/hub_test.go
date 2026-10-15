package transport

import (
	"testing"
)

// Старое соединение не должно «отписывать» уже перерегистрированный новый
// клиент: отложенный Unregister от старого подключения удалял новое из хаба,
// из-за чего сообщения уходили в офлайн-очередь и не доставлялись.
func TestUnregisterDoesNotRemoveNewClient(t *testing.T) {
	hub := NewHub(nil)
	var id [16]byte

	oldClient := &Client{}
	newClient := &Client{}

	hub.Register(id, oldClient)
	hub.Register(id, newClient)

	// Отложенное завершение старого подключения
	hub.Unregister(id, oldClient)

	hub.mu.RLock()
	got, ok := hub.clients[id]
	hub.mu.RUnlock()

	if !ok || got != newClient {
		t.Fatalf("новое подключение удалено из хаба старым Unregister (ok=%v)", ok)
	}

	// Отписка актуального клиента работает как раньше
	hub.Unregister(id, newClient)
	hub.mu.RLock()
	_, ok = hub.clients[id]
	hub.mu.RUnlock()
	if ok {
		t.Fatal("клиент не удалён из хаба после Unregister своего соединения")
	}
}
