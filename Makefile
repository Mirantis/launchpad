
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
# ./test holds the helpers the smoke and integration suites share. Its tests
# are hermetic (no AWS, no network) and run here so the credential handling in
# test/terraform.go is exercised by CI: the argv check runs on every leg, while
# the terraform round-trip check skips wherever no terraform binary is
# installed, which is every unit-test job today -- hashicorp/setup-terraform is
# only used by smoke-tests.yaml. Installing terraform for these jobs is tracked
# with the workflow work. The subdirectories under test/ are not included:
# those need provisioned infrastructure. ./cmd is included because it holds
# the download-launchpad verification tests added in PRODENG-3924; without
# it they exist but never run in CI.
.PHONY: unit-test
unit-test:
	go test -v --tags 'testing' $(TEST_FLAGS) ./pkg/... ./cmd/... ./test ./version/...

.PHONY: functional-test
functional-test:
	go test -v ./test/functional/... -timeout 20m

.PHONY: integration-test
integration-test:
	go test -v ./test/integration/... -timeout 20m

# TestCuttingEdgeCluster: rhel10, rocky10 and ubuntu26 as both managers and
# workers, on the newest MCR/MKE pairing. Run manually, or in CI on a push to
# main or the smoke-cutting-edge label.
.PHONY: smoke-cutting-edge
smoke-cutting-edge:
	go test -count=1 -v ./test/smoke/... -run TestCuttingEdgeCluster -timeout 50m

.PHONY: smoke-modern
smoke-modern:
	go test -count=1 -v ./test/smoke/... -run TestModernCluster -timeout 50m

.PHONY: smoke-legacy
smoke-legacy:
	go test -count=1 -v ./test/smoke/... -run TestLegacyCluster -timeout 50m

.PHONY: smoke-windows
smoke-windows:
	go test -count=1 -v ./test/smoke/... -run TestWindowsCluster -timeout 60m
.PHONY: smoke-fips
smoke-fips:
	go test -count=1 -v ./test/smoke/... -run TestFIPSCluster -timeout 60m
.PHONY: smoke-upgrade
smoke-upgrade:
	go test -count=1 -v ./test/smoke/... -run TestUpgrade -timeout 90m
.PHONY: smoke-swarm-pool
smoke-swarm-pool:
	go test -count=1 -v ./test/smoke/... -run TestSwarmAddrPoolCluster -timeout 60m
.PHONY: smoke-msr
smoke-msr:
	go test -count=1 -v ./test/smoke/... -run TestMSRCluster -timeout 60m
.PHONY: clean-launchpad-chart
clean-launchpad-chart:
	terraform -chdir=./examples/tf-aws/launchpad apply --auto-approve --destroy
