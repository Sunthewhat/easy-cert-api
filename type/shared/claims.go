package shared

import "github.com/golang-jwt/jwt/v4"

type UserClaims struct {
	UserId *string `json:"userId"`
	jwt.RegisteredClaims
}

func (u *UserClaims) Valid() error {
	return nil
}
