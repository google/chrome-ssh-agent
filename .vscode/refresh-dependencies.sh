#!/bin/bash -eux

cd $(dirname $0)/..

# NPM dependencies
pnpm install

# Incorporate new dependencies in go.mod
bazel run @rules_go//go -- mod tidy

# Add any new dependencies in sources to BUILD files
bazel run //:gazelle