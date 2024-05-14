lint:
	go mod tidy
	go run github.com/golangci/golangci-lint/cmd/golangci-lint@v1.55.2 run --fix

test:
	go test -cover -race -v ./...
