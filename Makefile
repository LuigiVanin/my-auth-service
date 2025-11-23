CMD_MAIN := ./cmd/main.go

run:
	go run $(CMD_MAIN)

dev:
	air

build:
	go build -o ./bin/auth_service $(CMD_MAIN)