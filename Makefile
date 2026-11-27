.PHONY: generate migrate run tools docker-build docker-run

ifneq (,$(wildcard .env))
    include .env
    export
endif

docker-build:
	docker build -t trip-service:local .

docker-run:
	docker run --rm -it --network host --env-file .env trip-service:local

lint:
	golangci-lint run

tools:
	go get -tool github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen
	go get -tool github.com/pressly/goose/v3/cmd/goose
	go get -tool go.uber.org/mock/mockgen

test:
	go test -race -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -func=coverage.out | tail -n 1

test-integration:
	go test -race -tags=integration -cover ./...

generate:
	go tool oapi-codegen -config config/oapi-codegen.yaml contracts/openapi/trip-service.openapi.yaml

migrate:
	go tool goose -dir migrations postgres "$(DATABASE_URL)" up

migrate-down:
	go tool goose -dir migrations postgres "$(DATABASE_URL)" down

migrate-reset:
	go tool goose -dir migrations postgres "$(DATABASE_URL)" down-to 0

mutate:
	gremlins unleash --config .gremlins.yaml

run:
	go run ./cmd/trip-service