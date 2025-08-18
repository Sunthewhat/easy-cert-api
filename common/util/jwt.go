package util

import (
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/sunthewhat/secure-docs-api/common"
	"github.com/sunthewhat/secure-docs-api/type/shared"
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
