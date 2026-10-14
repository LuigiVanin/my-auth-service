package guards

import (
	"fmt"

	e "auth_service/app/errors"

	"github.com/gofiber/fiber/v3"
)

// readLocal reads what an earlier guard left in Locals, telling apart the two
// ways it can fail. Absent means the chain is out of order. Present with another
// type means the writer changed what it stores - every reader compiles anyway,
// because Locals is `any`, so the detail names both types.
func readLocal[T any](ctx fiber.Ctx, key string, reader string, writer string) (*T, error) {
	value := ctx.Locals(key)

	if value == nil {
		return nil, e.ThrowInternalServerError(
			fmt.Sprintf("Locals %q missing in %s - should be run after %s", key, reader, writer),
		)
	}

	typed, ok := value.(*T)

	if !ok || typed == nil {
		return nil, e.ThrowInternalServerError(
			fmt.Sprintf("Locals %q in %s holds %T, expected %T - %s stored the wrong type", key, reader, value, typed, writer),
		)
	}

	return typed, nil
}
