#!/bin/bash -eux

cd $(dirname $0)/..

# NPM dependencies
pnpm install

# Incorporate new dependencies in go.mod
bazel run @go_sdk//:bin/go -- mod tidy
bazel run //:gazelle-update-repos

# Add any new dependencies in sources to BUILD files
bazel run //:gazelle