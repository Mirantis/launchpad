#!/bin/bash

set -e

cd test
rm -f ./id_rsa_launchpad
ssh-keygen -t rsa -f ./id_rsa_launchpad -N ""

export LINUX_IMAGE=${LINUX_IMAGE:-"quay.io/footloose/ubuntu18.04"}
export UCP_VERSION=${UCP_VERSION:-"3.2.6"}
export UCP_IMAGE_REPO=${UCP_IMAGE_REPO:-"docker.io/docker"}
export ENGINE_VERSION=${ENGINE_VERSION:-"19.03.5"}
export CLUSTER_NAME=$BUILD_TAG
export TELEMETRY_DISABLED="true"
export ACCEPT_LICENSE="true"

envsubst < cluster.yaml.tpl > cluster.yaml
envsubst < footloose.yaml.tpl > footloose.yaml
cat cluster.yaml

function cleanup {
  ./footloose delete
  docker volume prune -f
  ## Clean the local state
  rm -rf ~/.mirantis-launchpad/cluster/$CUSTER_NAME
}

function downloadFootloose() {
  OS=$(uname)
  if [ "$OS" == "Darwin" ]; then
      curl -L https://github.com/weaveworks/footloose/releases/download/0.6.3/footloose-0.6.3-darwin-x86_64.tar.gz > ./footloose.tar.gz
      tar -xvf footloose.tar.gz
  else
      curl -L https://github.com/weaveworks/footloose/releases/download/0.6.3/footloose-0.6.3-linux-x86_64 > ./footloose
  fi
}

downloadFootloose

chmod +x ./footloose
./footloose create

set +e
if ! ../bin/launchpad --debug apply ; then
  cleanup
  exit 1
fi

export UCP_VERSION=${UCP_UPGRADE_VERSION:-"3.3.1"}
export ENGINE_VERSION=${ENGINE_UPGRADE_VERSION:-"19.03.8"}
envsubst < cluster.yaml.tpl > cluster.yaml
envsubst < footloose.yaml.tpl > footloose.yaml
cat cluster.yaml

../bin/launchpad --debug apply
result=$?

cleanup

exit $result
