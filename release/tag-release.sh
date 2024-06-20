#!/bin/bash
set -eu

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
#  (1) Update version in manifest.json and merge into master
#  (2) Run tag-release.sh

cd "$(dirname "$0")"/..

function die() {
  echo "$1" >&2
  exit 1
}

# Read current version from manifest.
readonly MANIFEST="${PWD}/manifest.json"
readonly MANIFEST_BETA="${PWD}/manifest-beta.json"
readonly VERSION=$(jq -r '.version' "$MANIFEST")
readonly TAG="v$VERSION"

# Ensure we are currently in the master branch.
test "$(git branch --show-current)" = "master" \
  || die "Must be in master to tag a new release"

# Ensure there are no pending local changes.
test -z "$(git status --porcelain)" \
  || die "Cannot release when there are pending changes."

# Pull must recent changes.  We don't want to release from outdated state.
git pull

# Ensure both manifests have the same version. This could happen if only one of
# the manifests was updated.
readonly VERSION_BETA=$(jq -r '.version' "$MANIFEST_BETA")
test "$VERSION" = "$VERSION_BETA" \
  || die "Prod and Beta versions do not match; Prod is $VERSION, Beta is $VERSION_BETA"

# Ensure the tag doesn't already exist.  This could happen if someone forgot to
# update the version in manifest.json.
git tag | grep -qFx "$TAG" \
  && die "Version $VERSION already exists"

# Ensure all tests pass.
bazel test ...

# Tag the release.
git tag -a "$TAG" -m "Tag version $VERSION"

# Push to remote repository, both changes and tags.
git push
git push --tags