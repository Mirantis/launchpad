#!/bin/bash

set -e

cd test
. ./smoke.common.sh
trap cleanup EXIT

setup

[ "${REUSE_CLUSTER}" = "" ] && ../bin/launchpad --debug apply

export UCP_VERSION=${UCP_UPGRADE_VERSION:-"3.3.2"}
export ENGINE_VERSION=${ENGINE_UPGRADE_VERSION:-"19.03.8"}
export DTR_VERSION=${DTR_UPGRADE_VERSION:"2.8.2"}

generateYaml

../bin/launchpad --debug apply
