package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var ErrInvalidToken = errors.New("Invalid token")

type JWT struct {
	Secret []byte
}

type Claims struct {
	UserID   int64
	Username string
}

func NewJWT(secret string) *JWT {
	return &JWT{Secret: []byte(secret)}
}

func (j *JWT) Sign(userID int64, username string) (string, error) {
	claims := jwt.MapClaims{
		"uid": userID,
		"sub": username,
		"exp": time.Now().Add(24 * time.Hour).Unix(),
		"iat": time.Now().Unix(),
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString(j.Secret)
}

func (j *JWT) Verify(tokenStr string) (Claims, error) {
	t, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return j.Secret, nil
	})
	if err != nil || !t.Valid {
		return Claims{}, ErrInvalidToken
	}
	claims, ok := t.Claims.(jwt.MapClaims)
	if !ok {
		return Claims{}, ErrInvalidToken
	}
	sub, ok := claims["sub"].(string)
	if !ok || sub == "" {
		return Claims{}, ErrInvalidToken
	}
	uidFloat, ok := claims["uid"].(float64)
	if !ok || uidFloat <= 0 {
		return Claims{}, ErrInvalidToken
	}
	return Claims{UserID: int64(uidFloat), Username: sub}, nil
}
