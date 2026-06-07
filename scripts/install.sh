#!/usr/bin/env sh
set -eu

REPO="${JIRA_CLI_REPO:-sean2077/jira-cli}"
VERSION="${JIRA_CLI_VERSION:-latest}"
INSTALL_DIR="${JIRA_CLI_INSTALL_DIR:-/usr/local/bin}"
TOKEN="${GH_TOKEN:-${GITHUB_TOKEN:-}}"

usage() {
  cat <<'EOF'
Install jira-cli from GitHub Releases.

Usage:
  install.sh [VERSION] [--dir DIR]

Examples:
  curl -fsSL https://github.com/sean2077/jira-cli/raw/refs/heads/main/scripts/install.sh | sh
  curl -fsSL https://github.com/sean2077/jira-cli/raw/refs/heads/main/scripts/install.sh | sh -s -- --dir "$HOME/.local/bin"
  curl -fsSL https://github.com/sean2077/jira-cli/raw/refs/heads/main/scripts/install.sh | sh -s -- v1.0.0

Environment:
  GH_TOKEN              Optional token for private repository release assets.
  GITHUB_TOKEN          Optional token fallback for private release assets.
  JIRA_CLI_VERSION      Optional version tag. Defaults to latest.
  JIRA_CLI_INSTALL_DIR  Install directory. Defaults to /usr/local/bin.
  JIRA_CLI_REPO         GitHub repo. Defaults to sean2077/jira-cli.
EOF
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --dir)
      [ "$#" -ge 2 ] || {
        echo "ERR --dir requires a value" >&2
        exit 2
      }
      INSTALL_DIR="$2"
      shift 2
      ;;
    --help|-h)
      usage
      exit 0
      ;;
    -*)
      echo "ERR unknown flag: $1" >&2
      usage >&2
      exit 2
      ;;
    *)
      VERSION="$1"
      shift
      ;;
  esac
done

need() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "ERR missing required command: $1" >&2
    exit 1
  }
}

need curl
need uname

curl_text() {
  if [ -n "$TOKEN" ]; then
    curl -fsSL -H "Authorization: Bearer $TOKEN" "$@"
  else
    curl -fsSL "$@"
  fi
}

curl_download() {
  if [ -n "$TOKEN" ]; then
    curl -fL -H "Authorization: Bearer $TOKEN" "$@"
  else
    curl -fL "$@"
  fi
}

curl_asset_api() {
  curl -fL \
    -H "Authorization: Bearer $TOKEN" \
    -H "Accept: application/octet-stream" \
    "$@"
}

asset_api_url() {
  wanted_asset="$1"
  release_json="$(curl_text "https://api.github.com/repos/$REPO/releases/tags/$VERSION")"
  printf '%s\n' "$release_json" | awk -v asset="$wanted_asset" '
    /^[[:space:]]*"url":[[:space:]]*"https:\/\/api.github.com\/repos\/.*\/releases\/assets\// {
      url = $0
      sub(/^[[:space:]]*"url":[[:space:]]*"/, "", url)
      sub(/".*$/, "", url)
    }
    /^[[:space:]]*"name":[[:space:]]*"/ {
      name = $0
      sub(/^[[:space:]]*"name":[[:space:]]*"/, "", name)
      sub(/".*$/, "", name)
      if (name == asset && url != "") {
        print url
        exit
      }
    }
  '
}

latest_version() {
  latest_url="$(curl -fsSL -o /dev/null -w '%{url_effective}' "https://github.com/$REPO/releases/latest" 2>/dev/null || true)"
  case "$latest_url" in
    */releases/tag/*)
      tag="${latest_url##*/releases/tag/}"
      tag="${tag%%\?*}"
      tag="${tag%%#*}"
      if [ -n "$tag" ]; then
        printf '%s\n' "$tag"
        return 0
      fi
      ;;
  esac

  curl_text "https://api.github.com/repos/$REPO/releases/latest" | sed -n 's/.*"tag_name":[[:space:]]*"\([^"]*\)".*/\1/p' | head -n 1
}

if [ "$VERSION" = "latest" ]; then
  VERSION="$(latest_version)"
  [ -n "$VERSION" ] || {
    echo "ERR could not resolve latest release for $REPO" >&2
    exit 1
  }
fi

case "$(uname -s)" in
  Linux) os="linux" ;;
  Darwin) os="darwin" ;;
  *)
    echo "ERR unsupported OS: $(uname -s)" >&2
    exit 1
    ;;
