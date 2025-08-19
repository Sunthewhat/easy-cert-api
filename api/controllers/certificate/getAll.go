package certificate_controller

import (
	"github.com/gofiber/fiber/v2"
	certificatemodel "github.com/sunthewhat/easy-cert-api/api/model/certificateModel"
	"github.com/sunthewhat/easy-cert-api/type/response"
)

func GetAll(c *fiber.Ctx) error {
	certificates, err := certificatemodel.GetAll()

	if err != nil {
		return response.SendInternalError(c, err)
	}

	return response.SendSuccess(c, "Certificate fetched", certificates)
}
