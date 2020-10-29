#!/bin/bash

set -e

cd test
. ./smoke.common.sh
trap cleanup EXIT

setup

[ "${REUSE_CLUSTER}" = "" ] && ../bin/launchpad apply

UNAME=$(uname)
if [ "${UNAME}" = "Darwin" ]; then
  SEDOPTS="-i -e"
else
  SEDOPTS="-i"
fi

# Remove a node from the launchpad.yaml
echo -e "Removing one DTR node from launchpad.yaml..."
sed $SEDOPTS '/REMOVE_THIS/d' launchpad.yaml

echo "Running with prune: false"
../bin/launchpad apply

echo "Running with prune"
../bin/launchpad apply --prune

