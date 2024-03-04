#!/bin/bash

set -e

cd test
. ./smoke.common.sh
trap cleanup EXIT

setup && downloadTools

set +e
${LAUNCHPAD} apply --debug --config ${LAUNCHPAD_CONFIG}
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

echo "Testing exec"
${LAUNCHPAD} exec --debug -r manager echo hello from manager
${LAUNCHPAD} exec --debug --role worker echo hello from worker
${LAUNCHPAD} exec --debug --first echo hello from first host

echo "Apply succeeded, downloading bundle"
${LAUNCHPAD} client-config --config ${LAUNCHPAD_CONFIG}

# to source the env file succesfully we must be in the same directory
pushd ~/.mirantis-launchpad/cluster/$CLUSTER_NAME/bundle/admin/
source env.sh
popd

docker ps
docker ps --filter name=dtr
./kubectl get pods --all-namespaces
${LAUNCHPAD} describe --debug --config ${LAUNCHPAD_CONFIG} hosts
${LAUNCHPAD} describe --debug --config ${LAUNCHPAD_CONFIG} mke
${LAUNCHPAD} describe --debug --config ${LAUNCHPAD_CONFIG} msr
