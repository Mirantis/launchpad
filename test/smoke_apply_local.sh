#!/bin/bash

set -e

cd test
. ./smoke.common.sh
trap cleanup EXIT

setup && downloadTools

./footloose ssh root@manager0 "cd /launchpad; REUSE_CLUSTER=true test/smoke_apply.sh"
