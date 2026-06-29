#!/usr/bin/env bash
# Canonical Homebrew-formula version/sha bump, reused for the in-tree formula
# and the tap repo formula so the rewrite lives in exactly one place.
#
#   bump-formula.sh <formula-path> <version-without-v> <sha256>
set -euo pipefail

file="$1"
version="$2"
sha256="$3"

# Validate inputs up front: the formula's url pins a strict MAJOR.MINOR.PATCH
# tag and a 64-hex sha256, so reject anything the rewrite can't faithfully land.
[[ "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] \
  || { echo "bump-formula: version '$version' is not MAJOR.MINOR.PATCH" >&2; exit 1; }
[[ "$sha256" =~ ^[a-f0-9]{64}$ ]] \
  || { echo "bump-formula: sha256 '$sha256' is not 64 lowercase hex chars" >&2; exit 1; }
[[ -f "$file" ]] || { echo "bump-formula: no such file: $file" >&2; exit 1; }

# perl -i behaves identically on GNU/Linux (the CI runner) and BSD/macOS,
# unlike `sed -i`, so the bump stays testable locally.
VERSION="$version" SHA256="$sha256" perl -i -pe '
  s{archive/refs/tags/v[0-9]+\.[0-9]+\.[0-9]+\.tar\.gz}{archive/refs/tags/v$ENV{VERSION}.tar.gz};
  s{^(  sha256 ")[a-f0-9]+(")}{$1$ENV{SHA256}$2};
' "$file"

# Treat the bump as a transaction: assert the file actually holds the intended
# url and sha256 now, so a formula whose shape drifted fails loudly here rather
# than being committed and released as a silent no-op.
grep -qF "archive/refs/tags/v${version}.tar.gz" "$file" \
  || { echo "bump-formula: url not updated to v${version} in $file" >&2; exit 1; }
grep -qE "^  sha256 \"${sha256}\"\$" "$file" \
  || { echo "bump-formula: sha256 not updated in $file" >&2; exit 1; }
