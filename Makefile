GOPATH ?= $(shell go env GOPATH)
GOBIN ?= $(GOPATH)/bin

static-analysis/lint:
	@golangci-lint run --fix

static-analysis/vulncheck:
	@go tool -modfile=./tools/go.mod govulncheck ./...

static-analysis/vulncheck/sarif:
	@mkdir -p build
	@go tool -modfile=./tools/go.mod govulncheck -format sarif ./... > build/govulncheck-report.sarif

build:
	@go build

test:
	@mkdir -p build
	@go test ./... -v -failfast -coverprofile=build/coverage.out
