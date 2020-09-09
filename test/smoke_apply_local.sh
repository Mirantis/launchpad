#!/bin/bash

set -e

cd test
. ./smoke.common.sh
trap cleanup EXIT

downloadFootloose
generateKey
createCluster
generateYaml

./footloose ssh root@manager0 "cd /launchpad/test; ../bin/launchpad --debug apply"
