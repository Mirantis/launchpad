#!/bin/bash

FOOTLOOSE_TEMPLATE=${FOOTLOOSE_TEMPLATE:-"footloose.yaml.tpl"}
LAUNCHPAD_CONFIG=${LAUNCHPAD_CONFIG:-"launchpad.yaml"}
LAUNCHPAD="../bin/launchpad"

export LINUX_IMAGE=${LINUX_IMAGE:-"quay.io/footloose/ubuntu18.04"}
export MKE_VERSION=${MKE_VERSION:-"3.3.3"}
export MKE_IMAGE_REPO=${MKE_IMAGE_REPO:-"docker.io/mirantis"}
export MSR_VERSION=${MSR_VERSION:-"2.8.3"}
export MSR_IMAGE_REPO=${MSR_IMAGE_REPO:-"docker.io/mirantis"}
export MSR3_VERSION=${MSR_VERSION:-"3.1.1"}
export MSR3_IMAGE_REGISTRY=${MSR_IMAGE_REGISTRY:-"registry.mirantis.com"}
export MSR3_IMAGE_REPO=${MSR_IMAGE_REPO:-"msr"}
export MCR_VERSION=${MCR_VERSION:-"19.03.12"}
export PRESERVE_CLUSTER=${PRESERVE_CLUSTER:-""}
export REUSE_CLUSTER=${REUSE_CLUSTER:-""}
export CLUSTER_NAME=${BUILD_TAG:-"local"}
export DISABLE_TELEMETRY="true"
export ACCEPT_LICENSE="true"

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
  envsubst < "${FOOTLOOSE_TEMPLATE}" > footloose.yaml
  ./footloose delete && docker volume prune -f
}

function createCluster() {
  envsubst < "${FOOTLOOSE_TEMPLATE}" > footloose.yaml
  ./footloose create
}

function createUsers() {
  envsubst < "${FOOTLOOSE_TEMPLATE}" > footloose.yaml
  for h in $(./footloose show -o json|grep '"hostname"'|cut -d'"' -f 4); do
    ./footloose ssh root@${h} "useradd -m launchpad"
    ./footloose ssh root@${h} "mkdir ~launchpad/.ssh && chown launchpad:launchpad ~launchpad/.ssh && chmod 0755 ~launchpad/.ssh"
    ./footloose ssh root@${h} "cat > ~launchpad/.ssh/authorized_keys" < id_rsa_launchpad.pub
    ./footloose ssh root@${h} "chmod 0644 ~launchpad/.ssh/authorized_keys"
    ./footloose ssh root@${h} "echo \"launchpad ALL=(ALL) NOPASSWD: ALL\" > /etc/sudoers"
    ./footloose ssh launchpad@${h} "pwd"
  done
}

function setup() {
  if [ -z "${REUSE_CLUSTER}" ]; then
    generateKey
    downloadFootloose
    deleteCluster
    createCluster
    createUsers
  fi
  export MKE_MANAGER_IP=$(docker inspect mke-manager0 --format='{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}')
  if [ ! -z "${CONFIG_TEMPLATE}" ]; then
    export LAUNCHPAD_CONFIG="${CONFIG_TEMPLATE%.tpl}"
    envsubst < ${CONFIG_TEMPLATE} > ${LAUNCHPAD_CONFIG}
  fi
}

function cloneImages() {
  echo "Setting up a temporary footloose machine for pushing images"
  envsubst < footloose-localrepo.yaml.tpl > footloose-localrepo.yaml
  FLOPT="--config footloose-localrepo.yaml"
  ./footloose delete $FLOPT || true
  ./footloose create $FLOPT
  ./footloose ssh $FLOPT root@pusher0 -- \
    'apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y apt-utils && \
     curl -fsSL https://get.docker.com | DEBIAN_FRONTEND=noninteractive bash -s && \
     mkdir -p /etc/docker && \
     echo "{ \"insecure-registries\":[\"172.16.86.100:5000\"] }" > /etc/docker/daemon.json && \
     service docker restart'

  docker rm --force /registry || true
  docker run --name registry -d --network footloose-cluster --expose 5000 --ip 172.16.86.100 registry:latest

  echo "Pulling + pushing MKE images..."

  docker pull $MKE_IMAGE_REPO/ucp:$MKE_VERSION
  for image in $(docker run --rm ${MKE_IMAGE_REPO}/ucp:${MKE_VERSION} images --list | sort | uniq); do
    imagebase=$(basename ${image})
    imagebase=${imagebase%:*}
    fullimage="$MKE_IMAGE_REPO/$imagebase:$MKE_VERSION"
    ./footloose ssh $FLOPT root@pusher0 -- \
      "docker pull ${fullimage} && \
       docker tag ${fullimage} 172.16.86.100:5000/test/$imagebase:$MKE_VERSION && \
       docker push 172.16.86.100:5000/test/$imagebase:$MKE_VERSION"
  done

  echo "Pulling + pushing MSR3 images..."

  $MSR3_REPO=$MSR3_IMAGE_REGISTRY/$MSR3_IMAGE_REPO
  for image in {enzi:1.0.85,rethinkdb/rethinkdb:2.4.3-mirantis-0.1.3}; do
    fullimage="$MSR3_REPO/$image"
    ./footloose ssh $FLOPT root@pusher0 -- \
      "docker pull ${fullimage} && \
      docker tag ${fullimage} 172.16.86.100:5000/test/$image && \
      docker push 172.16.86.100:5000/test/$image"
  done

  for image in {msr-nginx,msr-registry,msr-garant,msr-notary-signer,msr-notary-server,msr-jobrunner}; do
    fullimage="$MSR3_REPO/$image:$MSR3_VERSION"
    ./footloose ssh $FLOPT root@pusher0 -- \
      "docker pull ${fullimage} && \
      docker tag ${fullimage} 172.16.86.100:5000/test/$image:$MSR_VERSION && \
      docker push 172.16.86.100:5000/test/$imagebase:$MSR_VERSION"
  done

  echo "Pulling + pushing MSR2 images..."

  docker pull $MSR_IMAGE_REPO/dtr:$MSR_VERSION
  for image in $(docker run --rm ${MSR_IMAGE_REPO}/dtr:${MSR_VERSION} images | sort | uniq); do
    imagebase=$(basename ${image})
    imagebase=${imagebase%:*}
    fullimage="$MSR_IMAGE_REPO/$imagebase:$MSR_VERSION"
    ./footloose ssh $FLOPT root@pusher0 -- \
      "docker pull ${fullimage} && \
       docker tag ${fullimage} 172.16.86.100:5000/test/$imagebase:$MSR_VERSION && \
       docker push 172.16.86.100:5000/test/$imagebase:$MSR_VERSION"
  done

  ./footloose delete $FLOPT

  export MKE_IMAGE_REPO=172.16.86.100:5000/test
  export MSR_IMAGE_REPO=172.16.86.100:5000/test
  export MSR3_IMAGE_REGISTRY="172.16.86.100:5000"
  export MSR3_IMAGE_REPO="test"
}

