
# Ensure we use auto toolchain
export GOTOOLCHAIN=auto


.PHONY: clean
clean:
	rm -fr dist

# TODO: Digicert signing will be reimplemented in GitHub Actions.

# Local build of the plugin. This saves time building only the host platform.
# Uses native Go commands to avoid Goreleaser dependency.
.PHONY: local
local:
	mkdir -p dist
	GOOS=$(shell go env GOOS) GOARCH=$(shell go env GOARCH) \
	VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo 0.0.0) \
	COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo HEAD) \
	output_name="dist/launchpad_$${GOOS}_$${GOARCH}"; \
	if [ "$${GOOS}" = "windows" ]; then \
		output_name="$${output_name}.exe"; \
	fi; \
	go build \
		-ldflags "-X github.com/Mirantis/launchpad/version.Version=$${VERSION} -X github.com/Mirantis/launchpad/version.GitCommit=$${COMMIT}" \
		-o "$${output_name}" ./main.go && \
	./$${output_name} version

# run linting
.PHONY: lint
lint:
	golangci-lint run

# security scanning
.PHONY: security-scan
security-scan:
	govulncheck ./...

# Testing related targets

# TEST_FLAGS can be set in CI to e.g. -short to skip tests that need network/OCI
TEST_FLAGS?=
.PHONY: unit-test
unit-test:
	go test -v --tags 'testing' $(TEST_FLAGS) ./pkg/...

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
