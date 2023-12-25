FROM mcr.microsoft.com/devcontainers/base:debian

# Install basic software that enables other software to be installed
RUN apt-get update && export DEBIAN_FRONTEND=noninteractive \
    && apt-get install -y --no-install-recommends \
        curl apt-transport-https ca-certificates \
        gnupg \
        pkg-config \
        zip unzip \
        git

# Install packages required for running Chrome in integration tests.
# We don't technically require the full package since we pull it in hermetically
# with Bazel, but this will ensure that any dependencies are installed.
RUN export DEBIAN_FRONTEND=noninteractive \
    && apt-get install -y --no-install-recommends \
        chromium-driver

# Clean extraneous packages and files
RUN apt-get autoremove -y \
    && apt-get clean -y \
    && rm -rf /var/lib/apt/lists/*

