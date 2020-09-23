#!/bin/bash

set -e

cd test
. ./smoke.common.sh
trap cleanup EXIT

setup && downloadTools

set +e
../bin/launchpad --debug apply
RES=$?
set -e

if [ "$RES" -eq 0 ] && [ ! -z "$MUST_FAIL" ]; then
  echo "Command succeeded, but was expected to fail, exiting with error"
  exit 1
fi

if [ "$RES" -ne 0 ]; then
  if [ -z "$MUST_FAIL" ]; then
    exit $RES
  else
    echo "Command failed with $RES and MUST_FAIL set, returning success"
    exit 0
  fi
fi

echo "Apply succeeded, downloading bundle"
../bin/launchpad --debug download-bundle --username admin --password orcaorcaorca

# to source the env file succesfully we must be in the same directory
pushd ~/.mirantis-launchpad/cluster/$CLUSTER_NAME/bundle/admin/
source env.sh
popd

docker ps
docker ps --filter name=dtr
./kubectl get pods --all-namespaces
../bin/launchpad describe hosts
../bin/launchpad describe ucp
../bin/launchpad describe dtr
