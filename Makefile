
GO=$(shell which go)

RELEASE_FOLDER=dist/release

CHECKSUM=$(shell which sha256sum)

VOLUME_MOUNTS=-v "$(CURDIR):/v"

GOLANGCI_LINT?=docker run -t --rm -v "$(CURDIR):/data" -w "/data" golangci/golangci-lint:latest golangci-lint

SEGMENT_TOKEN?=""

.PHONY: clean
clean:
	rm -fr dist

# TODO: Digicert signing will be reimplemented in GitHub Actions.

# Force a clean build of the artifacts by first cleaning
# and then building
.PHONY: build-release
build-release: clean $(RELEASE_FOLDER)
# Build all the binaries for release using native Go commands.
# This replaces Goreleaser to avoid dependency on external tools.
$(RELEASE_FOLDER):
	mkdir -p $(RELEASE_FOLDER)
	platforms=("linux/amd64" "linux/arm64" "windows/amd64" "windows/arm64" "darwin/amd64" "darwin/arm64")
	for platform in "$${platforms[@]}"; do \
		GOOS=$${platform%/*} GOARCH=$${platform#*/} \
		output_name="$(RELEASE_FOLDER)/launchpad_$${GOOS}_$${GOARCH}" \
		if [ "$${GOOS}" = "windows" ]; then \
			output_name+=".exe" \
		fi \
		echo "Building $${output_name}" \
		GOOS=$${GOOS} GOARCH=$${GOARCH} $(GO) build -o "$${output_name}" ./main.go; \
	done

.PHONY: create-checksum
create-checksum:
	cd $(RELEASE_FOLDER) && \
	for f in *; do \
		$(CHECKSUM) $$f > $$f.sha256; \
	done

.PHONY: verify-checksum
verify-checksum:
	for f in $(RELEASE_FOLDER)/*.sha256; do \
		$(CHECKSUM) -c $$f; \
		echo "Verified checksum for $$f"; \
	done

# clean out any existing release build
.PHONY: clean-release
clean-release:
	rm -fr $(RELEASE_FOLDER)

# Local build of the plugin. This saves time building only the host platform.
# Uses native Go commands to avoid Goreleaser dependency.
.PHONY: local
local:
	mkdir -p dist
	GOOS=$(shell go env GOOS) GOARCH=$(shell go env GOARCH) \
	output_name="dist/launchpad_$${GOOS}_$${GOARCH}"; \
	if [ "$${GOOS}" = "windows" ]; then \
		output_name="$${output_name}.exe"; \
	fi; \
	$(GO) build -o "$${output_name}" ./main.go && \
	./$${output_name} --help

# run linting
.PHONY: lint
lint:
	$(GOLANGCI_LINT) run

# Testing related targets

# TEST_FLAGS can be set in CI to e.g. -short to skip tests that need network/OCI
TEST_FLAGS?=
.PHONY: unit-test
unit-test:
	$(GO) test -v --tags 'testing' $(TEST_FLAGS) ./pkg/...

.PHONY: functional-test
functional-test:
	go test -v ./test/functional/... -timeout 20m

.PHONY: integration-test
integration-test:
	go test -v ./test/integration/... -timeout 20m

.PHONY: smoke-small
smoke-small:
	go test -count=1 -v ./test/smoke/... -run TestSmallCluster -timeout 20m

.PHONY: smoke-full
smoke-full:
	go test -count=1 -v ./test/smoke/... -run TestSupportedMatrixCluster -timeout 50m

.PHONY: clean-launchpad-chart
clean-launchpad-chart:
	terraform -chdir=./examples/tf-aws/launchpad apply --auto-approve --destroy
