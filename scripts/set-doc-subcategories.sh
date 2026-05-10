#!/bin/bash
# Sets the `subcategory` front-matter field on each generated docs/*.md file
# so the Terraform Registry groups them in the sidebar by product area
# (mirrors digitalocean/digitalocean's layout).
#
# tfplugindocs emits `subcategory: ""` by default. Run this after every
# `tfplugindocs generate` (the Makefile's `docs` target chains them).
#
# Idempotent — replaces the existing subcategory line each run.
#
# Uses a case statement (not associative arrays) for macOS bash 3.2 compat.

set -euo pipefail

cd "$(dirname "$0")/.."

# resource-or-data-source name → sidebar category
category_for() {
  case "$1" in
    # VMs
    vm|vms|vm_pricing|templates) echo "VMs" ;;
    # Volumes
    volume|volumes|volume_pricing) echo "Volumes" ;;
    # Snapshots
    snapshot|snapshots|snapshot_pricing) echo "Snapshots" ;;
    # Backups
    backup|backups|backup_schedule|backup_schedules|backup_pricing) echo "Backups" ;;
    # Networking
    vpc|vpcs|ip|ips|security_group|security_groups|ip_pricing) echo "Networking" ;;
    # Projects & Access (IAM)
    project|projects|project_member|project_members|member|members|role|roles|api_key|api_keys|ssh_key|ssh_keys) echo "Projects & Access" ;;
    # Reference
    regions) echo "Reference" ;;
    *) echo "" ;;
  esac
}

for path in docs/resources/*.md docs/data-sources/*.md; do
  [ -f "$path" ] || continue
  name="$(basename "$path" .md)"
  cat="$(category_for "$name")"
  if [ -z "$cat" ]; then
    echo "warning: no category mapping for $path" >&2
    continue
  fi
  # Replace existing subcategory: "..." line in the front-matter block.
  # Uses sed -i with empty backup arg for portability between GNU/BSD sed.
  if grep -q '^subcategory:' "$path"; then
    sed -i.bak -E "s|^subcategory:.*$|subcategory: \"$cat\"|" "$path"
    rm -f "$path.bak"
  else
    echo "warning: no subcategory line in $path (skipping)" >&2
  fi
done

echo "subcategories applied across $(find docs/resources docs/data-sources -name '*.md' | wc -l | tr -d ' ') docs files"
