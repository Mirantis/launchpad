#!/bin/bash

set -e

cd test
. ./smoke.common.sh
trap cleanup EXIT

setup && downloadTools

BASTION_IP=$(./footloose status bastion0 -o json | grep "\"ip\": \"172" | head -1 |cut -d\" -f4)
BASTION_PORT=$(docker inspect mke-bastion0|grep -A3 22/tcp|grep HostPort|head -1|cut -d\" -f4)
MANAGER_IP=$(./footloose status manager0 -o json | grep "\"ip\": \"172" | head -1 |cut -d\" -f4)
WORKER_IP=$(./footloose status worker0 -o json | grep "\"ip\": \"172" | head -1 |cut -d\" -f4)
export MANAGER_IP
export WORKER_IP
export BASTION_IP
export BASTION_PORT

${LAUNCHPAD} apply --debug --config ${LAUNCHPAD_CONFIG}

echo "Testing exec"
${LAUNCHPAD} exec --debug --all --parallel --config ${LAUNCHPAD_CONFIG} hostname
