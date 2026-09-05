test:  # run tests
	go test -coverprofile=coverage.out ./... -tags=!mocks
	go tool cover -func=coverage.out | grep total
	go tool cover -html=coverage.out -o coverage.html

run-client: # run client
	go run ./cmd/client/main.go

run:  # build and run binary
	go build -o bin/custodia ./cmd/custodia
	./bin/custodia

mocks: # generate all mocks
	go generate ./...
