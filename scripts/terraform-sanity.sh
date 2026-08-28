#!/usr/bin/env bash
# Terraform-level sanity check: builds the real provider binary, points a
# throwaway Terraform CLI config at it via a dev override (see
# https://developer.hashicorp.com/terraform/cli/config/config-file#development-overrides-for-provider-developers),
# then runs a minimal config (scripts/terraform-sanity/main.tf) that logs in
# and lists the edge summary through the real Terraform plugin protocol.
#
# This is the Terraform-driven counterpart to `make sanity` (cmd/sanity),
# which talks to the SDK directly with no Terraform involved at all — use
# this one when you want to confirm the provider binary itself behaves
# correctly end to end, not just that credentials/connectivity work.
#
# Usage:
#   export GRAPHIANT_ACCESS_TOKEN="..."   # or GRAPHIANT_USERNAME + GRAPHIANT_PASSWORD
#   ./scripts/terraform-sanity.sh

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
sanity_dir="$repo_root/scripts/terraform-sanity"

if ! command -v terraform >/dev/null 2>&1; then
  echo "error: terraform CLI not found on PATH" >&2
  exit 1
fi

if [ -z "${GRAPHIANT_ACCESS_TOKEN:-}" ] && { [ -z "${GRAPHIANT_USERNAME:-}" ] || [ -z "${GRAPHIANT_PASSWORD:-}" ]; }; then
  echo "error: set GRAPHIANT_ACCESS_TOKEN, or GRAPHIANT_USERNAME + GRAPHIANT_PASSWORD" >&2
  exit 1
fi

echo "Building provider binary..."
(cd "$repo_root" && go build -o terraform-provider-graphiant .)

cli_config="$(mktemp -t terraform-sanity-rc.XXXXXX)"
cleanup() {
  rm -f "$cli_config"
  rm -rf "$sanity_dir/.terraform" "$sanity_dir/.terraform.lock.hcl"
  rm -f "$sanity_dir"/terraform.tfstate*
}
trap cleanup EXIT

cat >"$cli_config" <<EOF
provider_installation {
  dev_overrides {
    "Graphiant-Inc/graphiant" = "$repo_root"
  }
  direct {}
}
EOF

echo "Running terraform apply against scripts/terraform-sanity/main.tf..."
echo "(dev override in place — no 'terraform init' needed)"
echo

TF_CLI_CONFIG_FILE="$cli_config" terraform -chdir="$sanity_dir" apply -auto-approve -no-color
