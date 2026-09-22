.PHONY: tools generate

tools:
	go get -tool github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen
	go get -tool github.com/pressly/goose/v3/cmd/goose
	go get -tool go.uber.org/mock/mockgen

generate: tools