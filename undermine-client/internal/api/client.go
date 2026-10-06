package api

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client представляет HTTP клиент для взаимодействия с сервером
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

// NewClient создает новый API клиент
func NewClient(baseURL string) *Client {
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// RegisterRequest представляет запрос на регистрацию
type RegisterRequest struct {
	Username  string `json:"username"`
	Email     string `json:"email"`
	Password  string `json:"password"`
	PublicKey string `json:"public_key"`
}

// LoginRequest представляет запрос на вход
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// AuthResponse представляет ответ сервера на авторизацию
type AuthResponse struct {
	Token    string `json:"token"`
	UserID   string `json:"user_id"`
	Username string `json:"username"`
}

// UserSearchResponse представляет ответ на поиск пользователя
type UserSearchResponse struct {
	UserID    string `json:"user_id"`
	Username  string `json:"username"`
	PublicKey string `json:"public_key"`
}

// Register выполняет регистрацию нового пользователя
func (c *Client) Register(username, email, password string, publicKey [32]byte) (*AuthResponse, error) {
	reqBody := RegisterRequest{
		Username:  username,
		Email:     email,
		Password:  password,
		PublicKey: hex.EncodeToString(publicKey[:]),
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("ошибка сериализации запроса: %w", err)
	}

	resp, err := c.HTTPClient.Post(
		c.BaseURL+"/api/v1/auth/register",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения ответа: %w", err)
	}

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("ошибка регистрации: %s", string(body))
	}

	var authResp AuthResponse
	if err := json.Unmarshal(body, &authResp); err != nil {
		return nil, fmt.Errorf("ошибка парсинга ответа: %w", err)
	}

	return &authResp, nil
}

// Login выполняет вход пользователя
func (c *Client) Login(username, password string) (*AuthResponse, error) {
	reqBody := LoginRequest{
		Username: username,
		Password: password,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("ошибка сериализации запроса: %w", err)
	}

	resp, err := c.HTTPClient.Post(
		c.BaseURL+"/api/v1/auth/login",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения ответа: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ошибка входа: %s", string(body))
	}

	var authResp AuthResponse
	if err := json.Unmarshal(body, &authResp); err != nil {
		return nil, fmt.Errorf("ошибка парсинга ответа: %w", err)
	}

	return &authResp, nil
}

// SearchUser выполняет поиск пользователя по имени
func (c *Client) SearchUser(token, username string) (*UserSearchResponse, error) {
	url := fmt.Sprintf("%s/api/v1/users/search?username=%s", c.BaseURL, username)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания запроса: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения ответа: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ошибка поиска: %s", string(body))
	}

	var searchResp UserSearchResponse
	if err := json.Unmarshal(body, &searchResp); err != nil {
		return nil, fmt.Errorf("ошибка парсинга ответа: %w", err)
	}

	return &searchResp, nil
}
