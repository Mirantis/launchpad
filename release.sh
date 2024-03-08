#!/bin/bash
set -e

if [ -z "${TAG_NAME}" ]; then
  echo "TAG_NAME not set"
  exit 1
fi

artifact_path="dist/release"
artifacts=$(find ${artifact_path}/* -exec basename {} \;)
echo "Releasing with:"
for artifact in ${artifacts}; do echo "- ${artifact}"; done

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

for artifact in ${artifacts}
do
   ./github-release upload \
    --user Mirantis \
    --repo mcc \
    --tag "${TAG_NAME}" \
    --name "${artifact}" \
    --file "${artifact_path}/${artifact}"
done

if [ -z "$releaseopt" ]; then
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

  for artifact in ${artifacts}
  do
    ./github-release upload \
      --user Mirantis \
      --repo launchpad \
      --tag "${TAG_NAME}" \
      --name "${artifact}" \
      --file "${artifact_path}/${artifact}"
  done
fi

rm ./github-release
