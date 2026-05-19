golint:
	golangci-lint run -E gocritic -v ./...

test:
	go test ./internal/... -v

run:
	 go run cmd/main.go