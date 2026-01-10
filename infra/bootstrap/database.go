package bootstrap

import (
	"auth_service/infra/config"

	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func NewDatabase(cfg *config.Config, zapLogger *zap.Logger) *gorm.DB {

	db, err := gorm.Open(postgres.Open(cfg.FormatDatabaseUrl()), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})

	if err != nil {
		zapLogger.Error("failed to connect to database", zap.Error(err))
		panic(err.Error())
	}

	return db
}
