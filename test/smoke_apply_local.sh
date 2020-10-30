#!/bin/bash

set -e

export SMOKE_DIR="$( pwd -P )"

cd test
. ./smoke.common.sh
trap cleanup EXIT

echo SMOKE_DIR=$SMOKE_DIR

setup && downloadTools

./footloose status worker0 -o json
export WORKER_IP=$(./footloose status worker0 -o json | grep "\"ip\": \"172" | head -1 |cut -d\" -f4)
echo WORKER_IP=$WORKER_IP
generateYaml

./footloose ssh root@manager0 "cd /launchpad/test; DISABLE_TELEMETRY=true ACCEPT_LICENSE=true ../bin/launchpad --debug apply"
