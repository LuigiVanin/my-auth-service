CMD_MAIN := ./cmd/main.go
CMD_MIGRATE := ./cmd/database/migration/main.go
CMD_SEED := ./cmd/database
CMD_CIPHER := ./cmd/helpers/chipher.go

.PHONY: run dev build migrate seed cipher

start:
	go run $(CMD_MAIN)

dev:
	air -- $(filter-out $@,$(MAKECMDGOALS))

build:
	go build -o ./build/auth_service $(CMD_MAIN)

migrate:
	go run $(CMD_MIGRATE) $(filter-out $@,$(MAKECMDGOALS))

# create-migration is deprecated with GORM AutoMigrate
# create-migration:
# 	@if [ -z "$(filter-out $@,$(MAKECMDGOALS))" ]; then \
# 		echo "Usage: make create-migration <migration-name>"; \
# 		exit 1; \
# 	fi
# 	./migrate create -ext sql -dir ./migrations -digits 3 -seq $(filter-out $@,$(MAKECMDGOALS))

# Usage: make seed init [environment] | make seed reset [environment]
seed:
	go run $(CMD_SEED) $(filter-out $@,$(MAKECMDGOALS))

fresh:
	make seed reset && make migrate && make seed init

cipher:
	go run $(CMD_CIPHER) $(filter-out $@,$(MAKECMDGOALS))

%:
	@:
