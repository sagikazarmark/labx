#!/usr/bin/env bash
set -euo pipefail

if [[ $(id -u) == 0 ]]; then
   echo "This script must NOT be run as root"
   exit 1
fi

# @block:init
dagger init --sdk go ~/my-module
cd ~/my-module
# @endblock

# @block:install_go
dagger install github.com/sagikazarmark/daggerverse/go
# @endblock

# @block:install_favorite
dagger install --name my-favorite github.com/sagikazarmark/daggerverse/golangci-lint
# @endblock

# @block:install_helm
dagger install github.com/sagikazarmark/daggerverse/helm@v0.13.0
# @endblock

# @block:update_helm
dagger update helm
# @endblock

sleep 3

# @block:unpin_helm
jq '.dependencies |= map(if .name == "helm" and has("source") then .source |= sub("@.*$"; "") else . end)' ~/my-module/dagger.json | sponge ~/my-module/dagger.json
# @endblock

dagger update helm

# @block:install_helm_docs
dagger install github.com/sagikazarmark/daggerverse/helm-docs
# @endblock

sleep 2

# @block:uninstall_helm_docs
dagger uninstall helm-docs
# @endblock
