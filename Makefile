# ========== BUILD / RUN ==========

VERSION := $(shell git describe --tags --always 2>/dev/null || echo "")
DATE    := $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
COMMIT  := $(shell git rev-parse HEAD 2>/dev/null || echo "")

server:  # build and run binary
	go build -ldflags "-X main.buildVersion=$(VERSION) \
                   -X main.buildDate=$(DATE) \
                   -X main.buildCommit=$(COMMIT)" \
                   -o bin/server ./cmd/server
	./bin/server

client:
	go build -ldflags "-X main.buildVersion=$(VERSION) \
                   -X main.buildDate=$(DATE) \
                   -X main.buildCommit=$(COMMIT)" \
                   -o bin/custodia ./cmd/custodia
	go install ./cmd/custodia

# ========== GENERATE ==========

proto:
	protoc --go_opt=default_api_level=API_OPAQUE \
		--go_out=. \
 		--go_opt=module=github.com/max-marek-projects/custodia \
		--go-grpc_out=. \
 		--go-grpc_opt=module=github.com/max-marek-projects/custodia \
		--openapiv2_out=pkg/openapi --openapiv2_opt=module=github.com/max-marek-projects/custodia \
		api/custodia.proto

mocks: # generate all mocks
	go generate ./...

# ========== TESTING AND LINTING ==========

lint:
	gofmt -w .
	goimports -w .

static-lint:
	go run ./cmd/staticlint/main.go ./cmd/...
	go run ./cmd/staticlint/main.go ./internal/...

test:  # run tests
	go test -coverprofile=coverage.out ./internal/... ./cmd/...
	go tool cover -func=coverage.out | grep total
	go tool cover -html=coverage.out -o coverage.html
