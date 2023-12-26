#!/bin/bash -eu

# Install additional packages:
#
# - gnupg
#     git commit signing
#
# - chromium-driver
#     Required for running Chrome in integration tests. We don't technically
#     require the full package since we pull it in hermetically with Bazel,
#     but this will ensure that any dependencies are installed.
#
# - python3
#     Used by scripts to manage Chrome extension manifest file.
export DEBIAN_FRONTEND=noninteractive
sudo apt-get update
sudo apt-get install -y --no-install-recommends \
    gnupg \
    chromium-driver \
    python3