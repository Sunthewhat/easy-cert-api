package util

import (
	"github.com/go-playground/validator/v10"
)

var validate *validator.Validate

func init() {
	validate = validator.New()
}

// ValidateStruct validates a struct using validator tags
func ValidateStruct(s interface{}) error {
	return validate.Struct(s)
}

// GetValidationErrors formats validation errors into readable messages
func GetValidationErrors(err error) []string {
	var errors []string
	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		for _, fieldError := range validationErrors {
			switch fieldError.Tag() {
			case "required":
				errors = append(errors, fieldError.Field()+" is required")
			case "email":
				errors = append(errors, fieldError.Field()+" must be a valid email")
			case "min":
				errors = append(errors, fieldError.Field()+" must be at least "+fieldError.Param()+" characters")
			case "max":
				errors = append(errors, fieldError.Field()+" must be at most "+fieldError.Param()+" characters")
			default:
				errors = append(errors, fieldError.Field()+" is invalid")
			}
		}
	}
	return errors
}