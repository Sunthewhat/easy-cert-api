package middleware

import (
	"github.com/bsthun/gut"
	"github.com/gofiber/fiber/v2"
	jwtware "github.com/gofiber/jwt/v3"
	"github.com/sunthewhat/easy-cert-api/common"
	"github.com/sunthewhat/easy-cert-api/type/shared"
)

func Jwt() fiber.Handler {
	conf := jwtware.Config{
		SigningKey:  []byte(*common.Config.JWTSecret),
		TokenLookup: "header:Authorization",
		AuthScheme:  "Bearer",
		ContextKey:  "auth",
		Claims:      new(shared.UserClaims),
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return gut.Err(false, "JWT validation failure", err)
		},
	}
	return jwtware.New(conf)
}
