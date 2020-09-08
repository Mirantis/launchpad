#!/bin/sh

set -e

cd test
. ./smoke.common.sh
trap cleanup EXIT

setup && downloadTools

footloose ssh root@manager0 "cd /launchpad; bash test/smoke_apply.sh"
