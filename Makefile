APP_NAME := zapi
DB_URL ?= $(shell grep DATABASE_URL .env 2>/dev/null | cut -d= -f2-)

.PHONY: build run test clean migrate migrate-down dev docker docker-run

## Build

build:
	go build -o $(APP_NAME) ./cmd/server/

run: build
	./$(APP_NAME)

dev:
	go run ./cmd/server/

## Test

test:
	go test ./... -count=1

test-v:
	go test ./... -v -count=1

## Database

migrate:
	@echo "Migrations run automatically on server start. Use 'make dev' or 'make run'."

migrate-down:
	psql "$(DB_URL)" -c "DROP TABLE IF EXISTS rate_snapshots, subscribers, schema_migrations CASCADE;"

migrate-reset: migrate-down dev

seed:
	psql "$(DB_URL)" -c "INSERT INTO subscribers (name, api_key, is_admin) VALUES ('default', 'zapi-test-key-123', true) ON CONFLICT DO NOTHING;"

## Docker

docker:
	docker build -t $(APP_NAME) .

docker-run:
	docker run --rm --env-file .env -p 8080:8080 $(APP_NAME)

## Clean

clean:
	rm -f $(APP_NAME)
	go clean -testcache
