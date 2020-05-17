GIT_COMMIT = $(shell git rev-parse --short=7 HEAD)

# TODO MCC_VERSION needs to come from Jenksins or such and as git tag
MCC_VERSION = 0.0.1
LD_FLAGS = "-w -X github.com/Mirantis/mcc/version.GitCommit=$(GIT_COMMIT) -X github.com/Mirantis/mcc/version.Version=$(MCC_VERSION)
BUILD_FLAGS = -a -tags "netgo static_build" -installsuffix netgo -ldflags $(LD_FLAGS) -extldflags '-static'"

BUILDER_IMAGE = mcc-builder
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
	$(GO) go build $(BUILD_FLAGS) -o bin/mcc main.go

build-all: builder
	GOOS=linux GOARCH=amd64 $(GO) go build $(BUILD_FLAGS) -o bin/mcc-linux-x64 main.go
	GOOS=windows GOARCH=amd64 $(GO) go build $(BUILD_FLAGS) -o bin/mcc-win-x64.exe main.go
	GOOS=darwin GOARCH=amd64 $(GO) go build $(BUILD_FLAGS) -o bin/mcc-darwin-x64 main.go

lint: builder
	$(GO) go vet ./...
	$(GO) golint -set_exit_status ./...

smoke-test: build
	./test/smoke.sh

smoke-upgrade-test: build
	./test/smoke_upgrade.sh
