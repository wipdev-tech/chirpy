fmt:
	go fmt ./...

lint: fmt
	golangci-lint run

r: fmt
	go run .
