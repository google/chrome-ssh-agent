#!/bin/bash -eu

# Ensure that brew environment is loaded.
eval $(/home/linuxbrew/.linuxbrew/bin/brew shellenv)

# Install NPM dependencies to node_modules. Enables type completion.
# Remove extraneous packages.
pnpm install
pnpm prune

# Initiate build. Enables type completion for generated files.
# Don't bother cleaning, as Bazel provides hermetic/repeatable builds.
bazel build --keep_going ...