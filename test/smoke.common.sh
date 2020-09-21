#!/bin/bash

FOOTLOOSE_TEMPLATE=${FOOTLOOSE_TEMPLATE:-"footloose.yaml.tpl"}
CONFIG_TEMPLATE=${CONFIG_TEMPLATE:-"launchpad.yaml.tpl"}
export LINUX_IMAGE=${LINUX_IMAGE:-"quay.io/footloose/ubuntu18.04"}
export UCP_VERSION=${UCP_VERSION:-"3.3.3"}
export UCP_IMAGE_REPO=${UCP_IMAGE_REPO:-"docker.io/mirantis"}
export DTR_VERSION=${DTR_VERSION:-"2.8.3"}
export DTR_IMAGE_REPO=${DTR_IMAGE_REPO:-"docker.io/mirantis"}
export ENGINE_VERSION=${ENGINE_VERSION:-"19.03.12"}
export PRESERVE_CLUSTER=${PRESERVE_CLUSTER:-""}
export REUSE_CLUSTER=${REUSE_CLUSTER:-""}
export CLUSTER_NAME=${BUILD_TAG:-"local"}
export DISABLE_TELEMETRY="true"
export ACCEPT_LICENSE="true"

if [ ! -z "${REGISTRY_CREDS_USR}" ]; then
  export REGISTRY_USERNAME="${REGISTRY_CREDS_USR}"
fi
if [ ! -z "${REGISTRY_CREDS_PSW}" ]; then
  export REGISTRY_PASSWORD="${REGISTRY_CREDS_PSW}"
fi

function cleanup() {
    echo -e "Cleaning up..."
    unset DOCKER_HOST
    unset DOCKER_CERT_PATH
    unset DOCKER_TLS_VERIFY

    if [ -z "${PRESERVE_CLUSTER}" ]; then
      deleteCluster
      docker network rm footloose-cluster 2> /dev/null || true
      ## Clean the local state
      rm -rf ~/.mirantis-launchpad/cluster/$CLUSTER_NAME || true
      rm -f ./kubectl || true
      rm -f ./footloose || true
    fi
}

function downloadFootloose() {
  if [ ! -f footloose ]; then
    echo -e "Downloading footloose"
    OS=$(uname)
    if [ "$OS" == "Darwin" ]; then
        curl -L https://github.com/weaveworks/footloose/releases/download/0.6.3/footloose-0.6.3-darwin-x86_64.tar.gz > ./footloose.tar.gz
        tar -xvf footloose.tar.gz
    else
        curl -L https://github.com/weaveworks/footloose/releases/download/0.6.3/footloose-0.6.3-linux-x86_64 > ./footloose
    fi
    chmod +x ./footloose
    ./footloose version
  fi
  echo -e "Creating footloose-cluster network..."
  docker network inspect footloose-cluster || docker network create footloose-cluster --subnet 172.16.86.0/24 --gateway 172.16.86.1 --attachable 2> /dev/null
}

function downloadTools() {
    downloadFootloose

    echo -e "Downloading tools for test..."
    OS=$(uname)
    if [ "$OS" == "Darwin" ]; then
      [ -f kubectl ] || (curl -L https://storage.googleapis.com/kubernetes-release/release/v1.18.0/bin/darwin/amd64/kubectl > ./kubectl && chmod +x ./kubectl)
    else
      [ -f kubectl ] || (curl -L https://storage.googleapis.com/kubernetes-release/release/v1.18.0/bin/linux/amd64/kubectl > ./kubectl && chmod +x ./kubectl)
    fi
    ./kubectl version --client=true
}

function generateKey() {
  rm -f ./id_rsa_launchpad
  ssh-keygen -t rsa -f ./id_rsa_launchpad -N ""
}

function deleteCluster() {
  # cleanup any existing cluster
  envsubst < footloose-dtr.yaml.tpl > footloose.yaml
  ./footloose delete && docker volume prune -f
}

function createCluster() {
  envsubst < "${FOOTLOOSE_TEMPLATE}" > footloose.yaml
  ./footloose create
}

function generateYaml() {
  export UCP_MANAGER_IP=$(docker inspect ucp-manager0 --format='{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}')
  envsubst < "${CONFIG_TEMPLATE}" > launchpad.yaml
  cat launchpad.yaml
}

function setup() {
  if [ -z "${REUSE_CLUSTER}" ]; then
    generateKey
    downloadFootloose
    deleteCluster
    createCluster
  fi
  generateYaml
}
