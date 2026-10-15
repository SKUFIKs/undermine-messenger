package api

import (
	"encoding/hex"
	"encoding/json"
	"net"
	"net/http"
	"strings"

	"undermine.ru/undermine-server/internal/auth"
)

type RegisterRequest struct {
	Username  string `json:"username"`
	Email     string `json:"email"`
	Password  string `json:"password"`
	PublicKey string `json:"public_key"`
}

type AuthResponse struct {
	Token    string `json:"token"`
	UserID   string `json:"user_id"`
	Username string `json:"username"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type AuthHandler struct {
	authService *auth.AuthService
}

func NewAuthHandler(authService *auth.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

func (h *AuthHandler) HandleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Некорректный JSON", http.StatusBadRequest)
		return
	}

	pubKeyBytes, err := hex.DecodeString(req.PublicKey)
	if err != nil || len(pubKeyBytes) != 32 {
		http.Error(w, "Неверный формат публичного ключа (ожидается 64 hex-символа)", http.StatusBadRequest)
		return
	}

	var pubKey [32]byte
	copy(pubKey[:], pubKeyBytes)

	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	deviceInfo := r.UserAgent()

	token, user, err := h.authService.Register(req.Username, req.Email, req.Password, pubKey, ip, deviceInfo)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(AuthResponse{
		Token:    token,
		UserID:   hex.EncodeToString(user.ID[:]),
		Username: user.Username,
	})
}

func (h *AuthHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Некорректный JSON", http.StatusBadRequest)
		return
	}

	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	deviceInfo := r.UserAgent()

	token, user, err := h.authService.Login(req.Username, req.Password, ip, deviceInfo)
	if err != nil {
		if strings.Contains(err.Error(), "IP заблокирован") {
			http.Error(w, err.Error(), http.StatusTooManyRequests)
			return
		}
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(AuthResponse{
		Token:    token,
		UserID:   hex.EncodeToString(user.ID[:]),
		Username: user.Username,
	})
}

type UserSearchResponse struct {
	UserID    string `json:"user_id"`
	Username  string `json:"username"`
	PublicKey string `json:"public_key"`
}

func (h *AuthHandler) HandleSearchUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	rawToken := extractBearerToken(r)
	if rawToken == "" {
		http.Error(w, "Отсутствует заголовок Authorization", http.StatusUnauthorized)
		return
	}

	_, err := h.authService.ValidateToken(rawToken)
	if err != nil {
		http.Error(w, "Недействительный токен авторизации", http.StatusUnauthorized)
		return
	}

	username := strings.TrimSpace(r.URL.Query().Get("username"))
	if username == "" {
		http.Error(w, "Параметр username обязателен", http.StatusBadRequest)
		return
	}

	user, pubKey, err := h.authService.SearchUser(username)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(UserSearchResponse{
		UserID:    hex.EncodeToString(user.ID[:]),
		Username:  user.Username,
		PublicKey: hex.EncodeToString(pubKey[:]),
	})
}

func (h *AuthHandler) HandleDirectory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	entries, err := h.authService.Directory()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	list := make([]UserSearchResponse, 0, len(entries))
	for _, e := range entries {
		list = append(list, UserSearchResponse{
			UserID:    hex.EncodeToString(e.UserID[:]),
			Username:  e.Username,
			PublicKey: hex.EncodeToString(e.PublicKey[:]),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}

func (h *AuthHandler) HandleUpdateKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	rawToken := extractBearerToken(r)
	if rawToken == "" {
		http.Error(w, "Отсутствует заголовок Authorization", http.StatusUnauthorized)
		return
	}

	userID, err := h.authService.ValidateToken(rawToken)
	if err != nil {
		http.Error(w, "Недействительный токен авторизации", http.StatusUnauthorized)
		return
	}

	var req struct {
		PublicKey string `json:"public_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Некорректный JSON", http.StatusBadRequest)
		return
	}

	pubKeyBytes, err := hex.DecodeString(req.PublicKey)
	if err != nil || len(pubKeyBytes) != 32 {
		http.Error(w, "Неверный формат публичного ключа (ожидается 64 hex-символа)", http.StatusBadRequest)
		return
	}

	var pubKey [32]byte
	copy(pubKey[:], pubKeyBytes)

	err = h.authService.UpdatePublicKey(userID, pubKey)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func extractBearerToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimPrefix(authHeader, "Bearer ")
	}
	return ""
}
