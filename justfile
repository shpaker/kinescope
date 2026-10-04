gocmd := "go"
lab := "./cmd/kinescope-lab"

# Release tags are vMAJOR.MINOR.PATCH, as Go modules require; a release
# raises MINOR (or MAJOR), PATCH stays 0
release_tag := '^v[0-9]+\.[0-9]+\.[0-9]+$'
# Commit types that make a release, in changelog order
release_types := "feat fix perf refactor"
# A breaking change: `type!:` in the subject or a BREAKING CHANGE footer
breaking_subject := '^[a-z]+(\([^)]*\))?!:'
breaking_body := '^BREAKING[ -]CHANGE:'

default:
    @just --list

deps:
    {{gocmd}} mod download

install-tools:
    {{gocmd}} install golang.org/x/tools/cmd/goimports@latest
    {{gocmd}} install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.8

fmt:
    gofmt -s -w .
    goimports -local github.com/shpaker/kinescope -w .

fmt-check:
    #!/bin/bash
    set -euo pipefail
    unformatted=$(gofmt -s -l .)
    if [ -n "$unformatted" ]; then
        echo "Not formatted:"
        echo "$unformatted"
        exit 1
    fi

lint:
    golangci-lint run ./...

lint-wasm:
    GOOS=js GOARCH=wasm golangci-lint run ./...

test:
    {{gocmd}} test ./...

# Build the lab for the web into dist/; the version it shows is the
# release tag given, or what git describes
build-web version="":
    #!/bin/bash
    set -euo pipefail
    rm -rf dist
    mkdir -p dist
    version='{{version}}'
    version=${version:-$(git describe --tags --always 2>/dev/null || echo dev)}
    GOOS=js GOARCH=wasm {{gocmd}} build -trimpath -ldflags "-s -w -X main.version=$version" -o dist/kinescope-lab.wasm {{lab}}
    cp "$({{gocmd}} env GOROOT)/lib/wasm/wasm_exec.js" dist/
    cp web/* dist/
    echo "Web build stored in dist/"

# Serve the lab on localhost:8080
serve-web: build-web
    python -m http.server 8080 --directory dist

# The next release tag from the commits since the last one; empty when
# nothing calls for a release. At v0 a breaking change raises MINOR.
next-version:
    #!/bin/bash
    set -euo pipefail
    last=$(git tag --merged HEAD | grep -E '{{release_tag}}' | sort -V | tail -n1 || true)
    range="${last:+$last..}HEAD"
    subjects=$(git log --format='%s' "$range")
    bodies=$(git log --format='%b' "$range")
    types='{{release_types}}'
    version="${last#v}"
    major="${version%%.*}"
    rest="${version#*.}"
    minor="${rest%%.*}"
    if [ -z "$last" ]; then
        major=0
        minor=0
    fi
    if [ "$major" -gt 0 ] && { grep -qE '{{breaking_subject}}' <<< "$subjects"         || grep -qE '{{breaking_body}}' <<< "$bodies"; }; then
        echo "v$((major + 1)).0.0"
    elif grep -qE '{{breaking_subject}}' <<< "$subjects"         || grep -qE '{{breaking_body}}' <<< "$bodies"         || grep -qE "^(${types// /|})(\([^)]*\))?:" <<< "$subjects"; then
        echo "v${major}.$((minor + 1)).0"
    fi

# The changelog of release ref since the release before it: breaking
# changes first, then by type; all commits if none of them qualify
changelog ref="HEAD":
    #!/bin/bash
    set -euo pipefail
    ref='{{ref}}'
    last=$(git tag --merged "$ref" --no-contains "$ref" | grep -E '{{release_tag}}' | sort -V | tail -n1 || true)
    subjects=$(git log --reverse --format='%s' "${last:+$last..}$ref")
    changes=$(
        grep -E '{{breaking_subject}}' <<< "$subjects" || true
        for type in {{release_types}}; do
            grep -E "^${type}(\([^)]*\))?:" <<< "$subjects" || true
        done
    )
    sed '/^$/d; s/^/- /' <<< "${changes:-$subjects}"
