package bootstrap

import (
	"auth_service/infra/config"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

func NewDatabase(cfg *config.Config, logger *zap.Logger) *sqlx.DB {
	db, err := sqlx.Connect("postgres", cfg.FormatDatabaseUrl())

	if err != nil {
		logger.Error("failed to connect to database", zap.Error(err))
		panic(err.Error())
	}

	return db
}
