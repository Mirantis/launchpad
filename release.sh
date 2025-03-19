#!/bin/bash
set -e

REPO_NAME="${REPO_NAME:-launchpad}"
REPO_USER="${REPO_USER:-Mirantis}"

if [ -z "${TAG_NAME}" ]; then
  echo "TAG_NAME not set"
  exit 1
fi

artifact_path="dist/release"
artifacts=$(find ${artifact_path}/* -exec basename {} \;)
echo "Releasing with:"
for artifact in ${artifacts}; do echo "- ${artifact}"; done

curl -L https://github.com/github-release/github-release/releases/download/v0.10.0/linux-amd64-github-release.bz2 > ./github-release.bz2
bunzip2 -f ./github-release.bz2
chmod +x ./github-release

if [[ "${TAG_NAME}" == *-* ]] ; then
  releaseopt="--pre-release"
else
  releaseopt=""
fi

echo "Creating release named ${TAG_NAME} in ${REPO_NAME} repo"

./github-release release \
  $releaseopt \
  --user "${REPO_USER}" \
  --repo "${REPO_NAME}" \
  --tag "${TAG_NAME}" \
  --name "${TAG_NAME}"

sleep 10

echo "Uploading the artifacts to ${TAG_NAME} in ${REPO_NAME} repo"

for artifact in ${artifacts}
do
   ./github-release upload \
    --user "${REPO_USER}" \
    --repo "${REPO_NAME}" \
    --tag "${TAG_NAME}" \
    --name "${artifact}" \
    --file "${artifact_path}/${artifact}"
done

rm ./github-release
