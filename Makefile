
# Important: This Makefile is only for local use and contains shortcuts for working on the project
# Do not use these make targets in a pipeline or production build context.

HAS_GOTESTSUM := $(shell which gotestsum)
ifdef HAS_GOTESTSUM
    TEST_CMD = gotestsum --format testname --packages="./..." -- -count=1 -tags=integration -v -p 1
else
    TEST_CMD = go test ./... --count=1 -tags=integration
endif

test:
	@$(TEST_CMD)

test-run:
	@$(TEST_CMD) -run=$(RUN)

test-coverage:
	go test ./... -coverprofile coverage.out && go tool cover -html=coverage.out -o coverage.html
	go-cover-treemap -coverprofile coverage.out > coverage.svg

check-comment-tags:
	 go run tasks.go -comment-tags

gitleaks:
	gitleaks detect -v -c gitleaks.toml

gen:
	go generate -v ./...

mod:
	@go mod tidy
	@go mod vendor

clean:
	go clean -modcache

build:
	go build -o scc main.go

all: mod gen build test
