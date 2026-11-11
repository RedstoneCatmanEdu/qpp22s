#!/bin/bash
set -e
cd "$(dirname "$0")/.."

echo "=== Installing Python dependencies ==="
pip install -r requirements.txt

echo "=== Compiling Go binaries into bin/ ==="
mkdir -p bin
cd src

go build -o ../bin/server ./server/main.go ./server/cid.go ./server/tls.go
go build -o ../bin/client ./client/main.go
go build -o ../bin/microclient ./microclient/main.go

echo "=== Setup complete! Binaries are in bin/ ==="