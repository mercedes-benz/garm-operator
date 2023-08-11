#!/bin/bash

# make sure you have github-cli installed and are authenticate with github-cli
OWNER="GitHub-Actions"
REPO="garm-server-setup-k8s"
BRANCH="main"
FILE_PATH="vault-setup/k8s_auth.sh"

FILE_CONTENT=$(gh api repos/$OWNER/$REPO/contents/$FILE_PATH -q ".content")

echo "$FILE_CONTENT" | base64 -d | bash -s -- garm-operator-controller-manager -n garm-infra-stage-int -p kubernetes -r cicd -f
