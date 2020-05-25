GIT_COMMIT = $(shell git rev-parse --short=7 HEAD)
ifdef TAG_NAME
	ENVIRONMENT = "production"
endif
ENVIRONMENT ?= "development"
LAUNCHPAD_VERSION ?= $(or ${TAG_NAME},dev)
LD_FLAGS = "-w -X github.com/Mirantis/mcc/version.Environment=$(ENVIRONMENT) -X github.com/Mirantis/mcc/version.GitCommit=$(GIT_COMMIT) -X github.com/Mirantis/mcc/version.Version=$(LAUNCHPAD_VERSION)
BUILD_FLAGS = -a -tags "netgo static_build" -installsuffix netgo -ldflags $(LD_FLAGS) -extldflags '-static'"

BUILDER_IMAGE = launchpad-builder
GO = docker run --rm -v "$(CURDIR)":/go/src/github.com/Mirantis/mcc \
	-w "/go/src/github.com/Mirantis/mcc" \
	-e GOPATH\
	-e GOOS \
	-e GOARCH \
	-e GOEXE \
	$(BUILDER_IMAGE)

builder:
	docker build -t $(BUILDER_IMAGE) -f Dockerfile.builder .

unit-test: builder
	$(GO) go test -v ./...

build: builder
	$(GO) go build $(BUILD_FLAGS) -o bin/launchpad main.go

build-all: builder
	GOOS=linux GOARCH=amd64 $(GO) go build $(BUILD_FLAGS) -o bin/launchpad-linux-x64 main.go
	GOOS=windows GOARCH=amd64 $(GO) go build $(BUILD_FLAGS) -o bin/launchpad-win-x64.exe main.go
	GOOS=darwin GOARCH=amd64 $(GO) go build $(BUILD_FLAGS) -o bin/launchpad-darwin-x64 main.go

release: build-all
	./release.sh

lint: builder
	$(GO) go vet ./...
	$(GO) golint -set_exit_status ./...

smoke-test: build
	./test/smoke.sh

smoke-upgrade-test: build
	./test/smoke_upgrade.sh
