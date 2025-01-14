# Some people have gotestsum installed and like it so use it if it exists
HAS_GOTESTSUM := $(shell which gotestsum)
ifdef HAS_GOTESTSUM
    TEST_CMD = gotestsum --format testname --packages="./..." -- -count=1 -tags=integration -v -p 1
else
    TEST_CMD = go test ./... --count=1 -tags=integration
endif

lint:
	@golangci-lint run --disable-all --enable gci --fix
	@golangci-lint run

test:
	@$(TEST_CMD)

test-run:
	@$(TEST_CMD) -run=$(RUN)

fuzz:
	go test -fuzz=FuzzTestGitIgnore -fuzztime 30s

test-coverage:
	go test ./... -coverprofile coverage.out && go tool cover -html=coverage.out -o coverage.html

mod:
	@go mod tidy
	@go mod vendor

clean:
	go clean -modcache

all: mod lint test fuzz
