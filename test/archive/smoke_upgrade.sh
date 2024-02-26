#!/bin/bash

set -e

cd test
. ./smoke.common.sh
trap cleanup EXIT

setup

[ "${REUSE_CLUSTER}" = "" ] && ${LAUNCHPAD} apply --debug --config ${LAUNCHPAD_CONFIG}

../bin/launchpad describe --debug hosts
../bin/launchpad describe --debug mke
../bin/launchpad describe --debug msr

export MKE_VERSION=$MKE_UPGRADE_VERSION:-"3.3.3"}
export MKE_IMAGE_REPO=${MKE_UPGRADE_IMAGE_REPO:-"docker.io/mirantis"}
export MCR_VERSION=${MCR_UPGRADE_VERSION:-"19.03.12"}
export MSR_VERSION=${MSR_UPGRADE_VERSION:"2.8.3"}

generateYaml

cat launchpad.yaml

${LAUNCHPAD} apply --debug --config ${LAUNCHPAD_CONFIG}

${LAUNCHPAD} describe --debug --config ${LAUNCHPAD_CONFIG} hosts
${LAUNCHPAD} describe --debug --config ${LAUNCHPAD_CONFIG} mke
${LAUNCHPAD} describe --debug --config ${LAUNCHPAD_CONFIG} msr
