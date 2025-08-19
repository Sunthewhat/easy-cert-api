package userModel

import (
	"errors"
	"log/slog"

	"github.com/sunthewhat/easy-cert-api/common"
	"github.com/sunthewhat/easy-cert-api/type/shared/model"
	"gorm.io/gorm"
)

func GetByUsername(username string) (*model.User, error) {
	user, queryErr := common.Gorm.User.Where(common.Gorm.User.Username.Eq(username)).First()

	if queryErr != nil {
		if errors.Is(queryErr, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		slog.Error("User GetByUsername", "error", queryErr, "username", username)
		return nil, queryErr
	}

	return user, nil
}

func CreateNewUser(username string, password string, firstname string, lastname string) (*model.User, error) {
	user := &model.User{
		Username:  username,
		Firstname: firstname,
		Lastname:  lastname,
		Password:  password,
	}

	createErr := common.Gorm.User.Create(user)

	if createErr != nil {
		slog.Error("User CreateNewUser", "error", createErr, "user", user)
		return nil, createErr
	}

	return user, nil
}
