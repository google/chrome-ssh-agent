#!/bin/bash -eu
#
# Copyright 2018 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Usage:
#  (1) Update version in manifest.json
#  (2) Run tag-release.sh

cd $(dirname $0)/..

function die() {
  echo $1
  exit 1
}

# Read current version from manifest.
readonly MANIFEST=${PWD}/manifest.json
readonly MANIFEST_BETA=${PWD}/manifest-beta.json
readonly VERSION=$(cat "${MANIFEST}" | python3 -c "import sys, json; print(json.load(sys.stdin)['version'])")
readonly VERSION_BETA=$(cat "${MANIFEST_BETA}" | python3 -c "import sys, json; print(json.load(sys.stdin)['version'])")
readonly TAG=v${VERSION}

# Ensure both manifests have the same version. This could happen if only one of
# the manifests was updated.
test "${VERSION}" = "${VERSION_BETA}" \
  || die "Prod and Beta versions do not match; Prod is ${VERSION}, Beta is ${VERSION_BETA}"

# Ensure the tag doesn't already exist.  This could happen if someone forgot to
# update the version in manifest.json.
test -z $(git tag | grep --line-regexp "${TAG}") \
  || die "Version ${VERSION} already exists"

# Ensure all tests pass.
bazel test ...

# Commit everything. This should include the change to manifest.json.
git add .
git commit -m "Bump to version ${VERSION}"

# Tag the release.
git tag -a "${TAG}" -m "Tag version ${VERSION}"

# Push to remote repository, both changes and tags.
git push
git push --tags
