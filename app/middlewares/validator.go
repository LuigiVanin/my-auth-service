package middleware

import (
	e "auth_service/common/errors"

	v "github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
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

func validate[T any](payload T) error {
	validator := v.New()

	if err := validator.Struct(payload); err != nil {

		if _, ok := err.(*v.InvalidValidationError); ok {
			return e.ThrowUnprocessableEntity(err.Error())
		}

		errors := []ValidationFieldError{}
		for _, err := range err.(v.ValidationErrors) {
			errors = append(errors, ValidationFieldError{
				Field: err.Field(),
				Tag:   err.Tag(),
				Param: err.Param(),
				Value: err.Value(),
			})
		}

		return ValidationError{
			error: err,
			List:  errors,
		}
	}

	return nil
}

func BodyValidator[T any]() func(ctx *fiber.Ctx) error {

	return func(ctx *fiber.Ctx) error {

		var payload T
		if err := ctx.BodyParser(&payload); err != nil {
			return ctx.
				Status(fiber.StatusBadRequest).
				JSON(
					fiber.Map{
						"message": "Invalid request body",
						"error":   err.Error(),
					})
		}

		err := validate(payload)

		if validationErr, ok := err.(ValidationError); ok {
			return ctx.Status(fiber.StatusBadRequest).JSON(
				fiber.Map{
					"message": "Validation error",
					"list":    validationErr.List,
					"error":   validationErr.Error(),
				},
			)
		} else if err != nil {
			return ctx.Status(fiber.StatusBadRequest).JSON(
				fiber.Map{
					"message": "Invalid request body",
					"error":   err.Error(),
				})
		}

		return ctx.Next()
	}
}
