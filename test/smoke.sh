#!/bin/bash

set -e

export LINUX_IMAGE=${LINUX_IMAGE:-"quay.io/footloose/ubuntu18.04"}
export UCP_VERSION=${UCP_VERSION:-"3.3.0"}
export ENGINE_VERSION=${ENGINE_VERSION:-"19.03.8"}
export CLUSTER_NAME=${BUILD_TAG:-"local"}

export ANALYTICS_DISABLED="true"

cd test
rm -f ./id_rsa_launchpad
ssh-keygen -t rsa -f ./id_rsa_launchpad -N ""

envsubst < cluster.yaml.tpl > cluster.yaml
envsubst < footloose.yaml.tpl > footloose.yaml

curl -L https://github.com/weaveworks/footloose/releases/download/0.6.3/footloose-0.6.3-linux-x86_64 > ./footloose
chmod +x ./footloose
./footloose create

curl -L https://storage.googleapis.com/kubernetes-release/release/v1.18.0/bin/linux/amd64/kubectl > ./kubectl
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

./kubectl get pods
result=$?
if [ $result -ne 0 ]; then
    echo "'kubectl get pods' returned non-zero exit code " $result
    exit $result
fi

unset DOCKER_HOST
unset DOCKER_CERT_PATH
unset DOCKER_TLS_VERIFY

./footloose delete
docker volume prune -f
## Clean the local state
rm -rf ~/.mirantis-launchpad/cluster/$CLUSTER_NAME
