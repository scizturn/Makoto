.PHONY: test build run-once scheduler worker

test:
	go test ./...

build:
	go build -o bin/makoto ./cmd/makoto

run-once:
	MAKOTO_MODE=run-once go run ./cmd/makoto

scheduler:
	MAKOTO_MODE=scheduler go run ./cmd/makoto

worker:
	MAKOTO_MODE=worker go run ./cmd/makoto
