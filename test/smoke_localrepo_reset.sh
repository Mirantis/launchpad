#!/bin/bash

set -e

cd test
. ./smoke.common.sh

function registrycleanup() {
  docker kill /registry
  docker rm --force /registry
  cleanup
}
trap registrycleanup EXIT

setup && downloadTools

echo "Setting up a temporary footloose machine for pushing images"
envsubst < footloose-localrepo.yaml.tpl > footloose-localrepo.yaml
FLOPT="--config footloose-localrepo.yaml"
./footloose create $FLOPT
./footloose ssh $FLOPT root@pusher0 -- \
  'apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y apt-utils && \
   curl -fsSL https://get.docker.com | DEBIAN_FRONTEND=noninteractive bash -s && \
   mkdir -p /etc/docker && \
   echo "{ \"insecure-registries\":[\"172.16.86.100:5000\"] }" > /etc/docker/daemon.json && \
   service docker restart'

docker run --name registry -d --network footloose-cluster --expose 5000 --ip 172.16.86.100 registry:latest

echo "Pulling + pushing UCP images..."

docker pull $UCP_IMAGE_REPO/ucp:$UCP_VERSION
for image in $(docker run --rm ${UCP_IMAGE_REPO}/ucp:${UCP_VERSION} images --list | sort | uniq); do
  imagebase=$(basename ${image})
  imagebase=${imagebase%:*}
  fullimage="$UCP_IMAGE_REPO/$imagebase:$UCP_VERSION"
  ./footloose ssh $FLOPT root@pusher0 -- \
    "docker pull ${fullimage} && \
     docker tag ${fullimage} 172.16.86.100:5000/test/$imagebase:$UCP_VERSION && \
     docker push 172.16.86.100:5000/test/$imagebase:$UCP_VERSION"
done

echo "Pulling + pushing DTR images..."

docker pull $DTR_IMAGE_REPO/dtr:$DTR_VERSION
for image in $(docker run --rm ${DTR_IMAGE_REPO}/dtr:${DTR_VERSION} images | sort | uniq); do
  imagebase=$(basename ${image})
  imagebase=${imagebase%:*}
  fullimage="$DTR_IMAGE_REPO/$imagebase:$DTR_VERSION"
  ./footloose ssh $FLOPT root@pusher0 -- \
    "docker pull ${fullimage} && \
     docker tag ${fullimage} 172.16.86.100:5000/test/$imagebase:$DTR_VERSION && \
     docker push 172.16.86.100:5000/test/$imagebase:$DTR_VERSION"
done


export UCP_IMAGE_REPO=172.16.86.100:5000/test
export DTR_IMAGE_REPO=172.16.86.100:5000/test

${LAUNCHPAD} reset --config ${LAUNCHPAD_CONFIG} --force
