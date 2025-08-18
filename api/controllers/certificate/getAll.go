package certificate_controller

import (
	"github.com/gofiber/fiber/v2"
	certificatemodel "github.com/sunthewhat/secure-docs-api/api/model/certificateModel"
	"github.com/sunthewhat/secure-docs-api/type/response"
)

func GetAll(c *fiber.Ctx) error {
	certificates, err := certificatemodel.GetAll()

	if err != nil {
		return response.SendInternalError(c, err)
	}

	return response.SendSuccess(c, "Certificate fetched", certificates)
}
