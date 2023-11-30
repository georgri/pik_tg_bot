BINARY_NAME=pkg_tg_bot
.DEFAULT_GOAL := run

build:
	go build -o ./cmd/${BINARY_NAME}-app ./cmd/main.go


run: build
	./cmd/${BINARY_NAME}-app


build_and_run: build run


