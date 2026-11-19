#!/usr/bin/env bash
#
# Decide what kind of GitHub Release a pushed tag should produce.
#
# The policy, which .github/workflows/release.yml implements and
# docs/development/workflow.md states for humans:
#
#   vX.Y.Z                     -> draft, full release
#   vX.Y.Z-rcN / -tpN          -> draft, prerelease
#   vX.Y.Z-alphaN / -betaN     -> draft, prerelease
#   anything else              -> no release at all
#
# Every release is created as a draft. A human reviews the assembled release
# notes and the attached binaries and then publishes. That review step has been
# happening since the move to GitHub Actions; it was simply never written down,
# and neither was the prerelease flag, which a human had to remember to tick in
# the GitHub UI. Setting it from the tag shape is the point of this script: the
# flag is a property of the tag, not of whoever happens to publish.
#
# Tags that are neither a version nor a recognised prerelease -- `v1.5.19-PRODENG-3594`,
# `v1.5.12-557-PRODENG-2780`, `v1.5.5-jnesbitt` -- are scratch builds. They get
# no release object. The build job still runs and still uploads the binaries as
# a workflow artifact, which is what a scratch tag is pushed for. Creating a
# release for them only produces something a maintainer has to delete by hand,
# which is what happened to v1.5.19-PRODENG-3594.
#
# An unrecognised shape is reported loudly rather than guessed at. Guessing
# "full release" would publish an unintended GA; guessing "prerelease" would
# leave scratch drafts accumulating. Neither is better than saying so.
#
# Usage:
#   TAG_NAME=v1.5.20-rc1 ./release-state.sh          # writes $GITHUB_OUTPUT if set
#   ./release-state.sh v1.5.20-rc1                   # argument form, for dry runs
#
# The tag name is read from the environment or argv, never spliced into any
# script text: a tag name is chosen by whoever pushes the tag, and git permits
# `$`, backticks, `;`, `&`, `|` and parentheses in a ref.
#
set -euo pipefail

TAG_NAME="${1:-${TAG_NAME:-}}"

if [ -z "$TAG_NAME" ]; then
    echo "release-state.sh: no tag name given (set TAG_NAME or pass it as an argument)" >&2
    exit 1
fi

# Version core, then an optional prerelease suffix.
#
# The suffix alternatives are matched whole: `rc1` qualifies, `rc` and
# `release-candidate` do not. The numeral is optional because v1.5.5-alpha
# exists, and the separator before it is optional because both v1.5.5-alpha-1
# and v1.5.6-alpha2 exist. No such laxity applies to the version core: a tag
# that is not vMAJOR.MINOR.PATCH is not a release of this product.
VERSION_CORE='v[0-9]+\.[0-9]+\.[0-9]+'
PRERELEASE_SUFFIX='(rc|tp|alpha|beta)-?[0-9]*'

create_release=true
draft=true
prerelease=false
reason=

if [[ "$TAG_NAME" =~ ^${VERSION_CORE}$ ]]; then
    prerelease=false
    reason="release version tag"
elif [[ "$TAG_NAME" =~ ^${VERSION_CORE}-${PRERELEASE_SUFFIX}$ ]]; then
    prerelease=true
    reason="prerelease tag"
else
    create_release=false
    draft=false
    prerelease=false
    reason="not a release tag shape; binaries are uploaded as a workflow artifact only"
fi

echo "tag:             $TAG_NAME"
echo "create_release:  $create_release"
echo "draft:           $draft"
echo "prerelease:      $prerelease"
echo "reason:          $reason"

if [ -n "${GITHUB_OUTPUT:-}" ]; then
    {
        echo "create_release=$create_release"
        echo "draft=$draft"
        echo "prerelease=$prerelease"
    } >> "$GITHUB_OUTPUT"
fi

if [ -n "${GITHUB_STEP_SUMMARY:-}" ]; then
    # The decision belongs in the run summary, not only in the step log. A
    # maintainer who pushed a tag and got no release needs to see why without
    # expanding a step.
    {
        echo "### Release state for \`$TAG_NAME\`"
        echo
        echo "| create release | draft | prerelease |"
        echo "| --- | --- | --- |"
        echo "| $create_release | $draft | $prerelease |"
        echo
        echo "$reason"
    } >> "$GITHUB_STEP_SUMMARY"
fi
