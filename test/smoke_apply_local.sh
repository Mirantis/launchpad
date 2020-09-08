#!/bin/bash

set -e

cd test
. ./smoke.common.sh
trap cleanup EXIT

setup && downloadTools

./footloose ssh root@manager0 "cd /launchpad; test/smoke_apply.sh"
