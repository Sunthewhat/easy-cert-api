package api

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/sunthewhat/secure-docs-api/api/handler"
	"github.com/sunthewhat/secure-docs-api/api/middleware"
	"github.com/sunthewhat/secure-docs-api/api/routes"
	"github.com/sunthewhat/secure-docs-api/common"
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

	err := app.Listen(*common.Config.Port)

	if err != nil {
		log.Fatal("Failed to start server ", err)
	}
}
