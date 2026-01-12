#!/usr/bin/env bash

export DEBIAN_FRONTEND=noninteractive
sudo apt-get update && sudo apt-get install -y --no-install-recommends xdg-utils

# Directory ownership - automatically extract from devcontainer.json
sudo chown -R $(id -u):$(id -g)  \
  /home/vscode/.aws /home/vscode/.kube \
  /usr/local/go \
  /home/vscode/.cache \
  /tmp/.gocache /tmp/.gomodcache /go

sudo chown -R $(id -u):$(id -g) $HOME/.codex $HOME/.claude
