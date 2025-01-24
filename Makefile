
GO=$(shell which go)

RELEASE_FOLDER=dist/release

CHECKSUM=$(shell which sha256sum)

VOLUME_MOUNTS=-v "$(CURDIR):/v"
SIGN?=docker run --rm -i $(VOLUME_MOUNTS) -e SM_API_KEY -e SM_CLIENT_CERT_PASSWORD -e SM_CLIENT_CERT_FILE -v "$(SM_CLIENT_CERT_FILE):$(SM_CLIENT_CERT_FILE)" -w "/v" registry.mirantis.com/prodeng/digicert-keytools-jsign:latest sign

GOLANGCI_LINT?=docker run -t --rm -v "$(CURDIR):/data" -w "/data" golangci/golangci-lint:latest golangci-lint

SEGMENT_TOKEN?=""

.PHONY: clean
clean:
	rm -fr dist

# Sign release binaries (Windows)
# (build may need to be run in a separate make run)
.PHONY: sign-release
sign-release: $(RELEASE_FOLDER)
	for f in `find $(RELEASE_FOLDER)/*.exe`; do echo $(SIGN) "$$f"; done

# Force a clean build of the artifacts by first cleaning
# and then building
.PHONY: build-release
build-release: clean $(RELEASE_FOLDER)
# build all the binaries for release, using goreleaser, but
# don't use any of the other features of goreleaser - because
# we need to use digicert to sign the binaries first, and
# goreleaser doesn't allow for that (some pro features may
# allow it in a round about way.)
#
# If you are using more than one tag for a commit, then use
# the GORELEASER_CURRENT_TAG env var to clarify the version to
# avoid having the wrong tag version applied
$(RELEASE_FOLDER):
	goreleaser build --clean --config=.goreleaser.release.yml

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

# Local build of the plugin. This saves time building platforms that you
# won't test locally. To use it, find the path to your build binary path
# and alias it.
.PHONY: local
local:
	SEGMENT_TOKEN=${SEGMENT_TOKEN} goreleaser build --clean --single-target --skip=validate --snapshot --config .goreleaser.local.yml

# run linting
.PHONY: lint
lint:
	$(GOLANGCI_LINT) run

# Testing related targets

.PHONY: unit-test
unit-test:
	$(GO) test -v --tags 'testing' ./pkg/...

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
