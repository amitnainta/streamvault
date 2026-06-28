package auth

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	accessTokenTTL  = 15 * time.Minute
	refreshTokenTTL = 30 * 24 * time.Hour
)

type Claims struct {
	UserID     string   `json:"sub"`
	Role       string   `json:"role"`
	LibraryIDs []string `json:"libraries"`
	Scope      string   `json:"scope"` // "full" or "read"
	jwt.RegisteredClaims
}

type TokenPair struct {
	AccessToken  string
	RefreshToken string
}

type JWTService struct {
	secret []byte
}

func NewJWTService(secret []byte) *JWTService {
	return &JWTService{secret: secret}
}

// NewJWTServiceFromDB loads (or auto-generates) the JWT secret from the settings table.
func NewJWTServiceFromDB(db *sql.DB) *JWTService {
	var secretHex string
	err := db.QueryRow(`SELECT value FROM settings WHERE key='jwt_secret'`).Scan(&secretHex)
	if err == nil && len(secretHex) >= 32 {
		secret, decErr := hex.DecodeString(secretHex)
		if decErr == nil {
			return &JWTService{secret: secret}
		}
	}

	// Auto-generate a 256-bit secret and persist it
	buf := make([]byte, 32)
	rand.Read(buf)
	secretHex = hex.EncodeToString(buf)
	db.Exec(
		`INSERT INTO settings(key, value, updated_at) VALUES('jwt_secret',?,CURRENT_TIMESTAMP)
		 ON CONFLICT(key) DO UPDATE SET value=excluded.value`,
		secretHex,
	)
	return &JWTService{secret: buf}
}

func (s *JWTService) Issue(userID, role string, libraryIDs []string) (TokenPair, error) {
	access, err := s.sign(userID, role, libraryIDs, "full", accessTokenTTL)
	if err != nil {
		return TokenPair{}, err
	}
	refresh, err := s.sign(userID, role, libraryIDs, "refresh", refreshTokenTTL)
	if err != nil {
		return TokenPair{}, err
	}
	return TokenPair{AccessToken: access, RefreshToken: refresh}, nil
}

func (s *JWTService) Validate(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.secret, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	return claims, nil
}

func (s *JWTService) sign(userID, role string, libraryIDs []string, scope string, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:     userID,
		Role:       role,
		LibraryIDs: libraryIDs,
		Scope:      scope,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.secret)
}
