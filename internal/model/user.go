package model

import "time"

type Role string

const (
	RoleAdmin  Role = "admin"
	RoleViewer Role = "viewer"
)

type User struct {
	ID          string     `json:"id"`
	Username    string     `json:"username"`
	Role        Role       `json:"role"`
	IsEnabled   bool       `json:"is_enabled"`
	CreatedAt   time.Time  `json:"created_at"`
	LastLoginAt *time.Time `json:"last_login_at,omitempty"`
	// Libraries this user can access (populated from user_library_access)
	LibraryIDs []string `json:"library_ids,omitempty"`
}

type PlaybackProgress struct {
	UserID       string    `json:"user_id"`
	ItemID       string    `json:"item_id"`
	PositionMs   int64     `json:"position_ms"`
	PlayedPct    float64   `json:"played_pct"`
	IsWatched    bool      `json:"is_watched"`
	LastPlayedAt time.Time `json:"last_played_at"`
}

type APIToken struct {
	ID          string     `json:"id"`
	UserID      string     `json:"user_id"`
	Name        string     `json:"name"`
	Scope       string     `json:"scope"` // read|full
	CreatedAt   time.Time  `json:"created_at"`
	LastUsedAt  *time.Time `json:"last_used_at,omitempty"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	// RawToken is only populated at creation time, never stored
	RawToken string `json:"token,omitempty"`
}
