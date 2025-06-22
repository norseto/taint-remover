#!/usr/bin/env bash

export DEBIAN_FRONTEND=noninteractive
sudo apt-get update && sudo apt-get install -y --no-install-recommends xdg-utils

# Directory ownership - automatically extract from devcontainer.json
DIRS="/home/vscode/.aws /home/vscode/.kube /usr/local/go /go /tmp/gocache /home/vscode/.cache/JetBrains"

for dir in $DIRS; do
  if [ -d "$dir" ]; then
    sudo chown -R vscode:vscode "$dir"
  fi
done

ln -s $HOME/.bun/bin/bun $HOME/.bun/bin/node
ln -s $HOME/.bun/bin/bun $HOME/.bun/bin/npm
ln -s $HOME/.bun/bin/bunx $HOME/.bun/bin/npx

bun install -g @openai/codex
