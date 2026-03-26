
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
	output_name="dist/launchpad_$${GOOS}_$${GOARCH}"; \
	if [ "$${GOOS}" = "windows" ]; then \
		output_name="$${output_name}.exe"; \
	fi; \
	go build -o "$${output_name}" ./main.go && \
	./$${output_name} --help

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
