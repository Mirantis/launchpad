#!/bin/bash

set -e

export SMOKE_DIR="$( pwd -P )"

cd test
. ./smoke.common.sh
trap cleanup EXIT

setup && downloadTools

MANAGER_IP=$(./footloose status manager0 -o json | grep "\"ip\": \"172" | head -1 |cut -d\" -f4)
WORKER_IP=$(./footloose status worker0 -o json | grep "\"ip\": \"172" | head -1 |cut -d\" -f4)
export MANAGER_IP
export WORKER_IP

BASTION_PORT=$(docker inspect mke-bastion0|grep -A3 22/tcp|grep HostPort|head -1|cut -d\" -f4)

eval $(ssh-agent -s)
ssh-add ./id_rsa_launchpad
ssh -o StrictHostKeyChecking=no -A -i ./id_rsa_launchpad -p ${BASTION_PORT} 2root@localhost "cd /launchpad/test; MANAGER_IP=${MANAGER_IP} WORKER_IP=${WORKER_IP} CLUSTER_NAME=${CLUSTER_NAME} MKE_VERSION=${MKE_VERSION} MKE_IMAGE_REPO=${MKE_IMAGE_REPO} MCR_VERSION=${MCR_VERSION} DISABLE_TELEMETRY=true ACCEPT_LICENSE=true ../bin/launchpad apply --debug --config ${LAUNCHPAD_CONFIG}"
ssh -o StrictHostKeyChecking=no -A -i ./id_rsa_launchpad -p ${BASTION_PORT} root@localhost "cd /launchpad/test; MANAGER_IP=${MANAGER_IP} WORKER_IP=${WORKER_IP} CLUSTER_NAME=${CLUSTER_NAME} MKE_VERSION=${MKE_VERSION} MKE_IMAGE_REPO=${MKE_IMAGE_REPO} MCR_VERSION=${MCR_VERSION} DISABLE_TELEMETRY=true ACCEPT_LICENSE=true ../bin/launchpad exec --debug --config ${LAUNCHPAD_CONFIG} --all --parallel hostname"
