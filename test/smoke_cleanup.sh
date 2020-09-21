#!/bin/bash

set -e

cd test
. ./smoke.common.sh
unset PRESERVE_CLUSTER
cleanup
