#!/bin/bash

set -e

cd test
. ./smoke.common.sh
trap cleanup EXIT

setup

[ "${REUSE_CLUSTER}" = "" ] && ../bin/launchpad --debug apply

export UCP_VERSION=${UCP_UPGRADE_VERSION:-"3.3.2"}
export UCP_IMAGE_REPO=${UCP_UPGRADE_IMAGE_REPO:-"docker.io/mirantis"}
export ENGINE_VERSION=${ENGINE_UPGRADE_VERSION:-"19.03.8"}
export DTR_VERSION=${DTR_UPGRADE_VERSION:"2.8.2"}

generateYaml

cat cluster.yaml

../bin/launchpad --debug apply
