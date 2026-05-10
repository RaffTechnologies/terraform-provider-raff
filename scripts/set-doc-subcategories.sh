#!/bin/bash
# Sets the `subcategory` front-matter field on each generated docs/*.md file.
#
# tfplugindocs emits `subcategory: ""` by default. The Terraform Registry uses
# the subcategory to group the sidebar — but for our provider one umbrella
# bucket is enough, so every resource and data source is tagged with the same
# label. Adds clarity without splitting the sidebar into noisy sub-groups.
#
# Run after every `tfplugindocs generate` (the Makefile's `docs` target chains
# them). Idempotent — replaces the existing subcategory line each run.

set -euo pipefail

cd "$(dirname "$0")/.."

CATEGORY="Virtual Machines"

count=0
for path in docs/resources/*.md docs/data-sources/*.md; do
  [ -f "$path" ] || continue
  if grep -q '^subcategory:' "$path"; then
    # Replace existing subcategory: "..." line in the front-matter block.
    # sed -i.bak with empty backup arg for portability between GNU/BSD sed.
    sed -i.bak -E "s|^subcategory:.*$|subcategory: \"$CATEGORY\"|" "$path"
    rm -f "$path.bak"
    count=$((count + 1))
  else
    echo "warning: no subcategory line in $path (skipping)" >&2
  fi
done

echo "subcategory \"$CATEGORY\" applied to $count docs files"
