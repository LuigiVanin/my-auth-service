CMD_MAIN := ./cmd/main.go
CMD_MIGRATE := ./cmd/database/migration/main.go
CMD_START_UP := ./cmd/database/init/main.go
CMD_CIPHER := ./cmd/helpers/chipher.go

.PHONY: run dev build migrate init cipher

run:
	go run $(CMD_MAIN)

dev:
	air

build:
	go build -o ./build/auth_service $(CMD_MAIN)

migrate:
	go run $(CMD_MIGRATE)

# create-migration is deprecated with GORM AutoMigrate
# create-migration:
# 	@if [ -z "$(filter-out $@,$(MAKECMDGOALS))" ]; then \
# 		echo "Usage: make create-migration <migration-name>"; \
# 		exit 1; \
# 	fi
# 	./migrate create -ext sql -dir ./migrations -digits 3 -seq $(filter-out $@,$(MAKECMDGOALS))

init:
	go run $(CMD_START_UP)

cipher:
	go run $(CMD_CIPHER) $(filter-out $@,$(MAKECMDGOALS))

%:
	@:
