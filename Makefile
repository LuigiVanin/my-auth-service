CMD_MAIN := ./cmd/main.go
CMD_MIGRATE := ./cmd/database/migrate.go

.PHONY: run dev build migrate create-migration

run:
	go run $(CMD_MAIN)

dev:
	air

build:
	go build -o ./build/auth_service $(CMD_MAIN)

migrate:
	go run $(CMD_MIGRATE)

create-migration:
	@if [ -z "$(filter-out $@,$(MAKECMDGOALS))" ]; then \
		echo "Usage: make create-migration <migration-name>"; \
		exit 1; \
	fi
	./migrate create -ext sql -dir ./migrations -digits 3 -seq $(filter-out $@,$(MAKECMDGOALS))

%:
	@: