#!/bin/bash

set -e

CONFIG_TEMPLATE=${CONFIG_TEMPLATE:-cluster.yaml.tpl}
export LINUX_IMAGE=${LINUX_IMAGE:-"quay.io/footloose/ubuntu18.04"}
export UCP_VERSION=${UCP_VERSION:-"3.3.0"}
export UCP_IMAGE_REPO=${UCP_IMAGE_REPO:-"docker.io/docker"}
export ENGINE_VERSION=${ENGINE_VERSION:-"19.03.8"}
export CLUSTER_NAME=${BUILD_TAG:-"local"}

export ANALYTICS_DISABLED="true"

if [ ! -z "${REGISTRY_CREDS_USR}" ]; then
  export REGISTRY_USERNAME="${REGISTRY_CREDS_USR}"
fi
if [ ! -z "${REGISTRY_CREDS_PSW}" ]; then
  export REGISTRY_PASSWORD="${REGISTRY_CREDS_PSW}"
fi

function cleanup() {
    unset DOCKER_HOST
    unset DOCKER_CERT_PATH
    unset DOCKER_TLS_VERIFY

    ./footloose delete
    docker volume prune -f
    ## Clean the local state
    rm -rf ~/.mirantis-launchpad/cluster/$CLUSTER_NAME
    rm ./kubectl
    rm ./footloose
}
trap cleanup EXIT

function downloadTools() {
    OS=$(uname)
    if [ "$OS" == "Darwin" ]; then
        curl -L https://github.com/weaveworks/footloose/releases/download/0.6.3/footloose-0.6.3-darwin-x86_64.tar.gz > ./footloose.tar.gz
        tar -xvf footloose.tar.gz

        curl -L https://storage.googleapis.com/kubernetes-release/release/v1.18.0/bin/darwin/amd64/kubectl > ./kubectl
    else
        curl -L https://github.com/weaveworks/footloose/releases/download/0.6.3/footloose-0.6.3-linux-x86_64 > ./footloose

        curl -L https://storage.googleapis.com/kubernetes-release/release/v1.18.0/bin/linux/amd64/kubectl > ./kubectl
    fi
}

cd test
rm -f ./id_rsa_launchpad
ssh-keygen -t rsa -f ./id_rsa_launchpad -N ""

envsubst < "${CONFIG_TEMPLATE}" > cluster.yaml
envsubst < footloose.yaml.tpl > footloose.yaml

downloadTools

chmod +x ./footloose
./footloose create

chmod +x ./kubectl

set +e
../bin/launchpad --debug apply
result=$?

if [ $result -ne 0 ]; then
    echo "'launchpad apply' returned non-zero exit code " $result
    exit $result
fi

../bin/launchpad --debug download-bundle --username admin --password orcaorcaorca
# to source the env file succesfully we must be in the same directory
cd ~/.mirantis-launchpad/cluster/$CLUSTER_NAME/bundle/admin/
source env.sh
cd -

docker ps
result=$?
if [ $result -ne 0 ]; then
    echo "'docker ps' returned non-zero exit code " $result
    exit $result
fi

./kubectl get pods --all-namespaces
result=$?
if [ $result -ne 0 ]; then
    echo "'kubectl get pods' returned non-zero exit code " $result
    exit $result
fi
