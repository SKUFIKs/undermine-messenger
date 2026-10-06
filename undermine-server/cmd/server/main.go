package main

import (
	"encoding/hex"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	"undermine.ru/undermine-server/internal/api"
	"undermine.ru/undermine-server/internal/auth"
	"undermine.ru/undermine-server/internal/storage"
	"undermine.ru/undermine-server/internal/transport"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func main() {
	db, err := storage.New("./messenger.db")
	if err != nil {
		log.Fatal("Ошибка БД:", err)
	}

	authService := auth.NewAuthService(db)
	authHandler := api.NewAuthHandler(authService)
	hub := transport.NewHub(db)

	http.HandleFunc("/api/v1/auth/register", authHandler.HandleRegister)
	http.HandleFunc("/api/v1/auth/login", authHandler.HandleLogin)
	http.HandleFunc("/api/v1/users/search", authHandler.HandleSearchUser)

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get("token")
		userIDStr := r.URL.Query().Get("user_id")

		var userID [16]byte

		if token != "" {
			validUser, err := authService.ValidateToken(token)
			if err != nil {
				http.Error(w, "Недействительный токен: "+err.Error(), http.StatusUnauthorized)
				return
			}
			userID = validUser
			log.Printf("[WS] Подключение по токену: userID=%x", userID)
		} else if userIDStr != "" {
			bytes, err := hex.DecodeString(userIDStr)
			if err != nil || len(bytes) != 16 {
				http.Error(w, "Неверный формат user_id (ожидается 32 hex символа)", http.StatusBadRequest)
				return
			}
			copy(userID[:], bytes)
			log.Printf("[WS] Подключение по userID: userID=%x", userID)
		} else {
			http.Error(w, "Требуется авторизация (параметр token или user_id)", http.StatusUnauthorized)
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("Ошибка WebSocket Upgrade:", err)
			return
		}

		log.Printf("[WS] WebSocket подключение установлено: userID=%x", userID)

		client := transport.NewClient(userID, conn, hub)
		hub.Register(userID, client)

		pendingList, err := db.GetPendingMessages(userID, 100)
		if err == nil {
			log.Printf("[WS] Загрузка офлайн-сообщений для %x: %d шт", userID, len(pendingList))
			for _, msg := range pendingList {
				client.SendChannel <- msg
			}
		}

		go client.WritePump()
		client.ReadPump()
	})

	log.Println("--------------------------------------------------")
	log.Println("UnderMine Go Server running on 0.0.0.0:8080")
	log.Println("Auth routes: /api/v1/auth/register, /api/v1/auth/login")
	log.Println("User search route: /api/v1/users/search")
	log.Println("WebSocket endpoint: /ws")
	log.Println("--------------------------------------------------")

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal("Ошибка сервера:", err)
	}
}
