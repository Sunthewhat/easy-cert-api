package middleware

import (
	"github.com/gofiber/fiber/v2"
	jwtware "github.com/gofiber/jwt/v3"
	"github.com/sunthewhat/easy-cert-api/common"
	"github.com/sunthewhat/easy-cert-api/type/response"
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
			return response.SendFailed(c, "JWT validation failure")
		},
	}
	return jwtware.New(conf)
}
