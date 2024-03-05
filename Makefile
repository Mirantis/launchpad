GIT_COMMIT = $(shell git rev-parse --short=7 HEAD)

VOLUME_MOUNTS=-v "$(CURDIR):/v"
SIGN?=docker run --rm -i $(VOLUME_MOUNTS) -e SM_API_KEY -e SM_CLIENT_CERT_PASSWORD -e SM_CLIENT_CERT_FILE -v "$(SM_CLIENT_CERT_FILE):$(SM_CLIENT_CERT_FILE)" -w "/v" registry.mirantis.com/prodeng/digicert-keytools-jsign:latest sign

GO=$(shell which go)

ARTIFACTS_FOLDER=dist/artifacts

CHECKSUM=$(shell which sha256sum)
CHECKSUM_FILE?=checksums.txt

GOLANGCI_LINT?=docker run -t --rm -v "$(CURDIR):/data" -w "/data" golangci/golangci-lint:latest golangci-lint

# "Signing Windows binaries"
sign-win:
	for f in `find $(ARTIFACTS_FOLDER)/*.exe`; do echo $(SIGN) "$$f"; done

clean:
	rm -f dist

# build all the binaries for release, using goreleaser, but
# don't use any of the other features of goreleaser - because
# we need to use digicert to sign the binaries first, and
# goreleaser doesn't allow for that (some pro features may
# allow it in a round about way.)
build-release:
	goreleaser build --clean --config=.goreleaser.build.yml

.PHONY: unit-test
unit-test:
	$(GO) test -v ./pkg/...

.PHONY: smoke-small
smoke-small:
	go test -v ./test/smoke/... -run TestSmallCluster -timeout 20m

.PHONY: smoke-full
smoke-full:
	go test -v ./test/smoke/... -run TestSupportedMatrixCluster -timeout 30m

.PHONY: clean-launchpad-chart
clean-launchpad-chart:
	terraform -chdir=./examples/tf-aws/launchpad apply --auto-approve --destroy

checksum-release: build-release
	cd $(ARTIFACTS_FOLDER) && rm -rf $(CHECKSUM_FILE) && $(CHECKSUM) * > $(CHECKSUM_FILE)

# Local build of the plugin. This saves time building platforms that you
# won't test locally. To use it, find the path to your build binary path
# and alias it.
.PHONY: local
local:
	GORELEASER_CURRENT_TAG="$(LOCAL_TAG)" goreleaser build --clean --single-target --skip=validate --snapshot  --config .goreleaser.build.yml

# run the Github release script after a buil
release: build-release sign-win checksum-release
	./release.sh

.PHONY: lint
lint:
	$(GOLANGCI_LINT) run
