#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
VERSION="${1:-${GITHUB_REF_NAME:-dev}}"
OUT_DIR="${OUT_DIR:-$ROOT/dist}"

if [[ "$OUT_DIR" != /* ]]; then
  OUT_DIR="$ROOT/$OUT_DIR"
fi
if [[ "$OUT_DIR" == *..* ]]; then
  echo "ERR OUT_DIR must not contain '..': $OUT_DIR" >&2
  exit 1
fi
case "$OUT_DIR" in
  "$ROOT/dist"|"$ROOT/dist"/*) ;;
  *)
    echo "ERR OUT_DIR must stay under $ROOT/dist: $OUT_DIR" >&2
    exit 1
    ;;
esac

rm -rf "$OUT_DIR"
mkdir -p "$OUT_DIR"

build_one() {
  local goos="$1"
  local goarch="$2"
  local binary="jira_${VERSION}_${goos}_${goarch}"

  if [[ "$goos" == "windows" ]]; then
    binary="${binary}.exe"
  fi

  (
    cd "$ROOT"
    GOOS="$goos" GOARCH="$goarch" CGO_ENABLED=0 \
      go build -trimpath \
      -ldflags "-s -w -X github.com/sean2077/jira-cli/internal/cli.Version=$VERSION" \
      -o "$OUT_DIR/$binary" ./cmd/jira
  )
}

build_one linux amd64
build_one linux arm64
build_one darwin amd64
build_one darwin arm64
build_one windows amd64
build_one windows arm64

(
  cd "$OUT_DIR"
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum jira_"$VERSION"_* > checksums.txt
  else
    shasum -a 256 jira_"$VERSION"_* > checksums.txt
  fi
)
