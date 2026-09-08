# ========== BUILD / RUN ==========

server:  # build and run binary
	go build -o bin/server ./cmd/server
	./bin/server

client:
	go build -o bin/custodia ./cmd/custodia
	go install ./cmd/custodia

# ========== GENERATE ==========

proto:
	protoc --go_opt=default_api_level=API_OPAQUE \
		--go_out=. \
 		--go_opt=module=github.com/max-marek-projects/custodia \
		--go-grpc_out=. \
 		--go-grpc_opt=module=github.com/max-marek-projects/custodia \
		api/custodia.proto

mocks: # generate all mocks
	go generate ./...

# ========== TESTING AND LINTING ==========

lint:
	gofmt -w .
	goimports -w .

test:  # run tests
	go test -coverprofile=coverage.out ./internal/... ./cmd/...
	go tool cover -func=coverage.out | grep total
	go tool cover -html=coverage.out -o coverage.html
