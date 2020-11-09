#!/bin/bash

set -e

cd test
. ./smoke.common.sh
trap cleanup EXIT

setup

[ "${REUSE_CLUSTER}" = "" ] && ${LAUNCHPAD} apply --config - < ${LAUNCHPAD_CONFIG}
${LAUNCHPAD} reset --config ${LAUNCHPAD_CONFIG} --force
${LAUNCHPAD} describe --config ${LAUNCHPAD_CONFIG} hosts
${LAUNCHPAD} describe --config ${LAUNCHPAD_CONFIG} ucp
${LAUNCHPAD} describe --config ${LAUNCHPAD_CONFIG} dtr
