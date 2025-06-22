#!/usr/bin/env bash

# Directory ownership - automatically extract from devcontainer.json
DIRS="/home/vscode/.aws /home/vscode/.kube /usr/local/go /go /tmp/gocache /home/vscode/.cache/JetBrains"

for dir in $DIRS; do
  if [ -d "$dir" ]; then
    sudo chown -R vscode:vscode "$dir"
  fi
done