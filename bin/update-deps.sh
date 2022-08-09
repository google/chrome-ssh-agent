#!/bin/bash -eu

cd $(dirname $0)/..

go mod tidy

bazel run //:gazelle-update-repos
