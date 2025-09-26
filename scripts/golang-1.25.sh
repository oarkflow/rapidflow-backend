#!/bin/bash
set -e

echo "Installing Go 1.25..."

# Detect package manager
if command -v apk >/dev/null 2>&1; then
    # Alpine
    apk update
    apk add --no-cache go
elif command -v apt-get >/dev/null 2>&1; then
    # Ubuntu/Debian
    while fuser /var/lib/apt/lists/lock >/dev/null 2>&1; do sleep 1; done
    apt-get update
    apt-get install -y wget
    wget -q https://go.dev/dl/go1.25.0.linux-amd64.tar.gz
    tar -C /usr/local -xzf go1.25.0.linux-amd64.tar.gz
    export PATH=$PATH:/usr/local/go/bin
    echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
else
    echo "Unsupported package manager"
    exit 1
fi

# Verify installation
go version

echo "Go installation completed"
