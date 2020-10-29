#!/bin/bash

set -e

cd test
. ./smoke.common.sh

function registrycleanup() {
  docker kill /registry || true
  docker rm --force /registry || true
  ./footloose delete --config footloose-localrepo.yaml || true
  cleanup
}
trap registrycleanup EXIT

setup && downloadTools && cloneImages

../bin/launchpad reset
