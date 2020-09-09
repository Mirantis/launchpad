#!/bin/bash

set -e

cd test
. ./smoke.common.sh
trap cleanup EXIT

export SMOKE_DIR="$( cd "$(dirname "$0/..")" >/dev/null 2>&1 ; pwd -P )"

downloadFootloose
generateKey
createCluster
generateYaml

./footloose ssh root@manager0 "cd /launchpad/test; ../bin/launchpad --debug apply"
