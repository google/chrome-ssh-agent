#!/bin/bash -eu

bazel run @go_sdk//:bin/go -- get -u ./...
bazel run @go_sdk//:bin/go -- mod tidy
bazel run //:gazelle-update-repos