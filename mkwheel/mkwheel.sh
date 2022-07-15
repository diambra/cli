#!/bin/bash
set -euo pipefail

mkwheel() {
    local name="$1"; shift
    local version="$1"; shift
    local dest="$1"; shift
    local readme="$1"; shift
    local platform="$1"; shift
    local binary="$1"; shift

    local source
    source="$(dirname "$(realpath -s "$0")")/$name"

    local tempdir
    tempdir="$(mktemp -d)"
    pushd "$tempdir"

    local dist_info="$name-$version.dist-info"
    mkdir -p "$dist_info"

    local tag="py3-none-$platform"

    # METADATA
    VERSION="$version" envsubst '$VERSION' < "$source/metadata.envsubst" > "$dist_info/METADATA"
    cat "$readme" >> "$dist_info/METADATA"

    # WHEEL
    TAG=$tag envsubst '$PLATFORM' < "$source/wheel.envsubst" > "$dist_info/WHEEL"

    # Content
    mkdir -p "$name"
    touch "$name/__init__.py"
    cat <<EOF > "$name/__main__.py"
import os, sys, subprocess
sys.exit(subprocess.call([
	os.path.join(os.path.dirname(__file__), "$name"),
	*sys.argv[1:]
]))
EOF
    cp "$binary" "$name/$name"
    find "$name" -type f | while read -r f; do
        sha=$(openssl dgst -sha256 -binary <<< "$f" | openssl base64 -A)
        size=$(stat -c '%s' "$f")
        echo "$f,sha256=$sha,$size" >> "$dist_info/RECORD"
    done
    cd "$tempdir"
    zip -r "$dest/$name-$version-$tag.whl" .
    popd
}

main() {
    local version="$1"; shift
    local dest="$1"; shift
    declare -A wheels
    wheels=(
        [manylinux2014_x86_64]="linux-amd64"
        [manylinux2014_armv7l]="linux-arm64"
        [win_amd64]="windows-amd64"
        [macosx_10_9_x86_64]="darwin-amd64"
        [macosx_11_0_arm64]="darwin-arm64"
    )
    for wheel in "${!wheels[@]}"; do
        echo "Building $wheel in $dest"
        mkwheel "diambra" "$version" "$(realpath "$dest")" "$(realpath README.md)" "$wheel" "$(realpath bin/diambra.${wheels[$wheel]})"
    done
}

main "$@"