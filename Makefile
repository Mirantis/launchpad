GIT_COMMIT = $(shell git rev-parse --short=7 HEAD)
ifdef TAG_NAME
	ENVIRONMENT = "production"
endif
ENVIRONMENT ?= "development"
LAUNCHPAD_VERSION ?= $(or ${TAG_NAME},dev)
LD_FLAGS = -s -w -X github.com/Mirantis/mcc/version.Environment=$(ENVIRONMENT) -X github.com/Mirantis/mcc/version.GitCommit=$(GIT_COMMIT) -X github.com/Mirantis/mcc/version.Version=$(LAUNCHPAD_VERSION)
BUILD_FLAGS = -trimpath -a -tags "netgo static_build" -installsuffix netgo -ldflags "$(LD_FLAGS) -extldflags '-static'"
ifeq ($(OS),Windows_NT)
			 GOOS ?= "windows"
			 TARGET ?= "bin/launchpad.exe"
else
			 TARGET ?= "bin/launchpad"
       uname_s := $(shell uname -s | tr '[:upper:]' '[:lower:]')
       GOOS ?= ${uname_s}
endif
gosrc = $(wildcard *.go */*.go */*/*.go */*/*/*.go)

clean:
	rm -f bin/launchpad-*

bin/launchpad-linux-x64:
	GOARCH=amd64 go build ${BUILD_FLAGS} -o bin/launchpad-linux-x64 main.go

bin/launchpad-linux-arm64:
	GOARCH=arm64 go build ${BUILD_FLAGS} -o bin/launchpad-linux-arm64 main.go

bin/launchpad-win-x64.exe:
	GOARCH=amd64 go build ${BUILD_FLAGS} -o bin/launchpad-win-x64.exe main.go

bin/launchpad-darwin-x64:
	GOARCH=amd64 go build ${BUILD_FLAGS} -o bin/launchpad-darwin-x64 main.go

bin/launchpad-darwin-arm64:
	GOARCH=arm64 go build ${BUILD_FLAGS} -o bin/launchpad-darwin-arm64 main.go

bin/launchpad: bin/launchpad-${GOOS}-x64
	cp bin/launchpad-${GOOS}-x64 bin/launchpad

bin/launchpad.exe: bin/launchpad-win-x64.exe
	cp bin/launchpad-win-x64 bin/launchpad.exe

unit-test:
	go test -v ./...

build: $(TARGET)

build-all: bin/launchpad-linux-x64 bin/launchpad-linux-arm64 bin/launchpad-win-x64.exe bin/launchpad-darwin-x64 bin/launchpad-darwin-arm64

release: build-all
	./release.sh

lint: builder
	go vet ./...
	golint -set_exit_status ./...

smoke-register-test: build
	./test/smoke_register.sh

smoke-apply-test: build
	./test/smoke_apply.sh

smoke-apply-upload-test: build
	./test/smoke_upload.sh

smoke-apply-local-repo-test: build
	./test/smoke_localrepo.sh

smoke-reset-local-repo-test: build
	./test/smoke_localrepo_reset.sh

smoke-apply-test-localhost: build
	./test/smoke_apply_local.sh

smoke-apply-bastion-test: build
	./test/smoke_apply_bastion.sh

smoke-apply-forward-test: build
	./test/smoke_apply_forward.sh

smoke-upgrade-test: build
	./test/smoke_upgrade.sh

smoke-prune-test: build
	./test/smoke_prune.sh

smoke-reset-test: build
	./test/smoke_reset.sh

smoke-cleanup:
	./test/smoke_cleanup.sh

smoke-test: smoke-apply-test
