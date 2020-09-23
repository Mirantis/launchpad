#!/bin/bash

set -e

cd test
. ./smoke.common.sh
trap cleanup EXIT

[ "${REUSE_CLUSTER}" = "" ] && setup && ../bin/launchpad --debug apply
../bin/launchpad --debug reset --force
