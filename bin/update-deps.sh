#!/bin/bash -eu

cd $(dirname $0)/..

go mod tidy

bazel run //:gazelle -- update-repos --prune=true --from_file=go.mod --to_macro=go_deps.bzl%go_dependencies
