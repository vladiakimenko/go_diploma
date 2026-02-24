package validator

import (
	"errors"
	"fmt"

	"github.com/go-playground/validator/v10"
)

var ErrTagValidationFailed = errors.New("tag validation failed")
var ErrCustomValidationFailed = errors.New("custom validation failed")

var validate = validator.New()

func ModelValidate(value any) error {
	// defaults
	if v, ok := value.(interface{ SetDefaults() }); ok {
		v.SetDefaults()
	}

	// tag
	if err := validate.Struct(value); err != nil {
		return fmt.Errorf("%w: %v", ErrTagValidationFailed, err)
	}

	// custom
	if v, ok := value.(interface{ CustomValidate() error }); ok {
		if err := v.CustomValidate(); err != nil {
			return fmt.Errorf("%w: %v", ErrCustomValidationFailed, err)
		}
	}

	return nil
}
