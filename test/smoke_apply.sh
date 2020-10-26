#!/bin/bash

set -e

cd test
. ./smoke.common.sh
trap cleanup EXIT

setup && downloadTools

set +e
${LAUNCHPAD} apply --config ${LAUNCHPAD_CONFIG}
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
../bin/launchpad --debug exec -r manager echo hello from manager
../bin/launchpad --debug exec --role worker echo hello from worker
../bin/launchpad --debug exec --first echo hello from first host

echo "Apply succeeded, downloading bundle"
${LAUNCHPAD} download-bundle --username admin --password orcaorcaorca --config ${LAUNCHPAD_CONFIG}

# to source the env file succesfully we must be in the same directory
pushd ~/.mirantis-launchpad/cluster/$CLUSTER_NAME/bundle/admin/
source env.sh
popd

docker ps
docker ps --filter name=dtr
./kubectl get pods --all-namespaces
${LAUNCHPAD} describe --config ${LAUNCHPAD_CONFIG} hosts
${LAUNCHPAD} describe --config ${LAUNCHPAD_CONFIG} ucp
${LAUNCHPAD} describe --config ${LAUNCHPAD_CONFIG} dtr
