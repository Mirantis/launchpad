#!/bin/bash

set -e

if [ -z "${TAG_NAME}" ]; then
  echo "TAG_NAME not set"
  exit 1
fi

declare -a binaries=("mcc-darwin-x64" "mcc-win-x64.exe" "mcc-linux-x64")

description="### Checksums\n\nFilename | Sha256\n---------|-------\n"

for bin in "${binaries[@]}"
do
  filesum=$(sha256sum -b "./bin/${bin}" | cut -d" " -f1)
  description=${description}"${bin} | ${filesum}\n"
  echo "${filesum} *${bin}" > "./bin/${bin}.sha256"
done

echo -e "${description}"

curl -L https://github.com/github-release/github-release/releases/download/v0.8.1/linux-amd64-github-release.bz2 > ./github-release.bz2
bunzip2 ./github-release.bz2
chmod +x ./github-release

if [[ "${TAG_NAME}" == *-* ]] ; then
  releaseopt="--pre-release"
else
  releaseopt="--draft"
fi

echo -e "${description}" | ./github-release release \
  $releaseopt \
  --user Mirantis \
  --repo mcc \
  --tag "${TAG_NAME}" \
  --name "${TAG_NAME}" \
  --description -

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
    --file "./bin/${bin}.sha256"
done

rm ./github-release
