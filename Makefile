GIT_COMMIT = $(shell git rev-parse --short=7 HEAD)

# TODO MCC_VERSION needs to come from Jenksins or such and as git tag
MCC_VERSION = 0.0.1
LD_FLAGS = "-w -X github.com/Mirantis/mcc/version.GitCommit=$(GIT_COMMIT) -X github.com/Mirantis/mcc/version.Version=$(MCC_VERSION)
BUILD_FLAGS = -a -tags "netgo static_build" -installsuffix netgo -ldflags $(LD_FLAGS) -extldflags '-static'"

test:
	go test -v ./...

build:
	go build $(BUILD_FLAGS) -o bin/mcc main.go

build-all:
	GOOS=linux GOARCH=amd64 go build $(BUILD_FLAGS) -o bin/mcc-linux-x64 main.go
	GOOS=windows GOARCH=amd64 go build $(BUILD_FLAGS) -o bin/mcc-win-x64.exe main.go
	GOOS=darwin GOARCH=amd64 go build $(BUILD_FLAGS) -o bin/mcc-darwin-x64 main.go

dump_envs:
	@echo COMMIT=$(GIT_COMMIT)

