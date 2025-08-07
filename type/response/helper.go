package response

import "github.com/gofiber/fiber/v2"

func SendSuccess(c *fiber.Ctx, msg string, data ...any) error {
	return c.Status(fiber.StatusOK).JSON(Success(msg, data...))
}

func SendUnauthorized(c *fiber.Ctx, msg string) error {
	return c.Status(fiber.StatusUnauthorized).JSON(Error(msg))
}

func SendFailed(c *fiber.Ctx, msg string) error {
	return c.Status(fiber.StatusBadRequest).JSON(Error(msg))
}

func SendError(c *fiber.Ctx, msg string) error {
	return c.Status(fiber.StatusInternalServerError).JSON(Error(msg))
}

func SendInternalError(c *fiber.Ctx, err error) error {
	return c.Status(fiber.StatusInternalServerError).JSON(Error(err.Error()))
}
