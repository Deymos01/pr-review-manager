CONFIG_PATH ?= ./configs/local.yaml
INT_CONFIG_PATH ?= ../configs/int_tests.yaml

.PHONY: run-app run migrate-up migrate-down integration-test

run-app:
	docker compose -f ./docker/docker-compose.yml up --build -d

migrate-up:
	CONFIG_PATH=$(CONFIG_PATH) go run cmd/migrator/main.go up

migrate-down:
	CONFIG_PATH=$(CONFIG_PATH) go run cmd/migrator/main.go down

unit-test:
	go test -v -cover ./internal/...

integration-test:
	docker compose -f ./docker/tests/docker-compose.yml up --build -d
	sleep 5
	CONFIG_PATH=$(INT_CONFIG_PATH) go test -v -tags=integration ./tests/...
	docker compose -f ./docker/tests/docker-compose.yml down
