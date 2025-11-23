package bootstrap

import (
	"auth_service/infra/config"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func NewZapLogger(cfg *config.Config) *zap.Logger {
	var loggerConfig zap.Config

	if cfg.Env == "dev" {
		loggerConfig = zap.NewDevelopmentConfig()
		loggerConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		loggerConfig.EncoderConfig.ConsoleSeparator = " "

		loggerConfig.EncoderConfig.EncodeTime = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendString(t.Format("[15:04:05]:"))
		}

		// Disable stack traces in development for cleaner logs
		loggerConfig.DisableStacktrace = true

	} else {
		// Use JSON encoding for production
		loggerConfig = zap.NewProductionConfig()
		loggerConfig.Encoding = "json"
		loggerConfig.EncoderConfig.TimeKey = "timestamp"
		loggerConfig.EncoderConfig.LevelKey = "level"
		loggerConfig.EncoderConfig.MessageKey = "message"
		loggerConfig.EncoderConfig.CallerKey = "caller"
		loggerConfig.EncoderConfig.StacktraceKey = "stacktrace"
	}

	loggerConfig.OutputPaths = []string{"stdout"}
	loggerConfig.ErrorOutputPaths = []string{"stderr"}
	loggerConfig.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	loggerConfig.EncoderConfig.LineEnding = "\n"

	logger := zap.Must(loggerConfig.Build())

	return logger
}