esac

case "$(uname -m)" in
  x86_64|amd64) arch="amd64" ;;
  arm64|aarch64) arch="arm64" ;;
  *)
    echo "ERR unsupported architecture: $(uname -m)" >&2
    exit 1
    ;;
esac

asset="jira_${VERSION}_${os}_${arch}"
url="https://github.com/$REPO/releases/download/$VERSION/$asset"
tmp="$(mktemp)"
checksums_tmp="$(mktemp)"
trap 'rm -f "$tmp" "$checksums_tmp"' EXIT INT TERM

echo "Downloading $url"
if [ -n "$TOKEN" ]; then
  api_url="$(asset_api_url "$asset")"
  if [ -z "$api_url" ]; then
    echo "ERR could not find release asset $asset in $REPO@$VERSION" >&2
    exit 1
  fi
  download_cmd_status=0
  curl_asset_api -o "$tmp" "$api_url" || download_cmd_status=$?
else
  download_cmd_status=0
  curl_download -o "$tmp" "$url" || download_cmd_status=$?
fi
if [ "$download_cmd_status" -ne 0 ]; then
  echo "ERR download failed. If $REPO is private, export GH_TOKEN or GITHUB_TOKEN with repo read access." >&2
  exit 1
fi

echo "Downloading checksums.txt"
if [ -n "$TOKEN" ]; then
  checksums_api_url="$(asset_api_url "checksums.txt")"
  if [ -z "$checksums_api_url" ]; then
    echo "ERR could not find release asset checksums.txt in $REPO@$VERSION" >&2
    exit 1
  fi
  curl_asset_api -o "$checksums_tmp" "$checksums_api_url"
else
  curl_download -o "$checksums_tmp" "https://github.com/$REPO/releases/download/$VERSION/checksums.txt"
fi

expected_sha="$(awk -v asset="$asset" '$2 == asset || $2 == "./" asset { print $1; exit }' "$checksums_tmp")"
if [ -z "$expected_sha" ]; then
  echo "ERR checksums.txt does not include $asset" >&2
  exit 1
fi
if command -v sha256sum >/dev/null 2>&1; then
  actual_sha="$(sha256sum "$tmp" | awk '{print $1}')"
elif command -v shasum >/dev/null 2>&1; then
  actual_sha="$(shasum -a 256 "$tmp" | awk '{print $1}')"
else
  echo "ERR missing required command: sha256sum or shasum" >&2
  exit 1
fi
if [ "$actual_sha" != "$expected_sha" ]; then
  echo "ERR checksum mismatch for $asset" >&2
  exit 1
fi
echo "Verified checksum for $asset"
chmod 0755 "$tmp"

if [ ! -d "$INSTALL_DIR" ]; then
  if ! mkdir -p "$INSTALL_DIR" 2>/dev/null; then
    command -v sudo >/dev/null 2>&1 || {
      echo "ERR cannot create $INSTALL_DIR and sudo is unavailable" >&2
      exit 1
    }
    sudo mkdir -p "$INSTALL_DIR"
  fi
fi

target="$INSTALL_DIR/jira"
if [ -w "$INSTALL_DIR" ]; then
  install -m 0755 "$tmp" "$target"
else
  command -v sudo >/dev/null 2>&1 || {
    echo "ERR $INSTALL_DIR is not writable and sudo is unavailable" >&2
    exit 1
  }
  sudo install -m 0755 "$tmp" "$target"
fi

echo "Installed $target"
"$target" version

case ":$PATH:" in
  *":$INSTALL_DIR:"*) ;;
  *)
    echo "NOTE $INSTALL_DIR is not on PATH. Add it before running jira from any shell."
    ;;
esac
