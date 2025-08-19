package util

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/sunthewhat/easy-cert-api/common"
	"github.com/sunthewhat/easy-cert-api/type/shared"
)

func GenerateAuthToken(id string) (string, error) {
	expirationTime := time.Now().Add(time.Hour * 24 * 2) // 2 days

	claims := &shared.UserClaims{
		UserId: &id,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString([]byte(*common.Config.JWTSecret))
}

func DecodeAuthToken(tokenString string) (*shared.UserClaims, error) {
	claims := &shared.UserClaims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(*common.Config.JWTSecret), nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}
