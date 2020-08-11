#!/bin/bash

set -e

FOOTLOOSE_TEMPLATE=${FOOTLOOSE_TEMPLATE:-footloose.yaml.tpl}
CONFIG_TEMPLATE=${CONFIG_TEMPLATE:-cluster.yaml.tpl}
export LINUX_IMAGE=${LINUX_IMAGE:-"quay.io/footloose/ubuntu18.04"}
export UCP_VERSION=${UCP_VERSION:-"3.3.0"}
export UCP_IMAGE_REPO=${UCP_IMAGE_REPO:-"docker.io/docker"}
export DTR_VERSION=${DTR_VERSION:-"2.8.1"}
export DTR_IMAGE_REPO=${DTR_IMAGE_REPO:-"docker.io/docker"}
export ENGINE_VERSION=${ENGINE_VERSION:-"19.03.8"}
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

    ## For instances when footloose doesn't get execute permissions prior
    ## to a failure
    chmod +x ./footloose
    ./footloose delete
    docker volume prune -f
    docker network rm footloose-cluster 2> /dev/null
    ## Clean the local state
    rm -rf ~/.mirantis-launchpad/cluster/$CLUSTER_NAME
    rm ./kubectl
    rm ./footloose
}
trap cleanup EXIT

function downloadTools() {
    OS=$(uname)
    echo -e "Downloading tools for test..."
    if [ "$OS" == "Darwin" ]; then
        [ -f footloose ] || (curl -L https://github.com/weaveworks/footloose/releases/download/0.6.3/footloose-0.6.3-darwin-x86_64.tar.gz > ./footloose.tar.gz && tar -xvf footloose.tar.gz)

        [ -f kubectl ] || curl -L https://storage.googleapis.com/kubernetes-release/release/v1.18.0/bin/darwin/amd64/kubectl > ./kubectl
    else
        [ -f footloose ] || curl -L https://github.com/weaveworks/footloose/releases/download/0.6.3/footloose-0.6.3-linux-x86_64 > ./footloose

        [ -f kubectl ] || curl -L https://storage.googleapis.com/kubernetes-release/release/v1.18.0/bin/linux/amd64/kubectl > ./kubectl
    fi
}

cd test
rm -f ./id_rsa_launchpad
ssh-keygen -t rsa -f ./id_rsa_launchpad -N ""

# cleanup any existing cluster
envsubst < footloose-dtr.yaml.tpl > footloose.yaml
./footloose delete && docker volume prune

envsubst < "${FOOTLOOSE_TEMPLATE}" > footloose.yaml
echo -e "Creating footloose-cluster network..."
docker network inspect footloose-cluster || docker network create footloose-cluster --subnet 172.16.86.0/24 --gateway 172.16.86.1 --attachable 2> /dev/null

downloadTools

chmod +x ./footloose
./footloose create

export UCP_MANAGER_IP=$(docker inspect ucp-manager0 --format='{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}')
envsubst < "${CONFIG_TEMPLATE}" > cluster.yaml
cat cluster.yaml

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

docker ps --filter name=dtr
result=$?
if [ $result -ne 0 ]; then
    echo "'docker ps --filter name=dtr' returned non-zero exit code " $result
    exit $result
fi

./kubectl get pods --all-namespaces
result=$?
if [ $result -ne 0 ]; then
    echo "'kubectl get pods' returned non-zero exit code " $result
    exit $result
fi
