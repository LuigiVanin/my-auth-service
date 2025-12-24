package bootstrap

import (
	"auth_service/infra/config"

	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func NewDatabase(cfg *config.Config, logger *zap.Logger) *gorm.DB {
	db, err := gorm.Open(postgres.Open(cfg.FormatDatabaseUrl()), &gorm.Config{})

	if err != nil {
		logger.Error("failed to connect to database", zap.Error(err))
		panic(err.Error())
	}

	return db
}
