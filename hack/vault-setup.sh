#!/bin/bash

set -e

VAULT_NAMESPACE=Action-Runners
VAULT_HOST=https://hcvault.app.corpintra.net

vault write auth/jwt/role/garm-operator - <<EOF
{
  "role_type": "jwt",
  "user_claim": "actor",
  "bound_claims": {
    "repository": "GitHub-Actions/garm-operator"
  },
  "policies": ["cicd"],
  "ttl": "30m"
}
EOF
