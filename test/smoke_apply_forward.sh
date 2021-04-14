#!/bin/bash

set -e

export SMOKE_DIR="$( pwd -P )"

cd test
. ./smoke.common.sh
trap cleanup EXIT

setup && downloadTools

MANAGER_IP=$(./footloose status manager0 -o json | grep "\"ip\": \"172" | head -1 |cut -d\" -f4)
WORKER_IP=$(./footloose status worker0 -o json | grep "\"ip\": \"172" | head -1 |cut -d\" -f4)

eval $(ssh-agent -s)
ssh-add ./id_rsa_launchpad
ssh -A -i ./id_rsa_launchpad -p 9022 2root@localhost "cd /launchpad/test; MANAGER_IP=${MANAGER_IP} WORKER_IP=${WORKER_IP} CLUSTER_NAME=${CLUSTER_NAME} MKE_VERSION=${MKE_VERSION} MKE_IMAGE_REPO=${MKE_IMAGE_REPO} MCR_VERSION=${MCR_VERSION} DISABLE_TELEMETRY=true ACCEPT_LICENSE=true ../bin/launchpad apply --debug --config ${LAUNCHPAD_CONFIG}"
ssh -A -i ./id_rsa_launchpad -p 9022 root@localhost "cd /launchpad/test; MANAGER_IP=${MANAGER_IP} WORKER_IP=${WORKER_IP} CLUSTER_NAME=${CLUSTER_NAME} MKE_VERSION=${MKE_VERSION} MKE_IMAGE_REPO=${MKE_IMAGE_REPO} MCR_VERSION=${MCR_VERSION} DISABLE_TELEMETRY=true ACCEPT_LICENSE=true ../bin/launchpad exec --debug --config ${LAUNCHPAD_CONFIG} --all --parallel hostname"
