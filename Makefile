.PHONY: tools generate migrate run

tools:
	go get -tool github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen
	go get -tool github.com/pressly/goose/v3/cmd/goose
	go get -tool go.uber.org/mock/mockgen

generate: tools
	go tool oapi-codegen -generate types,chi-server -package api -o internal/generated/api.gen.go contracts/openapi/trip-service.openapi.yaml

migrate:
	go tool goose -dir migrations postgres "$(DATABASE_URL)" up

run:
	go run ./cmd/trip-service