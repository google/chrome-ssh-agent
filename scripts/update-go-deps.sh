#!/bin/bash -eu

bazel run @rules_go//go -- get -u ./...
bazel run @rules_go//go -- mod tidy