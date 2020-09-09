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
export WORKER_IP=$(./footloose status worker0 -o json | grep "\"ip\":" | head -1|cut -d\" -f4)
generateYaml

./footloose ssh root@manager0 "cd /launchpad/test; DISABLE_TELEMETRY=true ACCEPT_LICENSE=true ../bin/launchpad --debug apply"
