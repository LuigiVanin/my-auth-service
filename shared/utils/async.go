package utils

import "go.uber.org/zap"

// Detach runs fn off the lifetime of the request that started it, for work whose
// result the response does not carry.
//
// The recover is the reason this exists rather than a bare `go`: the fiber
// recoverer wraps the handler's own stack, so a panic on a goroutine the handler
// spawned unwinds alone and takes the process down with it. `name` is what the log
// line is searched by.
func Detach(logger *zap.Logger, name string, fn func() error) {
	if logger == nil {
		logger = zap.NewNop()
	}

	go func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				logger.Error(
					"Detached work panicked",
					zap.String("work", name),
					zap.Any("panic", recovered),
					zap.Stack("stack"),
				)
			}
		}()

		if err := fn(); err != nil {
			logger.Error("Detached work failed", zap.String("work", name), zap.Error(err))
		}
	}()
}
