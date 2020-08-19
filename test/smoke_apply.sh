#!/bin/bash

set -e

cd test
. ./smoke.common.sh
trap cleanup EXIT

setup && downloadTools

../bin/launchpad --debug apply

../bin/launchpad --debug download-bundle --username admin --password orcaorcaorca

# to source the env file succesfully we must be in the same directory
pushd ~/.mirantis-launchpad/cluster/$CLUSTER_NAME/bundle/admin/
source env.sh
popd

docker ps
docker ps --filter name=dtr
./kubectl get pods --all-namespaces
