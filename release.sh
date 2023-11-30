#!/bin/bash

set -e

if [ -z "${TAG_NAME}" ]; then
  echo "TAG_NAME not set"
  exit 1
fi


declare -a binaries=("launchpad-darwin-x64" "launchpad-darwin-arm64" "launchpad-win-x64.exe" "launchpad-linux-x64" "launchpad-linux-arm64")

mkdir -p tmp.sha256
pushd bin

for bin in "${binaries[@]}"
do
  sha256sum -b "${bin}" > "../tmp.sha256/${bin}.sha256"
done

popd

curl -L https://github.com/github-release/github-release/releases/download/v0.8.1/linux-amd64-github-release.bz2 > ./github-release.bz2
bunzip2 -f ./github-release.bz2
chmod +x ./github-release

if [[ "${TAG_NAME}" == *-* ]] ; then
  releaseopt="--pre-release"
else
  releaseopt=""
fi

echo "Creating release named ${TAG_NAME} in MCC repo"

./github-release release \
  $releaseopt \
  --user Mirantis \
  --repo mcc \
  --tag "${TAG_NAME}" \
  --name "${TAG_NAME}"

sleep 10

echo "Uploading the artifacts to ${TAG_NAME} in MCC repo"

for bin in "${binaries[@]}"
do
   ./github-release upload \
    --user Mirantis \
    --repo mcc \
    --tag "${TAG_NAME}" \
    --name "${bin}" \
    --file "./bin/${bin}"

   ./github-release upload \
    --user Mirantis \
    --repo mcc \
    --tag "${TAG_NAME}" \
    --name "${bin}.sha256" \
    --file "./tmp.sha256/${bin}.sha256"
done

echo "Creating release named ${TAG_NAME} in Launchpad repo"

# Release to the public repo
./github-release release \
  $releaseopt \
  --draft \
  --user Mirantis \
  --repo launchpad \
  --tag "${TAG_NAME}" \
  --name "${TAG_NAME}"

sleep 10

echo "Uploading the artifacts to ${TAG_NAME} in Launchpad repo"

for bin in "${binaries[@]}"
do
  ./github-release upload \
    --user Mirantis \
    --repo launchpad \
    --tag "${TAG_NAME}" \
    --name "${bin}" \
    --file "./bin/${bin}"

  ./github-release upload \
    --user Mirantis \
    --repo launchpad \
    --tag "${TAG_NAME}" \
    --name "${bin}.sha256" \
    --file "./tmp.sha256/${bin}.sha256"
  done

rm ./github-release
rm -rf tmp.sha256
