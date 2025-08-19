package handler

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/sunthewhat/easy-cert-api/type/response"
)

func HandleNotFound(c *fiber.Ctx) error {
	return c.Status(fiber.StatusNotFound).JSON(
		response.Error(fmt.Sprintf("%s %s not found", c.Method(), c.Path())),
	)
}
