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
# OpenShell CLI for the nested test gateway; must match the openshell on the Mac
ARG OPENSHELL_VERSION=0.1.7
ARG OPENSHELL_SHA256_ARM64=696b523c263895b69eaa7c7e6afbb19b8af6c65cd79c7ffac7b685b6e3aa23da
ARG OPENSHELL_SHA256_AMD64=5c1adfcae797eac9b2e8b9280c66c1f75e4e84d9e31ce6938c5b994314ef6098

# cgo toolchain plus GTK4/WebKitGTK 6.0, which the wails3 CLI links against.
# podman and friends: nested rootless Podman for the OpenShell test gateway.
# It needs the runArgs in devcontainer.json (label=disable, unmask=ALL,
# /dev/net/tun) and a named volume for container storage (overlay on overlay
# does not work). jq is used by crun-no-hostname.
RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential pkg-config curl ca-certificates git xz-utils jq \
    libgtk-4-dev libwebkitgtk-6.0-dev \
    podman passt uidmap fuse-overlayfs catatonit \
    && rm -rf /var/lib/apt/lists/*

COPY containers.conf /etc/containers/containers.conf.d/50-shellight.conf
COPY --chmod=0755 crun-no-hostname /usr/local/bin/crun-no-hostname
# newuidmap/newgidmap with file caps instead of setuid root (as in
# quay.io/podman/stable). Writing a uid_map needs CAP_SYS_ADMIN in the target
# user namespace; a setuid-root helper has euid 0 instead of the namespace
# owner's uid and the devcontainer has no CAP_SYS_ADMIN, so the write fails
# with EPERM. With file caps the helper keeps the caller's euid.
RUN chmod 0755 /usr/bin/newuidmap /usr/bin/newgidmap \
    && setcap cap_setuid+ep /usr/bin/newuidmap \
    && setcap cap_setgid+ep /usr/bin/newgidmap

# OpenShell CLI, static musl build, checksum from the release
RUN set -eux; case "$(dpkg --print-architecture)" in \
    arm64) a=aarch64; sha="${OPENSHELL_SHA256_ARM64}";; \
    amd64) a=x86_64;  sha="${OPENSHELL_SHA256_AMD64}";; \
    *) echo "unsupported architecture: $(dpkg --print-architecture)" >&2; exit 1;; \
    esac; \
    curl -fsSL --retry 5 --retry-delay 3 --connect-timeout 15 \
    -o /tmp/openshell.tgz \
    "https://github.com/NVIDIA/OpenShell/releases/download/v${OPENSHELL_VERSION}/openshell-${a}-unknown-linux-musl.tar.gz"; \
    echo "${sha}  /tmp/openshell.tgz" | sha256sum -c -; \
    tar -xzf /tmp/openshell.tgz -C /usr/local/bin openshell; \
    rm /tmp/openshell.tgz

RUN set -eux; \
    case "$(dpkg --print-architecture)" in \
    arm64) go=arm64; node=arm64; musl=aarch64;; \
    amd64) go=amd64; node=x64;   musl=x86_64;; \
    *) echo "unsupported architecture: $(dpkg --print-architecture)" >&2; exit 1;; \
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

# Mount point for the named volume with nested Podman storage; created as
# vscode so the empty volume inherits the ownership on first mount.
RUN mkdir -p /home/vscode/.local/share/containers

RUN go install github.com/wailsapp/wails/v3/cmd/wails3@${WAILS_VERSION}
RUN curl -fsSL https://claude.ai/install.sh | bash

WORKDIR /workspaces/shellight
