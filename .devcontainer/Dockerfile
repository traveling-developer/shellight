FROM mcr.microsoft.com/devcontainers/base:trixie

ARG GO_VERSION=1.27.3
ARG NODE_VERSION=24.22.1
# Wails v3.0.0 pins exactly this Zig version for darwin cross builds
ARG ZIG_VERSION=0.14.0
ARG RCODESIGN_VERSION=0.29.0
ARG WAILS_VERSION=v3.0.0
# Local darwin builds only; releases are built on a GitHub macOS runner (S7)
ARG MACOS_SDK_URL=https://github.com/joseluisq/macosx-sdks/releases/download/14.5/MacOSX14.5.sdk.tar.xz
ARG MACOS_SDK_SHA256=6e146275d19f027faa2e8354da5e0267513abf013b8f16ad65a231653a2b1c5d

# cgo toolchain plus GTK4/WebKitGTK 6.0, which the wails3 CLI links against
RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential pkg-config curl ca-certificates git xz-utils \
    libgtk-4-dev libwebkitgtk-6.0-dev \
    && rm -rf /var/lib/apt/lists/*

RUN set -eux; \
    case "$(dpkg --print-architecture)" in \
    arm64) go=arm64; node=arm64; musl=aarch64;; \
    amd64) go=amd64; node=x64;   musl=x86_64;; \
    esac; \
    curl -fsSL "https://go.dev/dl/go${GO_VERSION}.linux-${go}.tar.gz" | tar -C /usr/local -xz; \
    curl -fsSL "https://nodejs.org/dist/v${NODE_VERSION}/node-v${NODE_VERSION}-linux-${node}.tar.xz" \
    | tar -C /usr/local --strip-components=1 -xJ; \
    mkdir -p /opt/zig; \
    curl -fsSL "https://ziglang.org/download/${ZIG_VERSION}/zig-linux-${musl}-${ZIG_VERSION}.tar.xz" \
    | tar -C /opt/zig --strip-components=1 -xJ; \
    ln -s /opt/zig/zig /usr/local/bin/zig; \
    curl -fsSL "https://github.com/indygreg/apple-platform-rs/releases/download/apple-codesign%2F${RCODESIGN_VERSION}/apple-codesign-${RCODESIGN_VERSION}-${musl}-unknown-linux-musl.tar.gz" \
    | tar -xzO --wildcards '*/rcodesign' > /usr/local/bin/rcodesign; \
    chmod 0755 /usr/local/bin/rcodesign

RUN set -eux; \
    curl -fsSL -o /tmp/sdk.tar.xz "${MACOS_SDK_URL}"; \
    echo "${MACOS_SDK_SHA256}  /tmp/sdk.tar.xz" | sha256sum -c -; \
    mkdir -p /opt/macos-sdk; \
    tar -C /opt/macos-sdk --strip-components=1 -xJf /tmp/sdk.tar.xz; \
    rm /tmp/sdk.tar.xz

USER vscode
# HOME explicit: some builders keep HOME=/root after USER, which breaks go install
ENV HOME=/home/vscode \
    GOTOOLCHAIN=local \
    PATH=/usr/local/go/bin:/home/vscode/go/bin:/home/vscode/.local/bin:${PATH}

RUN go install github.com/wailsapp/wails/v3/cmd/wails3@${WAILS_VERSION}
RUN curl -fsSL https://claude.ai/install.sh | bash

WORKDIR /workspaces/shellight
