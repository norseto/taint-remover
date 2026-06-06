#!/bin/bash -e

VERSION=$(grep 'RELEASE_VERSION\s*=' version.go  | awk -F= '{print $2}' | sed -e 's_"__g' -e 's/[[:space:]]//g')

if [[ ! "${VERSION}" =~ ^([0-9]+[.][0-9]+)[.]([0-9]+)(-(alpha|beta)[.]([0-9]+))?$ ]]; then
  echo "Version ${VERSION} must be 'X.Y.Z', 'X.Y.Z-alpha.N', or 'X.Y.Z-beta.N'" >&2
  exit 1
fi

MINOR=${BASH_REMATCH[1]}
RELEASE_BRANCH="release-${MINOR}"
TAG="v${VERSION}"

MODE=${1:-}

case "${MODE}" in
preflight)
  REMOTE_TAG=$(git ls-remote --tags origin "refs/tags/${TAG}")
  if [ -n "${REMOTE_TAG}" ]; then
    echo "Tag ${TAG} already exists" >&2
    exit 1
  fi

  echo "${VERSION}"
  ;;
finalize)
  git tag -a -m "Release ${VERSION}" "${TAG}"
  git push origin "${TAG}"

  if [[ ! "${VERSION}" =~ .0-beta.1$ ]]; then
    exit 0
  fi

  git branch "${RELEASE_BRANCH}"
  git push origin "${RELEASE_BRANCH}"
  ;;
*)
  echo "Usage: $0 preflight|finalize" >&2
  exit 1
  ;;
esac
