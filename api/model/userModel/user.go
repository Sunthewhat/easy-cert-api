package userModel

import (
	"errors"

	"github.com/bsthun/gut"
	"github.com/sunthewhat/secure-docs-api/common"
	"github.com/sunthewhat/secure-docs-api/type/shared/model"
	"gorm.io/gorm"
)

func GetByUsername(username string) (*model.User, error) {
	user, queryErr := common.Query.User.Where(common.Query.User.Username.Eq(username)).First()

	if queryErr != nil {
		if errors.Is(queryErr, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, gut.Err(false, queryErr.Error())
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

	createErr := common.Query.User.Create(user)

	if createErr != nil {
		return nil, gut.Err(false, createErr.Error())
	}

	return user, nil
}
