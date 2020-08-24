#!/bin/bash

set -e

cd test
. ./smoke.common.sh
trap cleanup EXIT

setup

[ "${REUSE_CLUSTER}" = "" ] && ../bin/launchpad --debug apply

# Remove a node from the cluster.yaml and run apply with --prune
echo -e "Removing one DTR node from cluster.yaml..."
sed -i '25,30d' cluster.yaml
cat cluster.yaml

../bin/launchpad --debug apply --prune
