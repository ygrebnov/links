ROOT_PATH := $(dir $(realpath $(lastword $(MAKEFILE_LIST))))
COVERAGE_PATH := $(ROOT_PATH).coverage
BIN_PATH ?= $(ROOT_PATH)bin

lint:
	@golangci-lint run

test-unit:
	@rm -rf $(COVERAGE_PATH)
	@mkdir -p $(COVERAGE_PATH)
	@go test -v -coverpkg=./internal ./internal -coverprofile $(COVERAGE_PATH)/coverage.txt
	@go tool cover -func=$(COVERAGE_PATH)/coverage.txt -o $(COVERAGE_PATH)/functions.txt
	@go tool cover -html=$(COVERAGE_PATH)/coverage.txt -o $(COVERAGE_PATH)/coverage.html


PKG := github.com/ygrebnov/links/internal

VERSION ?= $(shell git tag | sort -Vr | head -n1)
ifeq ($(VERSION),)
VERSION := v0.0.0
endif

BUILD_TIME ?= $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')

build-e2e:
	@go build \
    		-ldflags "-w -X $(PKG).version=$(VERSION) -X $(PKG).buildTime=$(BUILD_TIME)" \
    		-o tests/links

test-e2e: build-e2e
	@go test -v ./tests
	@rm -f tests/links

test: test-unit test-e2e

GOOS ?= $(shell uname -s | tr '[:upper:]' '[:lower:]')
GOARCH ?= $(shell uname -m)
ifeq ($(GOARCH), x86_64)
	override GOARCH = amd64
endif

build-locally:
	@rm -f $(BIN_PATH)/links*
	CGO_ENABLED=0 go build \
    		-ldflags "-w -X $(PKG).version=$(VERSION) -X $(PKG).buildTime=$(BUILD_TIME)" \
    		-o $(BIN_PATH)/links-$(GOOS)-$(GOARCH)$(shell [ ${GOOS} = windows ] && echo .exe)

build-image:
	@docker build \
		--build-arg PKG=$(PKG) \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_TIME=$(BUILD_TIME) \
		-t links:$(VERSION) .

build:
	@goreleaser release --snapshot --skip-publish --clean

.PHONY: lint test-unit build-e2e test-e2e test build-locally build-image build