#!/bin/sh
#
# Print the CHANGELOG.md section for one version.
#
# The release workflow feeds the output to goreleaser --release-notes.
#
# It stops a release whose notes were never written, rather than
# shipping a generated commit list.

set -eu

if [ $# -lt 1 ]; then
    echo "usage: $0 <version> [changelog]" >&2
    exit 2
fi

version=${1#v}
changelog=${2:-CHANGELOG.md}

notes=$(awk -v v="$version" '
    index($0, "## [" v "]") == 1 { inside = 1; next }
    inside && /^## / { exit }
    inside && /^\[[^]]+\]: / { exit }
    inside { lines[n++] = $0 }
    END {
        first = 0
        while (first < n && lines[first] == "") first++
        last = n - 1
        while (last >= first && lines[last] == "") last--
        for (i = first; i <= last; i++) print lines[i]
    }
' "$changelog")

if [ -z "$notes" ]; then
    echo "$changelog has no section for version $version" >&2
    exit 1
fi

printf '%s\n' "$notes"
