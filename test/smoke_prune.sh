#!/bin/bash

set -e

cd test
. ./smoke.common.sh
trap cleanup EXIT

[ "${REUSE_CLUSTER}" = "" ] && setup && ../bin/launchpad --debug apply

UNAME=$(uname)
if [ "${UNAME}" = "Darwin" ]; then
  SEDOPTS="-i -e"
else
  SEDOPTS="-i"
fi

# Remove a node from the launchpad.yaml
echo -e "Removing one DTR node from launchpad.yaml..."
sed $SEDOPTS '/REMOVE_THIS/d' launchpad.yaml
cat launchpad.yaml

../bin/launchpad describe hosts

echo "Running with prune: false"
../bin/launchpad --debug apply

# Flip prune to true
echo -e "Changing prune: false to prune: true..."
sed $SEDOPTS 's/prune: false/prune: true/' launchpad.yaml
cat launchpad.yaml

echo "Running with prune: true"
../bin/launchpad --debug apply

../bin/launchpad describe hosts
../bin/launchpad describe ucp
../bin/launchpad describe dtr
