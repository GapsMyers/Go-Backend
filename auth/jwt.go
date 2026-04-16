package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Claims represents JWT claims used by this backend.
type Claims struct {
	Email string `json:"email"`
	jwt.RegisteredClaims
}

// JWTService encapsulates token generation and validation.
type JWTService struct {
	secretKey []byte
	ttl       time.Duration
}

func NewJWTService(secret string, expireMinutes int) *JWTService {
	return &JWTService{
		secretKey: []byte(secret),
		ttl:       time.Duration(expireMinutes) * time.Minute,
	}
}

func (s *JWTService) GenerateToken(userID uuid.UUID, email string) (string, error) {
	now := time.Now()
	claims := Claims{
		Email: email,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.ttl)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secretKey)
}

func (s *JWTService) ValidateToken(tokenString string) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.secretKey, nil
	})
	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	if claims.Subject == "" {
		return nil, errors.New("token subject is missing")
	}

	return claims, nil
}

func (s *JWTService) ExpiresInSeconds() int64 {
	return int64(s.ttl.Seconds())
}
