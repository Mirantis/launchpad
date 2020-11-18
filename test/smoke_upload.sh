#!/bin/bash

set -e

cd test
. ./smoke.common.sh
trap cleanup EXIT

setup && downloadTools

mkdir -p images
docker pull docker.io/mirantis/ucp:3.3.3
for image in $(docker run --rm ${MKE_IMAGE_REPO}/ucp:${MKE_VERSION} images --list | sort | uniq); do
  docker pull docker.io/$image
  fn=${image//\//-}
  fn=${fn//:/}
  docker save $image | gzip > images/$fn.tar.gz
done

../bin/launchpad --debug apply

rm -rf images

../bin/launchpad describe hosts
../bin/launchpad describe mke
