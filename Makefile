.PHONY: test verify build tidy

test:
	go test ./...

verify:
	go run ./cmd/hjkl dev verify

build:
	go build -o hjkl ./cmd/hjkl

tidy:
	go mod tidy
