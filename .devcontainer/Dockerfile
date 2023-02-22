FROM mcr.microsoft.com/devcontainers/base:bullseye

# Install basic software that enables other software to be installed
RUN apt-get update && export DEBIAN_FRONTEND=noninteractive \
    && apt-get install -y --no-install-recommends \
        curl apt-transport-https ca-certificates \
        gnupg \
        pkg-config \
        zip unzip \
        python \
        git \
        libnss3-dev

# Install Bazel
ARG BAZELISK_VERSION=v1.10.1
ARG BAZELISK_DOWNLOAD_SHA=dev-mode
RUN curl -fSsL -o /usr/local/bin/bazelisk https://github.com/bazelbuild/bazelisk/releases/download/${BAZELISK_VERSION}/bazelisk-linux-amd64 \
    && ([ "${BAZELISK_DOWNLOAD_SHA}" = "dev-mode" ] || echo "${BAZELISK_DOWNLOAD_SHA} */usr/local/bin/bazelisk" | sha256sum --check - ) \
    && chmod 0755 /usr/local/bin/bazelisk \
    && ln -sf /usr/local/bin/bazelisk /usr/local/bin/bazel

# Install Homebrew. See:
#   https://docs.brew.sh/Installation#skip-tap-cloning-beta
#   https://docs.brew.sh/Installation#unattended-installation
ENV HOMEBREW_INSTALL_FROM_API=1
ENV NONINTERACTIVE=1
RUN /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)" \
    && echo 'eval "$(/home/linuxbrew/.linuxbrew/bin/brew shellenv)"' >> /etc/profile.d/99-homebrew.sh \
    && chmod 0755 /etc/profile.d/99-homebrew.sh \
    && eval "$(/home/linuxbrew/.linuxbrew/bin/brew shellenv)" \
    && brew --version

# Install pnpm package manager
RUN eval "$(/home/linuxbrew/.linuxbrew/bin/brew shellenv)" \
    && brew install pnpm

# Clean extraneous packages and files
RUN apt-get autoremove -y \
    && apt-get clean -y \
    && rm -rf /var/lib/apt/lists/*

