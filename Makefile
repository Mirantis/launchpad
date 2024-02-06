GIT_COMMIT = $(shell git rev-parse --short=7 HEAD)
ifdef TAG_NAME
	ENVIRONMENT = "production"
endif
ENVIRONMENT ?= "development"
LAUNCHPAD_VERSION ?= $(or ${TAG_NAME},dev)
LD_FLAGS = -s -w -X github.com/Mirantis/mcc/version.Environment=$(ENVIRONMENT) -X github.com/Mirantis/mcc/version.GitCommit=$(GIT_COMMIT) -X github.com/Mirantis/mcc/version.Version=$(LAUNCHPAD_VERSION)
BUILD_FLAGS = -trimpath -a -tags "netgo static_build" -installsuffix netgo -ldflags "$(LD_FLAGS) -extldflags '-static'" -v
LAUNCHPAD_VERSION ?= $(or ${TAG_NAME},dev)
ifeq ($(OS),Windows_NT)
       uname_s := "windows"
       TARGET ?= "bin\\launchpad.exe"
else
       uname_s := $(shell uname -s | tr '[:upper:]' '[:lower:]')
       TARGET ?= "bin/launchpad"
endif
GOOS ?= ${uname_s}
BUILDER_IMAGE = launchpad-builder
GO = docker run --rm -v "$(CURDIR)":/go/src/github.com/Mirantis/mcc \
	-w "/go/src/github.com/Mirantis/mcc" \
	-e GOPATH\
	-e GOOS \
	-e GOARCH \
	-e GOEXE \
	$(BUILDER_IMAGE)
gosrc = $(wildcard *.go */*.go */*/*.go */*/*/*.go)

VOLUME_MOUNTS=-v "$(CURDIR):/v"
SIGN?=docker run --rm -i $(VOLUME_MOUNTS) -e SM_API_KEY -e SM_CLIENT_CERT_PASSWORD -e SM_CLIENT_CERT_FILE -v "$(SM_CLIENT_CERT_FILE):$(SM_CLIENT_CERT_FILE)" -w "/v" registry.mirantis.com/prodeng/digicert-keytools-jsign:latest sign

sign-win:
	echo "Signing Windows binary"
	$(SIGN) bin/launchpad-win-x64.exe

clean:
	sudo rm -f bin/launchpad

builder:
	docker build --ssh default=${SSH_AUTH_SOCK} -t $(BUILDER_IMAGE) -f Dockerfile.builder .

unit-test: builder
	$(GO) go test -v ./... -tags testing

$(TARGET): $(gosrc)
	docker build --ssh default=${SSH_AUTH_SOCK} -t $(BUILDER_IMAGE) -f Dockerfile.builder .
	GOOS=${GOOS} $(GO) go build $(BUILD_FLAGS) -o $(TARGET) main.go

build: $(TARGET)

build-all: builder
	GOOS=linux GOARCH=amd64 $(GO) go build $(BUILD_FLAGS) -o bin/launchpad-linux-x64 main.go
	GOOS=linux GOARCH=arm64 $(GO) go build $(BUILD_FLAGS) -o bin/launchpad-linux-arm64 main.go
	GOOS=windows GOARCH=amd64 $(GO) go build $(BUILD_FLAGS) -o bin/launchpad-win-x64.exe main.go
	GOOS=darwin GOARCH=amd64 $(GO) go build $(BUILD_FLAGS) -o bin/launchpad-darwin-x64 main.go
	GOOS=darwin GOARCH=arm64 $(GO) go build $(BUILD_FLAGS) -o bin/launchpad-darwin-arm64 main.go

go-mod-tidy: builder
	$(GO) go mod tidy

release: build-all sign-win
	./release.sh

lint:
	docker run -t --rm -v "$(CURDIR):/data" -w "/data" golangci/golangci-lint:latest golangci-lint run

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
