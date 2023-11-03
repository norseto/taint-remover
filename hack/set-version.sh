#!/bin/bash -xe

if [ $# -lt 1 ]; then
  echo "Usage: $0 NEW_VERSION" 1>&2
  exit 1
fi

VERSION=$(grep 'RELEASE_VERSION\s*=' version.go  | awk -F= '{print $2}' | sed -e 's_"__g' -e 's/\s//g')

NEW_VERSION=$1
if [[ ! "${NEW_VERSION}" =~ ^([0-9]+[.][0-9]+)[.]([0-9]+)(-(alpha|beta)[.]([0-9]+))?$ ]]; then
  echo "New version ${VERSION} must be 'X.Y.Z', 'X.Y.Z-alpha.N', or 'X.Y.Z-beta.N'"
  exit 1
fi

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
BASE_DIR=${SCRIPT_DIR}/..

sed -i.bak -e "s@newTag:\s*v${VERSION}@newTag: v${NEW_VERSION}@g" ${BASE_DIR}/config/manager/kustomization.yaml
sed -i.bak -e "s@RELEASE_VERSION\s*=\s*\"${VERSION}\"@RELEASE_VERSION = \"${NEW_VERSION}\"@g" ${BASE_DIR}/version.go

