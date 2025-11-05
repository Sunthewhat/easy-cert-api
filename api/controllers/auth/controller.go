package auth_controller

import "github.com/sunthewhat/easy-cert-api/common/util"

type AuthController struct {
	ssoService util.ISSOService
}

func NewAuthController(sso util.ISSOService) *AuthController {
	return &AuthController{ssoService: sso}
}
