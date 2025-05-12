package handler

import (
	"errors"

	"github.com/bsthun/gut"
	"github.com/gofiber/fiber/v2"
	"github.com/sunthewhat/secure-docs-api/type/response"
)

func HandleError(c *fiber.Ctx, err error) error {
	var fiberErr *fiber.Error
	if errors.As(err, &fiberErr) {
		return c.Status(fiberErr.Code).JSON(
			response.Error(&fiberErr.Message),
		)
	}

	var respErr *gut.ErrorInstance
	if errors.As(err, &respErr) {
		return c.Status(fiber.StatusBadRequest).JSON(
			response.Error(gut.Ptr(respErr.Error())),
		)
	}

	return c.Status(fiber.StatusInternalServerError).JSON(
		response.Error(gut.Ptr(err.Error())),
	)
}
