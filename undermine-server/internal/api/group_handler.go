package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"undermine.ru/undermine-server/internal/auth"
)

type GroupHandler struct {
	authService *auth.AuthService
}

func NewGroupHandler(authService *auth.AuthService) *GroupHandler {
	return &GroupHandler{authService: authService}
}

type CreateGroupRequest struct {
	Name    string   `json:"name"`
	Members []string `json:"members"` // hex user IDs
}

type CreateGroupResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (h *GroupHandler) HandleCreateGroup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	rawToken := extractBearerToken(r)
	if rawToken == "" {
		http.Error(w, "Отсутствует заголовок Authorization", http.StatusUnauthorized)
		return
	}

	ownerID, err := h.authService.ValidateToken(rawToken)
	if err != nil {
		http.Error(w, "Недействительный токен авторизации", http.StatusUnauthorized)
		return
	}

	var req CreateGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Некорректный JSON", http.StatusBadRequest)
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		http.Error(w, "Название группы обязательно", http.StatusBadRequest)
		return
	}

	group, err := h.authService.CreateGroup(name, ownerID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Add initial members
	for _, memberHex := range req.Members {
		memberBytes, err := hex.DecodeString(memberHex)
		if err != nil || len(memberBytes) != 16 {
			continue
		}
		var memberID [16]byte
		copy(memberID[:], memberBytes)
		_ = h.authService.AddGroupMember(group.ID, memberID)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(CreateGroupResponse{
		ID:   hex.EncodeToString(group.ID[:]),
		Name: group.Name,
	})
}

func (h *GroupHandler) HandleListGroups(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
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

	groups, err := h.authService.ListGroups(userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	type groupJSON struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		OwnerID string `json:"owner_id"`
	}

	list := make([]groupJSON, 0, len(groups))
	for _, g := range groups {
		list = append(list, groupJSON{
			ID:      hex.EncodeToString(g.ID[:]),
			Name:    g.Name,
			OwnerID: hex.EncodeToString(g.OwnerID[:]),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}

func (h *GroupHandler) HandleListGroupMembers(w http.ResponseWriter, r *http.Request) {
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

	groupIDHex := strings.TrimPrefix(r.URL.Path, "/api/v1/groups/")
	groupIDHex = strings.TrimSuffix(groupIDHex, "/members")
	groupIDBytes, err := hex.DecodeString(groupIDHex)
	if err != nil || len(groupIDBytes) != 16 {
		http.Error(w, "Неверный формат group_id", http.StatusBadRequest)
		return
	}

	var groupID [16]byte
	copy(groupID[:], groupIDBytes)

	members, err := h.authService.ListGroupMembers(groupID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	type memberJSON struct {
		UserID    string `json:"user_id"`
		Username  string `json:"username"`
		PublicKey string `json:"public_key"`
	}

	list := make([]memberJSON, 0, len(members))
	for _, m := range members {
		list = append(list, memberJSON{
			UserID:    hex.EncodeToString(m.UserID[:]),
			Username:  m.Username,
			PublicKey: hex.EncodeToString(m.PublicKey[:]),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}

func (h *GroupHandler) HandleAddGroupMember(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
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

	// Extract group_id from path: /api/v1/groups/:id/members
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/groups/")
	path = strings.TrimSuffix(path, "/members")
	groupIDBytes, err := hex.DecodeString(path)
	if err != nil || len(groupIDBytes) != 16 {
		http.Error(w, "Неверный формат group_id", http.StatusBadRequest)
		return
	}

	var groupID [16]byte
	copy(groupID[:], groupIDBytes)

	var req struct {
		UserID string `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Некорректный JSON", http.StatusBadRequest)
		return
	}

	memberBytes, err := hex.DecodeString(req.UserID)
	if err != nil || len(memberBytes) != 16 {
		http.Error(w, "Неверный формат user_id", http.StatusBadRequest)
		return
	}

	var memberID [16]byte
	copy(memberID[:], memberBytes)

	err = h.authService.AddGroupMember(groupID, memberID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *GroupHandler) HandleRemoveGroupMember(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
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

	// Extract group_id and user_id from path: /api/v1/groups/:id/members/:userId
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/groups/")
	path = strings.TrimPrefix(path, "/members/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 {
		http.Error(w, "Неверный формат пути", http.StatusBadRequest)
		return
	}

	groupIDBytes, err := hex.DecodeString(parts[0])
	if err != nil || len(groupIDBytes) != 16 {
		http.Error(w, "Неверный формат group_id", http.StatusBadRequest)
		return
	}

	var groupID [16]byte
	copy(groupID[:], groupIDBytes)

	memberBytes, err := hex.DecodeString(parts[1])
	if err != nil || len(memberBytes) != 16 {
		http.Error(w, "Неверный формат user_id", http.StatusBadRequest)
		return
	}

	var memberID [16]byte
	copy(memberID[:], memberBytes)

	err = h.authService.RemoveGroupMember(groupID, memberID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *GroupHandler) HandleSaveGroupKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	rawToken := extractBearerToken(r)
	if rawToken == "" {
		http.Error(w, "Отсутствует заголовок Authorization", http.StatusUnauthorized)
		return
	}

	senderID, err := h.authService.ValidateToken(rawToken)
	if err != nil {
		http.Error(w, "Недействительный токен авторизации", http.StatusUnauthorized)
		return
	}

	// Extract group_id and target user_id from path: /api/v1/groups/:id/keys/:userId
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/groups/")
	path = strings.TrimPrefix(path, "/keys/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 {
		http.Error(w, "Неверный формат пути", http.StatusBadRequest)
		return
	}

	groupIDBytes, err := hex.DecodeString(parts[0])
	if err != nil || len(groupIDBytes) != 16 {
		http.Error(w, "Неверный формат group_id", http.StatusBadRequest)
		return
	}

	var groupID [16]byte
	copy(groupID[:], groupIDBytes)

	targetBytes, err := hex.DecodeString(parts[1])
	if err != nil || len(targetBytes) != 16 {
		http.Error(w, "Неверный формат user_id", http.StatusBadRequest)
		return
	}

	var targetID [16]byte
	copy(targetID[:], targetBytes)

	var req struct {
		EncryptedKey string `json:"encrypted_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Некорректный JSON", http.StatusBadRequest)
		return
	}

	keyBytes, err := hex.DecodeString(req.EncryptedKey)
	if err != nil || len(keyBytes) == 0 {
		http.Error(w, "Неверный формат encrypted_key", http.StatusBadRequest)
		return
	}

	err = h.authService.SaveGroupKey(groupID, targetID, keyBytes)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Printf("[Group] Ключ группы %x сохранён для пользователя %x (отправитель %x)",
		groupID, targetID, senderID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *GroupHandler) HandleGetGroupKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
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

	// Extract group_id from path: /api/v1/groups/:id/keys
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/groups/")
	path = strings.TrimSuffix(path, "/keys")
	groupIDBytes, err := hex.DecodeString(path)
	if err != nil || len(groupIDBytes) != 16 {
		http.Error(w, "Неверный формат group_id", http.StatusBadRequest)
		return
	}

	var groupID [16]byte
	copy(groupID[:], groupIDBytes)

	key, err := h.authService.GetGroupKey(groupID, userID)
	if err != nil {
		http.Error(w, "Ключ группы не найден", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"encrypted_key": hex.EncodeToString(key),
	})
}

func (h *GroupHandler) HandleDeleteGroup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
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

	groupIDHex := strings.TrimPrefix(r.URL.Path, "/api/v1/groups/")
	groupIDBytes, err := hex.DecodeString(groupIDHex)
	if err != nil || len(groupIDBytes) != 16 {
		http.Error(w, "Неверный формат group_id", http.StatusBadRequest)
		return
	}

	var groupID [16]byte
	copy(groupID[:], groupIDBytes)

	// Only the owner can delete the group; members can leave (remove themselves)
	groups, err := h.authService.ListGroups(userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var isOwner bool
	for _, g := range groups {
		if g.ID == groupID && g.OwnerID == userID {
			isOwner = true
			break
		}
	}

	if isOwner {
		err = h.authService.DeleteGroup(groupID)
	} else {
		err = h.authService.RemoveGroupMember(groupID, userID)
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// generateGroupID creates a new random 16-byte group ID
func generateGroupID() ([16]byte, error) {
	var id [16]byte
	_, err := rand.Read(id[:])
	return id, err
}
