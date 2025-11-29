package interfaces

import "github.com/gofiber/fiber/v2"

type IController interface {
	Register(server *fiber.App)
}
