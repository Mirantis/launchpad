#!/bin/bash

set -e

cd test
. ./smoke.common.sh
trap cleanup EXIT

[ "${REUSE_CLUSTER}" = "" ] && setup && ../bin/launchpad --debug apply

# Remove a node from the launchpad.yaml and run apply with --prune
echo -e "Removing one DTR node from launchpad.yaml..."
sed -i '25,30d' launchpad.yaml
cat launchpad.yaml

../bin/launchpad --debug apply --prune
