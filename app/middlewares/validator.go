package middleware

import (
	e "auth_service/app/errors"
	"auth_service/shared/utils"

	v "github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
)

type ValidationFieldError struct {
	Field string `json:"field"`
	Tag   string `json:"tag"`
	Param string `json:"param"`
	Value any    `json:"value"`
}

type ValidationError struct {
	error
	List []ValidationFieldError `json:"list"`
}

func Validate[T any](payload T) error {
	validator := v.New()

	if err := validator.Struct(payload); err != nil {

		if _, ok := err.(*v.InvalidValidationError); ok {
			return e.ThrowUnprocessableEntity(err.Error())
		}

		fields := []ValidationFieldError{}
		for _, err := range err.(v.ValidationErrors) {
			fields = append(fields, ValidationFieldError{
				Field: err.Field(),
				Tag:   err.Tag(),
				Param: err.Param(),
				Value: err.Value(),
			})
		}

		// return ValidationError{
		// 	error: err,
		// 	List:  errors,
		// }
		return e.ThrowValidationError(err.Error(), utils.JSON{"fields": fields})
	}

	return nil
}

func BodyValidator[T any]() func(ctx fiber.Ctx) error {

	return func(ctx fiber.Ctx) error {

		var payload T
		if err := ctx.Bind().Body(&payload); err != nil {
			return err
		}
		err := Validate(payload)

		if err != nil {
			return err
		}

		return ctx.Next()
	}
}
