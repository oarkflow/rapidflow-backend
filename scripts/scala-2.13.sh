#!/bin/bash
set -e

echo "Installing Scala 2.13 and SBT..."

# Detect package manager
if command -v apk >/dev/null 2>&1; then
    # Alpine
    apk update
    apk add --no-cache openjdk11 curl bash
    curl -L -o /usr/local/bin/sbt https://github.com/sbt/sbt/releases/download/v1.9.7/sbt-1.9.7.tgz | tar -xz -C /usr/local --strip-components=1
    chmod +x /usr/local/bin/sbt
elif command -v apt-get >/dev/null 2>&1; then
    # Ubuntu/Debian
    apt-get update
    apt-get install -y openjdk-11-jdk curl
    echo "deb https://repo.scala-sbt.org/scalasbt/debian all main" | tee /etc/apt/sources.list.d/sbt.list
    echo "deb https://repo.scala-sbt.org/scalasbt/debian /" | tee /etc/apt/sources.list.d/sbt_old.list
    curl -sL "https://keyserver.ubuntu.com/pks/lookup?op=get&search=0x2EE0EA64E40A89B84B2DF73499E82A75642AC823" | apt-key add
    apt-get update
    apt-get install -y sbt
else
    echo "Unsupported package manager"
    exit 1
fi

# Verify installation
java -version
sbt --version

echo "Scala and SBT installation completed"
