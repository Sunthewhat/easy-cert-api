package util

import "github.com/sunthewhat/easy-cert-api/type/shared"

// IAuthService is a unified interface for both SSO and internal authentication
type IAuthService interface {
	Login(username string, password string) (*shared.SsoTokenType, error)
	Refresh(token string) (*shared.SsoTokenType, error)
	Verify(token string) (*shared.SsoVerifyType, error)
	Decode(token string) (*shared.SsoJwtPayload, error)
}
