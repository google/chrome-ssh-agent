FROM mcr.microsoft.com/devcontainers/base:bullseye

# Install basic software that enables other software to be installed
RUN apt-get update && export DEBIAN_FRONTEND=noninteractive \
    && apt-get install -y --no-install-recommends \
        curl apt-transport-https ca-certificates \
        gnupg \
        pkg-config \
        zip unzip \
        python \
        git

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

# Install Bazel
RUN eval "$(/home/linuxbrew/.linuxbrew/bin/brew shellenv)" \
    && brew install bazelisk

# Install pnpm package manager
RUN eval "$(/home/linuxbrew/.linuxbrew/bin/brew shellenv)" \
    && brew install pnpm

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

