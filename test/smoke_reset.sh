#!/bin/bash

set -e

cd test
. ./smoke.common.sh
trap cleanup EXIT

setup

[ "${REUSE_CLUSTER}" = "" ] && ../bin/launchpad --debug apply
../bin/launchpad --debug reset
../bin/launchpad describe hosts
../bin/launchpad describe ucp
../bin/launchpad describe dtr
