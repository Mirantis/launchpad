#!/bin/bash
export LINUX_IMAGE=${LINUX_IMAGE:-"quay.io/footloose/ubuntu18.04"}
echo "Setting up a temporary footloose machine for pushing images"
envsubst < footloose-localrepo.yaml.tpl > footloose-localrepo.yaml
FLOPT="--config footloose-localrepo.yaml"
footloose create $FLOPT
footloose ssh $FLOPT root@pusher0 -- \
  'apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y apt-utils && \
   curl -fsSL https://get.docker.com | DEBIAN_FRONTEND=noninteractive bash -s && \
   mkdir -p /etc/docker && \
   echo "{ \"insecure-registries\":[\"172.16.86.100:5000\"] }" > /etc/docker/daemon.json && \
   service docker restart'

