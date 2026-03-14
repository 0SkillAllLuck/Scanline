#!/usr/bin/env bash
# Prepares a new release by updating the version in flake.nix and
# the <releases> section in the metainfo XML file using release
# notes from GitHub releases (via gh CLI).
#
# Usage: ./scripts/prepare-release.sh <new-version>
#   e.g. ./scripts/prepare-release.sh 0.2.0
#
# The new version's release notes are generated from PRs merged
# since the last release. Run this before tagging/releasing.
set -euo pipefail

METAINFO="assets/meta/dev.skillless.Scanline.metainfo.xml"
FLAKE="flake.nix"
REPO_URL="https://github.com/0skillallluck/scanline"

if ! command -v gh &>/dev/null; then
    echo "Error: gh CLI is required" >&2
    exit 1
fi

if [ $# -lt 1 ]; then
    echo "Usage: $0 <new-version>" >&2
    echo "  e.g. $0 0.2.0" >&2
    exit 1
fi

NEW_VERSION="$1"
NEW_TAG="v${NEW_VERSION}"
TODAY=$(date +%Y-%m-%d)

# --- Update version in flake.nix ---
echo "Updating version in $FLAKE to $NEW_VERSION..." >&2
sed -i -E "s/(version = \")([^\"]+)(\";)/\1${NEW_VERSION}\3/" "$FLAKE"

# Parse a markdown release body into XML description elements
parse_body() {
    local body="$1"
    local description=""
    local current_section=""

    while IFS= read -r line; do
        # Detect section headers (### New Features, ### Bug Fixes, etc.)
        if [[ "$line" =~ ^###[[:space:]]+(.*) ]]; then
            section="${BASH_REMATCH[1]}"
            if [ -n "$current_section" ]; then
                description+=$'        </ul>\n'
            fi
            description+="        <p>${section}:</p>"$'\n'
            description+=$'        <ul>\n'
            current_section="$section"
            continue
        fi

        # Detect list items (* item by @author in https://...)
        if [[ "$line" =~ ^\*[[:space:]]+(.*) ]]; then
            item="${BASH_REMATCH[1]}"
            # Skip "first contribution" lines
            if [[ "$item" =~ "made their first contribution" ]]; then
                continue
            fi
            # Strip " by @user in https://..." suffix
            item=$(echo "$item" | sed -E 's/ by @[^ ]+ in https:\/\/[^ ]+$//')
            # Strip category prefixes (e.g. "feat: ", "bug: ", "dependencies: ")
            item=$(echo "$item" | sed -E 's/^(feat|bug|fix|dependencies|deps|chore|docs|refactor|test|ci|build|style|perf): //i')
            # Escape XML special characters
            item=$(echo "$item" | sed 's/&/\&amp;/g; s/</\&lt;/g; s/>/\&gt;/g')
            description+="          <li>${item}</li>"$'\n'
        fi
    done <<< "$body"

    # Close last list
    if [ -n "$current_section" ]; then
        description+=$'        </ul>\n'
    fi

    echo -n "$description"
}

releases_xml=""

# --- Unreleased changes (upcoming release) ---
# Find the latest existing release tag to use as the base
latest_tag=$(gh release list --json tagName --jq '.[0].tagName' --limit 1 2>/dev/null || true)

if [ -n "$latest_tag" ]; then
    echo "Generating notes for $NEW_TAG (changes since $latest_tag)..." >&2
    unreleased_body=$(gh api repos/{owner}/{repo}/releases/generate-notes \
        -f tag_name="$NEW_TAG" \
        -f previous_tag_name="$latest_tag" \
        --jq .body)
else
    echo "Generating notes for $NEW_TAG (first release)..." >&2
    unreleased_body=$(gh api repos/{owner}/{repo}/releases/generate-notes \
        -f tag_name="$NEW_TAG" \
        --jq .body)
fi

description=$(parse_body "$unreleased_body")

if [ -n "$description" ]; then
    releases_xml+="    <release version=\"${NEW_VERSION}\" date=\"${TODAY}\">"$'\n'
    releases_xml+="      <url type=\"details\">${REPO_URL}/releases/tag/${NEW_TAG}</url>"$'\n'
    releases_xml+=$'      <description>\n'
    releases_xml+="${description}"$'\n'
    releases_xml+=$'      </description>\n'
    releases_xml+=$'    </release>\n'
else
    echo "  Warning: no parseable notes for upcoming release" >&2
fi

# --- Existing releases ---
mapfile -t tags < <(gh release list --json tagName --jq '.[].tagName' --limit 100)
mapfile -t dates < <(gh release list --json publishedAt --jq '.[].publishedAt' --limit 100)

for i in "${!tags[@]}"; do
    tag="${tags[$i]}"
    date="${dates[$i]%%T*}"
    version="${tag#v}"

    echo "Processing $tag..." >&2

    body=$(gh release view "$tag" --json body --jq .body)
    description=$(parse_body "$body")

    if [ -z "$description" ]; then
        echo "  Skipping $tag (no parseable release notes)" >&2
        continue
    fi

    releases_xml+="    <release version=\"${version}\" date=\"${date}\">"$'\n'
    releases_xml+="      <url type=\"details\">${REPO_URL}/releases/tag/${tag}</url>"$'\n'
    releases_xml+=$'      <description>\n'
    releases_xml+="${description}"$'\n'
    releases_xml+=$'      </description>\n'
    releases_xml+=$'    </release>\n'
done

# Build the full releases block
releases_block="<releases>"$'\n'"${releases_xml}  </releases>"

# Replace the <releases>...</releases> section in the metainfo file
content=$(cat "$METAINFO")
before="${content%%<releases>*}"
after="${content##*</releases>}"
printf '%s%s%s' "$before" "$releases_block" "$after" > "$METAINFO"

total=$((${#tags[@]} + 1))
echo "Updated $METAINFO with $total release(s)." >&2
