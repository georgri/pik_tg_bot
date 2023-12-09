BINARY_NAME=pkg_tg_bot
.DEFAULT_GOAL := run

build:
	go build -o ./cmd/${BINARY_NAME}-app ./cmd/main.go


run: build
	mkdir -p logs
	mkdir -p data
	./cmd/${BINARY_NAME}-app


build_and_run: build run

clean:
	go clean
	rm ./cmd/${BINARY_NAME}-app


test:
	go test ./...


test_coverage:
	go test ./... -coverprofile=coverage.out


dep:
	go mod download

vet:
	go vet

lint:
	golangcli-lint run --enable-all
