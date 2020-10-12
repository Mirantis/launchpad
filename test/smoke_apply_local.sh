#!/bin/bash

set -e

export SMOKE_DIR="$( pwd -P )"

cd test
. ./smoke.common.sh
trap cleanup EXIT

echo SMOKE_DIR=$SMOKE_DIR

downloadFootloose
generateKey
createCluster
./footloose status worker0 -o json
WORKER_IP=$(./footloose status worker0 -o json | grep "\"ip\": \"172" | head -1 |cut -d\" -f4)

./footloose ssh root@manager0 "cd /launchpad/test; WORKER_IP=${WORKER_IP} DISABLE_TELEMETRY=true ACCEPT_LICENSE=true ../bin/launchpad --debug apply --config ${LAUNCHPAD_CONFIG}"
./footloose ssh root@manager0 "cd /launchpad/test; WORKER_IP=${WORKER_IP} DISABLE_TELEMETRY=true ACCEPT_LICENSE=true ../bin/launchpad describe --config ${LAUNCHPAD_CONFIG} hosts"
./footloose ssh root@manager0 "cd /launchpad/test; WORKER_IP=${WORKER_IP} DISABLE_TELEMETRY=true ACCEPT_LICENSE=true ../bin/launchpad describe --config ${LAUNCHPAD_CONFIG} ucp"
./footloose ssh root@manager0 "cd /launchpad/test; WORKER_IP=${WORKER_IP} DISABLE_TELEMETRY=true ACCEPT_LICENSE=true ../bin/launchpad describe --config ${LAUNCHPAD_CONFIG} dtr"
