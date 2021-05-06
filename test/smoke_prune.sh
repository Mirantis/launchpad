#!/bin/bash

set -e

cd test
. ./smoke.common.sh
trap cleanup EXIT

setup

[ "${REUSE_CLUSTER}" = "" ] && ${LAUNCHPAD} apply --debug --config ${LAUNCHPAD_CONFIG}

UNAME=$(uname)
if [ "${UNAME}" = "Darwin" ]; then
  SEDOPTS="-i -e"
else
  SEDOPTS="-i"
fi

# Remove a node from the launchpad.yaml
echo -e "Removing one MSR node from launchpad.yaml..."
sed $SEDOPTS '/REMOVE_THIS/d' ${LAUNCHPAD_CONFIG}
cat ${LAUNCHPAD_CONFIG}

${LAUNCHPAD} describe --debug --config ${LAUNCHPAD_CONFIG} hosts

echo "Running with prune: false"
${LAUNCHPAD} apply --debug --config ${LAUNCHPAD_CONFIG}

# Flip prune to true
echo -e "Changing prune: false to prune: true..."
sed $SEDOPTS 's/prune: false/prune: true/' ${LAUNCHPAD_CONFIG}
cat ${LAUNCHPAD_CONFIG}

echo "Running with prune: true"
${LAUNCHPAD} apply --debug --config ${LAUNCHPAD_CONFIG}

${LAUNCHPAD} describe --debug --config ${LAUNCHPAD_CONFIG} hosts
${LAUNCHPAD} describe --debug --config ${LAUNCHPAD_CONFIG} mke
${LAUNCHPAD} describe --debug --config ${LAUNCHPAD_CONFIG} msr
