#!/bin/bash

set -e

cd test
. ./smoke.common.sh
trap cleanup EXIT

setup

[ "${REUSE_CLUSTER}" = "" ] && ${LAUNCHPAD} apply --debug --config - < ${LAUNCHPAD_CONFIG}
${LAUNCHPAD} reset --debug --config ${LAUNCHPAD_CONFIG} --force
${LAUNCHPAD} describe --debug --config ${LAUNCHPAD_CONFIG} hosts
${LAUNCHPAD} describe --debug --config ${LAUNCHPAD_CONFIG} mke
${LAUNCHPAD} describe --debug --config ${LAUNCHPAD_CONFIG} msr
