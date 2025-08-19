package handler

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/sunthewhat/easy-cert-api/type/response"
)

func HandleError(c *fiber.Ctx, err error) error {
	var fiberErr *fiber.Error
	if errors.As(err, &fiberErr) {
		return c.Status(fiberErr.Code).JSON(
			response.Error(fiberErr.Message),
		)
	}

	// Handle custom errors (can be expanded as needed)
	// For now, just treat all non-fiber errors as internal server errors

	return c.Status(fiber.StatusInternalServerError).JSON(
		response.Error(err.Error()),
	)
}
