CONFIG_PATH ?= ./configs/local.yaml
INT_CONFIG_PATH ?= ../configs/int_tests.yaml

.PHONY: run-app stop-app migrate-up migrate-down integration-test unit-test

run-app:
	docker compose up --build -d

stop-app:
	docker compose down -v

migrate-up:
	CONFIG_PATH=$(CONFIG_PATH) go run cmd/migrator/main.go up

migrate-down:
	CONFIG_PATH=$(CONFIG_PATH) go run cmd/migrator/main.go down

unit-test:
	go test -v -cover ./internal/...

integration-test:
	docker compose -f ./docker/tests/docker-compose.yml up --build -d
	CONFIG_PATH=$(INT_CONFIG_PATH) go test -v -tags=integration ./tests/...
	docker compose -f ./docker/tests/docker-compose.yml down -v
