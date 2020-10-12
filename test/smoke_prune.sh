#!/bin/bash

set -e

cd test
. ./smoke.common.sh
trap cleanup EXIT

[ "${REUSE_CLUSTER}" = "" ] && setup && ${LAUNCHPAD} apply --config ${LAUNCHPAD_CONFIG}

# Remove a node from the launchpad.yaml and run apply with --prune
echo -e "Removing one DTR node from launchpad.yaml..."
sed -i '25,30d' ${LAUNCHPAD_CONFIG}
cat ${LAUNCHPAD_CONFIG}

${LAUNCHPAD} describe --config ${LAUNCHPAD_CONFIG} hosts
${LAUNCHPAD} apply --config ${LAUNCHPAD_CONFIG} --prune
${LAUNCHPAD} describe --config ${LAUNCHPAD_CONFIG} hosts
${LAUNCHPAD} describe --config ${LAUNCHPAD_CONFIG} ucp
${LAUNCHPAD} describe --config ${LAUNCHPAD_CONFIG} dtr
