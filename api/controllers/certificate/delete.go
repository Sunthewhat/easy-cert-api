package certificate_controller

import (
	"github.com/gofiber/fiber/v2"
	certificatemodel "github.com/sunthewhat/easy-cert-api/api/model/certificateModel"
	"github.com/sunthewhat/easy-cert-api/type/response"
)

func Delete(c *fiber.Ctx) error {
	certId := c.Params("certId")

	cert, err := certificatemodel.Delete(certId)

	if err != nil {
		return response.SendInternalError(c, err)
	}

	return response.SendSuccess(c, "Certificate Deleted", cert)
}
