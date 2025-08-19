package api

import (
	"log/slog"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/sunthewhat/easy-cert-api/api/handler"
	"github.com/sunthewhat/easy-cert-api/api/middleware"
	"github.com/sunthewhat/easy-cert-api/api/routes"
	"github.com/sunthewhat/easy-cert-api/common"
)

func InitFiber() {
	cfg := fiber.Config{
		AppName:       "securedocs api",
		ErrorHandler:  handler.HandleError,
		Prefork:       false,
		StrictRouting: true,
		Network:       fiber.NetworkTCP,
	}
	app := fiber.New(cfg)

	app.Use(logger.New())
	app.Use(middleware.Recover())
	app.Use(middleware.Cors())

	routes.Init(app)

	app.Use(handler.HandleNotFound)

	slog.Info("Starting server", "port", *common.Config.Port)
	err := app.Listen(*common.Config.Port)

	if err != nil {
		slog.Error("Failed to start server", "error", err)
		os.Exit(1)
	}
}
