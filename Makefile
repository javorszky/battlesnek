PWD=$(shell pwd)


tidy:
	go mod tidy && go mod download && go mod vendor


run:
	go run main.go

lint:
	docker run --rm -v $(PWD):/app -w /app golangci/golangci-lint:v1.46.2 golangci-lint run -v --fix

gci:
	docker compose up lintfixer

.PHONY: tidy run lint gci
