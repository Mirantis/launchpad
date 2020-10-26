#!/bin/bash

set -e

cd test
. ./smoke.common.sh
trap cleanup EXIT

setup

[ "${REUSE_CLUSTER}" = "" ] && ${LAUNCHPAD} --debug apply --config ${LAUNCHPAD_CONFIG}

../bin/launchpad describe hosts
../bin/launchpad describe ucp
../bin/launchpad describe dtr

export UCP_VERSION=${UCP_UPGRADE_VERSION:-"3.3.3"}
export UCP_IMAGE_REPO=${UCP_UPGRADE_IMAGE_REPO:-"docker.io/mirantis"}
export ENGINE_VERSION=${ENGINE_UPGRADE_VERSION:-"19.03.12"}
export DTR_VERSION=${DTR_UPGRADE_VERSION:"2.8.3"}

generateYaml

cat launchpad.yaml

${LAUNCHPAD} apply --config ${LAUNCHPAD_CONFIG}

${LAUNCHPAD} describe --config ${LAUNCHPAD_CONFIG} hosts
${LAUNCHPAD} describe --config ${LAUNCHPAD_CONFIG} ucp
${LAUNCHPAD} describe --config ${LAUNCHPAD_CONFIG} dtr
