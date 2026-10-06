package client

import (
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"undermine.ru/undermine-server/internal/storage"
	"undermine.ru/undermine-server/internal/transport"
)

func TestFullE2EENetworkFlow(t *testing.T) {

	db, err := storage.New(":memory:")
	if err != nil {
		t.Fatalf("Ошибка создания тестовой БД: %v", err)
	}

	hub := transport.NewHub(db)
	upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userIDStr := r.URL.Query().Get("user_id")
		var userID [16]byte
		bytes, _ := hex.DecodeString(userIDStr)
		copy(userID[:], bytes)

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}

		client := transport.NewClient(userID, conn, hub)
		hub.Register(userID, client)

		go client.WritePump()
		client.ReadPump()
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "?user_id="

	var aliceID, bobID [16]byte
	copy(aliceID[:], []byte("alice_id_1234567"))
	copy(bobID[:], []byte("bob_id_123456789"))

	aliceKeys, _ := GenerateKeyPair()
	bobKeys, _ := GenerateKeyPair()

	aliceShared, _ := ComputeSharedKey(aliceKeys.PrivateKey, bobKeys.PublicKey)
	bobShared, _ := ComputeSharedKey(bobKeys.PrivateKey, aliceKeys.PublicKey)

	aliceClient := NewNetworkClient(aliceID)
	aliceClient.SetContactSharedKey(bobID, aliceShared)
	if err := aliceClient.Connect(wsURL + hex.EncodeToString(aliceID[:])); err != nil {
		t.Fatalf("Ошибка подключения Алисы: %v", err)
	}

	bobClient := NewNetworkClient(bobID)
	bobClient.SetContactSharedKey(aliceID, bobShared)
	if err := bobClient.Connect(wsURL + hex.EncodeToString(bobID[:])); err != nil {
		t.Fatalf("Ошибка подключения Боба: %v", err)
	}

	var wg sync.WaitGroup
	wg.Add(1)

	var receivedText string
	go bobClient.Listen(func(msgID [16]byte, text string) {
		receivedText = text
		wg.Done()
	}, func(msgID [16]byte, status byte) {})

	go aliceClient.Listen(func(msgID [16]byte, text string) {}, func(msgID [16]byte, status byte) {})

	sentText := "Привет из интеграционного теста! Сквозное шифрование работает 🚀"
	time.Sleep(50 * time.Millisecond)

	_, err = aliceClient.SendTextMessage(bobID, sentText)
	if err != nil {
		t.Fatalf("Ошибка отправки сообщения: %v", err)
	}

	wg.Wait()

	if receivedText != sentText {
		t.Fatalf("Полученный текст не совпал! Получено: %s", receivedText)
	}

	t.Logf("Интеграционный тест пройден успешно! Боб получил: %s", receivedText)
}
