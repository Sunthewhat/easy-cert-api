package common

import (
	"github.com/sunthewhat/secure-docs-api/type/shared"
	"github.com/sunthewhat/secure-docs-api/type/shared/query"
	"gorm.io/gorm"
)

var Config *shared.Config
var Database *gorm.DB
var Query *query.Query
