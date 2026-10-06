package domain

import "time"

type User struct {
	ID           [16]byte  `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

type Session struct {
	TokenHash [32]byte  `json:"token_hash"`
	UserID    [16]byte  `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
}
