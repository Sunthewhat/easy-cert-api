package util

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/sunthewhat/easy-cert-api/common"
	"github.com/sunthewhat/easy-cert-api/type/shared"
	"golang.org/x/crypto/bcrypt"
)

type IInternalAuthService interface {
	Login(username string, password string) (*shared.SsoTokenType, error)
	Refresh(token string) (*shared.SsoTokenType, error)
	Verify(token string) (*shared.SsoVerifyType, error)
	Decode(token string) (*shared.SsoJwtPayload, error)
}

type InternalAuthService struct{}

func NewInternalAuthService() IAuthService {
	return &InternalAuthService{}
}

func (s *InternalAuthService) Login(username string, password string) (*shared.SsoTokenType, error) {
	// Query user from database
	user, err := common.Gorm.User.Where(common.Gorm.User.Username.Eq(username)).First()
	if err != nil {
		return nil, errors.New("invalid user credentials")
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, errors.New("invalid user credentials")
	}

	// Generate JWT tokens
	accessToken, err := s.generateToken(user.Username, user.Firstname, user.Lastname, 15*time.Minute)
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.generateToken(user.Username, user.Firstname, user.Lastname, 7*24*time.Hour)
	if err != nil {
		return nil, err
	}

	return &shared.SsoTokenType{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    900, // 15 minutes in seconds
		TokenType:    "Bearer",
	}, nil
}

func (s *InternalAuthService) Refresh(token string) (*shared.SsoTokenType, error) {
	// Decode the refresh token
	jwtPayload, err := s.Decode(token)
	if err != nil {
		return nil, errors.New("invalid user token")
	}

	// Verify user still exists
	user, err := common.Gorm.User.Where(common.Gorm.User.Username.Eq(jwtPayload.PreferredUsername)).First()
	if err != nil {
		return nil, errors.New("invalid user token")
	}

	// Generate new tokens
	accessToken, err := s.generateToken(user.Username, user.Firstname, user.Lastname, 15*time.Minute)
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.generateToken(user.Username, user.Firstname, user.Lastname, 7*24*time.Hour)
	if err != nil {
		return nil, err
	}

	return &shared.SsoTokenType{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    900, // 15 minutes in seconds
		TokenType:    "Bearer",
	}, nil
}

func (s *InternalAuthService) Verify(token string) (*shared.SsoVerifyType, error) {
	jwtPayload, err := s.Decode(token)
	if err != nil {
		return &shared.SsoVerifyType{
			Active: false,
		}, nil
	}

	// Check if token is expired (Exp is int, time.Now().Unix() is int64)
	if int64(jwtPayload.Exp) < time.Now().Unix() {
		return &shared.SsoVerifyType{
			Active: false,
		}, nil
	}

	return &shared.SsoVerifyType{
		Active: true,
	}, nil
}

func (s *InternalAuthService) Decode(token string) (*shared.SsoJwtPayload, error) {
	// Parse JWT token
	parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(*common.Config.JWTSecret), nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse JWT token: %v", err)
	}

	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	if !ok || !parsedToken.Valid {
		return nil, fmt.Errorf("invalid JWT token")
	}

	// Extract claims
	username, _ := claims["preferred_username"].(string)
	givenName, _ := claims["given_name"].(string)
	familyName, _ := claims["family_name"].(string)
	email, _ := claims["email"].(string)
	exp, _ := claims["exp"].(float64)

	return &shared.SsoJwtPayload{
		PreferredUsername: username,
		GivenName:         givenName,
		FamilyName:        familyName,
		Email:             email,
		Exp:               int(exp),
	}, nil
}

func (s *InternalAuthService) generateToken(username, firstname, lastname string, expiration time.Duration) (string, error) {
	claims := jwt.MapClaims{
		"preferred_username": username,
		"given_name":         firstname,
		"family_name":        lastname,
		"email":              username, // Use username as email for internal auth
		"exp":                time.Now().Add(expiration).Unix(),
		"iat":                time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(*common.Config.JWTSecret))
}
