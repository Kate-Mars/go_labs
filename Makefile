.PHONY: generate migrate run tools docker-build docker-run

ifneq (,$(wildcard .env))
    include .env
    export
endif

docker-build:
	docker build -t trip-service:local .

docker-run:
	docker run --rm -it --network host --env-file .env trip-service:local

tools:
	go get -tool github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen
	go get -tool github.com/pressly/goose/v3/cmd/goose
	go get -tool go.uber.org/mock/mockgen

generate: tools
	go tool oapi-codegen -generate types,chi-server -package api -o internal/generated/api.gen.go contracts/openapi/trip-service.openapi.yaml

migrate:
	go tool goose -dir migrations postgres "$(DATABASE_URL)" up

migrate-down:
	go tool goose -dir migrations postgres "$(DATABASE_URL)" down

migrate-reset:
	go tool goose -dir migrations postgres "$(DATABASE_URL)" down-to 0

run:
	go run ./cmd/trip-service