package handler

import (
	"fmt"

	"github.com/bsthun/gut"
	"github.com/gofiber/fiber/v2"
	"github.com/sunthewhat/secure-docs-api/type/response"
)

func HandleNotFound(c *fiber.Ctx) error {
	return c.Status(fiber.StatusNotFound).JSON(
		&response.ErrorResponse{
			Success: false,
			Message: gut.Ptr(fmt.Sprintf("%s %s not found", c.Method(), c.Path())),
		},
	)
}
