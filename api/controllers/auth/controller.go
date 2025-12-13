package auth_controller

import "github.com/sunthewhat/easy-cert-api/common/util"

type AuthController struct {
	authService util.IAuthService
}

func NewAuthController(authService util.IAuthService) *AuthController {
	return &AuthController{authService: authService}
}
