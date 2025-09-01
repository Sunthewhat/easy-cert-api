package util

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/sunthewhat/easy-cert-api/common"
	"github.com/sunthewhat/easy-cert-api/type/shared"
)

func LoginSSO(username string, password string) (*shared.SsoTokenType, error) {
	tokenURL := fmt.Sprintf("%s/protocol/openid-connect/token", *common.Config.SsoIssuerUrl)

	data := url.Values{}
	data.Set("username", username)
	data.Set("password", password)
	data.Set("client_id", *common.Config.SsoClient)
	data.Set("client_secret", *common.Config.SsoSecret)
	data.Set("grant_type", "password")
	data.Set("scope", "openid")

	req, err := http.NewRequest("POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var ssoResponse shared.SsoTokenType
	if err := json.NewDecoder(resp.Body).Decode(&ssoResponse); err != nil {
		return nil, err
	}

	return &ssoResponse, nil
}

func RefreshSSO(token string) (*shared.SsoTokenType, error) {
	tokenURL := fmt.Sprintf("%s/protocol/openid-connect/token", *common.Config.SsoIssuerUrl)

	data := url.Values{}
	data.Set("client_id", *common.Config.SsoClient)
	data.Set("client_secret", *common.Config.SsoSecret)
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", token)

	req, err := http.NewRequest("POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var ssoResponse shared.SsoTokenType
	if err := json.NewDecoder(resp.Body).Decode(&ssoResponse); err != nil {
		return nil, err
	}

	return &ssoResponse, nil
}

func VerifySSO(token string) (*shared.SsoVerifyType, error) {
	verifyURL := fmt.Sprintf("%s/protocol/openid-connect/token/introspect", *common.Config.SsoIssuerUrl)

	data := url.Values{}
	data.Set("client_id", *common.Config.SsoClient)
	data.Set("client_secret", *common.Config.SsoSecret)
	data.Set("token", token)

	req, err := http.NewRequest("POST", verifyURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	var verifyResponse shared.SsoVerifyType
	if err := json.NewDecoder(resp.Body).Decode(&verifyResponse); err != nil {
		return nil, err
	}

	return &verifyResponse, nil
}

func DecodeJWTToken(token string) (*shared.SsoJwtPayload, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid JWT token format")
	}

	payload := parts[1]

	// Add padding if needed for base64 decoding
	for len(payload)%4 != 0 {
		payload += "="
	}

	decoded, err := base64.URLEncoding.DecodeString(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to decode JWT payload: %v", err)
	}

	var jwtPayload shared.SsoJwtPayload
	if err := json.Unmarshal(decoded, &jwtPayload); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JWT payload: %v", err)
	}

	return &jwtPayload, nil
}
