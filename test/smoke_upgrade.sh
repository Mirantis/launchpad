#!/bin/bash

set -e

cd test
rm -f ./id_rsa_launchpad
ssh-keygen -t rsa -f ./id_rsa_launchpad -N ""

export LINUX_IMAGE=${LINUX_IMAGE:-"quay.io/footloose/ubuntu18.04"}
export UCP_VERSION=${UCP_VERSION:-"3.2.6"}
export ENGINE_VERSION=${ENGINE_VERSION:-"19.03.5"}
export CLUSTER_NAME=$BUILD_TAG
export ANALYTICS_DISABLED="true"

envsubst < cluster.yaml.tpl > cluster.yaml
envsubst < footloose.yaml.tpl > footloose.yaml
cat cluster.yaml

function cleanup {
  ./footloose delete
  docker volume prune -f
  ## Clean the local state
  rm -rf ~/.mirantis-launchpad/cluster/$CUSTER_NAME
}

curl -L https://github.com/weaveworks/footloose/releases/download/0.6.3/footloose-0.6.3-linux-x86_64 > ./footloose
chmod +x ./footloose
./footloose create

set +e
if ! ../bin/launchpad --debug install ; then
  cleanup
  exit 1
fi

export UCP_VERSION=${UCP_UPGRADE_VERSION:-"3.3.0-rc1"}
export ENGINE_VERSION=${ENGINE_UPGRADE_VERSION:-"19.03.8-rc1"}
envsubst < cluster.yaml.tpl > cluster.yaml
envsubst < footloose.yaml.tpl > footloose.yaml
cat cluster.yaml

../bin/launchpad --debug install
result=$?

cleanup

exit $result
