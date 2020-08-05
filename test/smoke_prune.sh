#!/bin/bash

set -e

cd test
rm -f ./id_rsa_launchpad
ssh-keygen -t rsa -f ./id_rsa_launchpad -N ""

FOOTLOOSE_TEMPLATE=${FOOTLOOSE_TEMPLATE:-footloose.yaml.tpl}
CONFIG_TEMPLATE=${CONFIG_TEMPLATE:-cluster.yaml.tpl}
export LINUX_IMAGE=${LINUX_IMAGE:-"quay.io/footloose/ubuntu18.04"}
export UCP_VERSION=${UCP_VERSION:-"3.3.0"}
export UCP_IMAGE_REPO=${UCP_IMAGE_REPO:-"docker.io/docker"}
export DTR_COUNT=${DTR_COUNT:-"0"}
export DTR_VERSION=${DTR_VERSION:-"2.8.1"}
export DTR_IMAGE_REPO=${DTR_IMAGE_REPO:-"docker.io/docker"}
export ENGINE_VERSION=${ENGINE_VERSION:-"19.03.8"}
export CLUSTER_NAME=$BUILD_TAG
export DISABLE_TELEMETRY="true"
export ACCEPT_LICENSE="true"

function cleanup {
  echo -e "Cleaning up..."
  ./footloose delete
  docker volume prune -f
  docker network rm footloose-cluster 2> /dev/null
  ## Clean the local state
  rm -rf ~/.mirantis-launchpad/cluster/$CLUSTER_NAME
}

function downloadFootloose() {
  echo -e "Downloading tools for test..."
  OS=$(uname)
  if [ "$OS" == "Darwin" ]; then
      curl -L https://github.com/weaveworks/footloose/releases/download/0.6.3/footloose-0.6.3-darwin-x86_64.tar.gz > ./footloose.tar.gz
      tar -xvf footloose.tar.gz
  else
      curl -L https://github.com/weaveworks/footloose/releases/download/0.6.3/footloose-0.6.3-linux-x86_64 > ./footloose
  fi
}

echo -e "Creating footloose-cluster network..."
docker network create footloose-cluster --subnet 172.16.86.0/24 --gateway 172.16.86.1 --attachable 2> /dev/null

downloadFootloose

envsubst < "${FOOTLOOSE_TEMPLATE}" > footloose.yaml

chmod +x ./footloose
./footloose create

export UCP_MANAGER_IP=$(docker inspect ucp-manager0 --format='{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}')
envsubst < "${CONFIG_TEMPLATE}" > cluster.yaml
cat cluster.yaml

set +e
if ! ../bin/launchpad --debug apply ; then
  cleanup
  exit 1
fi

# Remove a node from the cluster.yaml and run apply with --prune
echo -e "Removing one DTR node from cluster.yaml..."
sed -i '25,30d' cluster.yaml
cat cluster.yaml

../bin/launchpad --debug apply --prune
result=$?

cleanup

exit $result
