#!/bin/bash

set -e

cd test
. ./smoke.common.sh
trap cleanup EXIT

setup

[ "${REUSE_CLUSTER}" = "" ] && ${LAUNCHPAD} --debug apply --config ${LAUNCHPAD_CONFIG}

../bin/launchpad describe hosts
../bin/launchpad describe mke
../bin/launchpad describe msr

export MKE_VERSION=$MKE_UPGRADE_VERSION:-"3.3.3"}
export MKE_IMAGE_REPO=${MKE_UPGRADE_IMAGE_REPO:-"docker.io/mirantis"}
export ENGINE_VERSION=${ENGINE_UPGRADE_VERSION:-"19.03.12"}
export MSR_VERSION=${MSR_UPGRADE_VERSION:"2.8.3"}

generateYaml

cat launchpad.yaml

${LAUNCHPAD} apply --config ${LAUNCHPAD_CONFIG}

${LAUNCHPAD} describe --config ${LAUNCHPAD_CONFIG} hosts
${LAUNCHPAD} describe --config ${LAUNCHPAD_CONFIG} mke
${LAUNCHPAD} describe --config ${LAUNCHPAD_CONFIG} msr
