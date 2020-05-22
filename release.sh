#!/bin/bash

set -e

if [ -z "${TAG_NAME}" ]; then
  echo "TAG_NAME not set"
  exit 1
fi

curl -L https://github.com/github-release/github-release/releases/download/v0.8.1/linux-amd64-github-release.bz2 > ./github-release.bz2
bunzip2 ./github-release.bz2
chmod +x ./github-release

if echo "${TAG_NAME}" | grep "-" ; then
  ./github-release release \
    --pre-release \
    --user Mirantis \
    --repo mcc \
    --tag "${TAG_NAME}" \
    --name "${TAG_NAME}"
else
  ./github-release release \
    --draft \
    --user Mirantis \
    --repo mcc \
    --tag "${TAG_NAME}" \
    --name "${TAG_NAME}"
fi

declare -a binaries=("launchpad-darwin-x64" "launchpad-win-x64.exe" "launchpad-linux-x64")

for bin in "${binaries[@]}"
do
   ./github-release upload \
    --user Mirantis \
    --repo mcc \
    --tag "${TAG_NAME}" \
    --name "${bin}" \
    --file "./bin/${bin}"
done

rm ./github-release
