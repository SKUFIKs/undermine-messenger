package main

import (
	"encoding/hex"
	"log"
	"net/http"
	"os"
	"strings"

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
	// Load .env file if present
	loadEnvFile()

	port := getEnv("PORT", "8080")
	host := getEnv("HOST", "0.0.0.0")

	db, err := storage.New("./messenger.db")
	if err != nil {
		log.Fatal("Ошибка БД:", err)
	}

	authService := auth.NewAuthService(db)
	authHandler := api.NewAuthHandler(authService)
	hub := transport.NewHub(db)

	http.HandleFunc("/api/v1/auth/register", authHandler.HandleRegister)
	http.HandleFunc("/api/v1/auth/login", authHandler.HandleLogin)
	http.HandleFunc("/api/v1/auth/key", authHandler.HandleUpdateKey)
	http.HandleFunc("/api/v1/users/search", authHandler.HandleSearchUser)
	http.HandleFunc("/api/v1/users/directory", authHandler.HandleDirectory)

	groupHandler := api.NewGroupHandler(authService)
	http.HandleFunc("/api/v1/groups", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			groupHandler.HandleCreateGroup(w, r)
		case http.MethodGet:
			groupHandler.HandleListGroups(w, r)
		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	})
	http.HandleFunc("/api/v1/groups/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch {
		case strings.HasSuffix(path, "/members") && r.Method == http.MethodGet:
			groupHandler.HandleListGroupMembers(w, r)
		case strings.HasSuffix(path, "/members") && r.Method == http.MethodPost:
			groupHandler.HandleAddGroupMember(w, r)
		case strings.HasSuffix(path, "/members/") && r.Method == http.MethodDelete:
			groupHandler.HandleRemoveGroupMember(w, r)
		case strings.HasSuffix(path, "/keys") && r.Method == http.MethodGet:
			groupHandler.HandleGetGroupKey(w, r)
		case strings.HasSuffix(path, "/keys/") && r.Method == http.MethodPost:
			groupHandler.HandleSaveGroupKey(w, r)
		case r.Method == http.MethodDelete:
			groupHandler.HandleDeleteGroup(w, r)
		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	})

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
	log.Printf("UnderMine Go Server running on %s:%s", host, port)
	log.Println("Auth routes: /api/v1/auth/register, /api/v1/auth/login")
	log.Println("User search route: /api/v1/users/search")
	log.Println("WebSocket endpoint: /ws")
	log.Println("--------------------------------------------------")

	if err := http.ListenAndServe(host+":"+port, corsMiddleware(http.DefaultServeMux)); err != nil {
		log.Fatal("Ошибка сервера:", err)
	}
}

// corsMiddleware allows browser clients (Vite dev server, Electron) to call the API
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// loadEnvFile reads .env file from the current directory and sets environment variables
func loadEnvFile() {
	data, err := os.ReadFile(".env")
	if err != nil {
		return // No .env file, use defaults
	}

	for _, line := range splitLines(string(data)) {
		line = trimSpace(line)
		if line == "" || line[0] == '#' {
			continue
		}
		eq := indexByte(line, '=')
		if eq < 0 {
			continue
		}
		key := trimSpace(line[:eq])
		value := trimSpace(line[eq+1:])
		// Remove quotes if present
		if len(value) >= 2 && (value[0] == '"' || value[0] == '\'') {
			value = value[1 : len(value)-1]
		}
		os.Setenv(key, value)
	}
}

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	lines = append(lines, s[start:])
	return lines
}

func trimSpace(s string) string {
	start := 0
	for start < len(s) && (s[start] == ' ' || s[start] == '\t' || s[start] == '\r') {
		start++
	}
	end := len(s)
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}

func indexByte(s string, b byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == b {
			return i
		}
	}
	return -1
}
