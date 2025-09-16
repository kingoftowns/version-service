FROM mcr.microsoft.com/devcontainers/go:1-1.24-bullseye

RUN apt-get update && apt-get install -y \
    redis-tools \
    make \
    && rm -rf /var/lib/apt/lists/*

RUN cd /tmp && \
    go mod init temp && \
    go install github.com/air-verse/air@v1.60.0 && \
    go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest && \
    go install github.com/go-delve/delve/cmd/dlv@latest && \
    go install github.com/swaggo/swag/cmd/swag@v1.16.3 && \
    rm -rf /tmp/go.mod /tmp/go.sum

RUN usermod -aG sudo vscode && \
    echo "vscode ALL=(ALL) NOPASSWD:ALL" >> /etc/sudoers

WORKDIR /workspace

USER vscode