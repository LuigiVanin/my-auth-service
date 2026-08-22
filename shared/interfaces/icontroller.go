package interfaces

import "github.com/gofiber/fiber/v3"

type IController interface {
	Register(server *fiber.App)
}
