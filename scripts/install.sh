#!/bin/sh
set -eu

# Bootstrap vibeCodingSecretManager into the current application repository.
#
# Local project install:
#   ./scripts/install.sh
#
# Remote one-liner from an application repo:
#   VCSM_REPO_URL=https://github.com/Yaling7788/vibeCodingSecretManager.git \
#   sh -c "$(curl -fsSL https://raw.githubusercontent.com/Yaling7788/vibeCodingSecretManager/main/scripts/install.sh)"
#
# Optional inputs:
#   VCSM_PROJECT=sample-webapp
#   VCSM_ENV=dev
#   VCSM_SECRETS=DATABASE_URL,OPENAI_API_KEY,RESEND_API_KEY
#   VCSM_DATABASE=~/KeePass/example-dev.kdbx
#   VCSM_KEY_FILE=~/KeePass/example-dev.key
#   VCSM_CLI_PATH=/Applications/KeePassXC.app/Contents/MacOS/keepassxc-cli
#   VCSM_INSTALL_KEEPASSXC=1

say() {
  printf '%s\n' "$*"
}

need() {
  if ! command -v "$1" >/dev/null 2>&1; then
    say "error: required command not found: $1" >&2
    exit 1
  fi
}

expand_tilde() {
  case "$1" in
    "~") printf '%s' "$HOME" ;;
    "~/"*) printf '%s/%s' "$HOME" "${1#~/}" ;;
    *) printf '%s' "$1" ;;
  esac
}

detect_keepassxc_cli() {
  if [ -n "${VCSM_CLI_PATH:-}" ]; then
    if [ -x "$VCSM_CLI_PATH" ] || command -v "$VCSM_CLI_PATH" >/dev/null 2>&1; then
      printf '%s' "$VCSM_CLI_PATH"
      return
    fi
  fi

  if command -v keepassxc-cli >/dev/null 2>&1; then
    command -v keepassxc-cli
    return
  fi

  if [ -x "/Applications/KeePassXC.app/Contents/MacOS/keepassxc-cli" ]; then
    printf '%s' "/Applications/KeePassXC.app/Contents/MacOS/keepassxc-cli"
    return
  fi

  printf '%s' "auto"
}

as_root() {
  if [ "$(id -u 2>/dev/null || printf 1)" = "0" ]; then
    "$@"
    return
  fi

  if command -v sudo >/dev/null 2>&1; then
    sudo "$@"
    return
  fi

  say "error: '$*' requires root privileges, but sudo was not found." >&2
  exit 1
}

verify_keepassxc_install() {
  if [ "$(detect_keepassxc_cli)" != "auto" ]; then
    return
  fi

  say "KeePassXC installation command completed, but keepassxc-cli still was not found in this shell." >&2
  say "Open a new terminal, or rerun with VCSM_CLI_PATH=/path/to/keepassxc-cli." >&2
  exit 1
}

install_keepassxc_macos() {
  if command -v brew >/dev/null 2>&1; then
    say "KeePassXC CLI was not found. Installing KeePassXC with Homebrew..."
    brew install --cask keepassxc
    verify_keepassxc_install
    return
  fi

  say "KeePassXC CLI was not found, and Homebrew is not installed." >&2
  say "Install KeePassXC manually from https://keepassxc.org/download/ or install Homebrew, then rerun this script." >&2
  exit 1
}

install_keepassxc_linux() {
  say "KeePassXC CLI was not found. Detecting Linux package manager..."

  if command -v apt-get >/dev/null 2>&1; then
    as_root apt-get update
    as_root apt-get install -y keepassxc
    verify_keepassxc_install
    return
  fi

  if command -v dnf >/dev/null 2>&1; then
    as_root dnf install -y keepassxc
    verify_keepassxc_install
    return
  fi

  if command -v yum >/dev/null 2>&1; then
    as_root yum install -y keepassxc
    verify_keepassxc_install
    return
  fi

  if command -v pacman >/dev/null 2>&1; then
    as_root pacman -S --noconfirm keepassxc
    verify_keepassxc_install
    return
  fi

  if command -v zypper >/dev/null 2>&1; then
    as_root zypper --non-interactive install keepassxc
    verify_keepassxc_install
    return
  fi

  if command -v apk >/dev/null 2>&1; then
    as_root apk add keepassxc
    verify_keepassxc_install
    return
  fi

  if command -v nix >/dev/null 2>&1; then
    nix profile install nixpkgs#keepassxc
    verify_keepassxc_install
    return
  fi

  say "KeePassXC CLI was not found, and no supported Linux package manager was detected." >&2
  say "Install KeePassXC manually, then rerun this script." >&2
  say "Examples: sudo apt-get install keepassxc | sudo dnf install keepassxc | sudo pacman -S keepassxc" >&2
  exit 1
}

install_keepassxc_windows() {
  say "KeePassXC CLI was not found. Detecting Windows package manager..."

  if command -v winget >/dev/null 2>&1; then
    winget install --id KeePassXCTeam.KeePassXC --exact --accept-package-agreements --accept-source-agreements
    verify_keepassxc_install
    return
  fi

  if command -v choco >/dev/null 2>&1; then
    choco install keepassxc -y
    verify_keepassxc_install
    return
  fi

  if command -v scoop >/dev/null 2>&1; then
    scoop install keepassxc
    verify_keepassxc_install
    return
  fi

  say "KeePassXC CLI was not found, and winget/choco/scoop were not detected." >&2
  say "Install KeePassXC manually from https://keepassxc.org/download/, then rerun this script." >&2
  exit 1
}

install_keepassxc_if_needed() {
  if [ "${VCSM_INSTALL_KEEPASSXC:-1}" = "0" ]; then
    return
  fi

  detected="$(detect_keepassxc_cli)"
  if [ "$detected" != "auto" ]; then
    return
  fi

  os_name="$(uname -s 2>/dev/null || printf unknown)"
  case "$os_name" in
    Darwin)
      install_keepassxc_macos
      ;;
    Linux)
      install_keepassxc_linux
      ;;
    MINGW*|MSYS*|CYGWIN*)
      install_keepassxc_windows
      ;;
    *)
      if [ "${OS:-}" = "Windows_NT" ]; then
        install_keepassxc_windows
      else
        say "KeePassXC CLI was not found on unsupported OS '$os_name'." >&2
        say "Install KeePassXC manually, or rerun with VCSM_CLI_PATH=/path/to/keepassxc-cli." >&2
        exit 1
      fi
      ;;
  esac
}

repo_root="$(CDPATH= cd -- "$(dirname -- "$0")/.." 2>/dev/null && pwd || true)"
if [ -n "$repo_root" ] && [ -f "$repo_root/go.mod" ] && [ -d "$repo_root/cmd/vibeCodingSecretManager" ]; then
  source_dir="$repo_root"
  cleanup=""
else
  need git
  repo_url="${VCSM_REPO_URL:-https://github.com/Yaling7788/vibeCodingSecretManager.git}"
  tmp_dir="$(mktemp -d)"
  cleanup="$tmp_dir"
  git clone --depth 1 "$repo_url" "$tmp_dir/vibeCodingSecretManager" >/dev/null
  source_dir="$tmp_dir/vibeCodingSecretManager"
fi

need go

install_keepassxc_if_needed

(cd "$source_dir" && go install ./cmd/vibeCodingSecretManager)

project="${VCSM_PROJECT:-$(basename "$PWD")}"
environment="${VCSM_ENV:-dev}"
database="$(expand_tilde "${VCSM_DATABASE:-~/KeePass/example-dev.kdbx}")"
key_file="${VCSM_KEY_FILE:-}"
cli_path="${VCSM_CLI_PATH:-$(detect_keepassxc_cli)}"
config_dir="${XDG_CONFIG_HOME:-$HOME/.config}/vibeCodingSecretManager"
config_file="$config_dir/config.yaml"
project_root="$PWD"
secrets_csv="${VCSM_SECRETS:-DATABASE_URL}"

mkdir -p "$config_dir" scripts

if [ ! -f "$config_file" ]; then
  {
    say "vault:"
    say "  type: keepassxc"
    say "  database: $database"
    if [ -n "$key_file" ]; then
      say "  key_file: $(expand_tilde "$key_file")"
    fi
    say "  cli_path: $cli_path"
    say ""
    say "projects:"
    say "  $project:"
    say "    root: $project_root"
    say "    environments:"
    say "      $environment:"
    say "        secrets:"
    old_ifs="$IFS"
    IFS=","
    for secret in $secrets_csv; do
      trimmed="$(printf '%s' "$secret" | tr -d '[:space:]')"
      [ -n "$trimmed" ] || continue
      say "          $trimmed: ${project}/${environment}/${trimmed}"
    done
    IFS="$old_ifs"
  } >"$config_file"
  chmod 600 "$config_file"
  say "Created config: $config_file"
else
  say "Config already exists: $config_file"
  say "Review it manually if you need to add project '$project'."
fi

cat > scripts/secret-dev <<EOF
#!/bin/sh
set -eu

COMMAND="\${1:-}"

case "\$COMMAND" in
  up)
    exec vibeCodingSecretManager run $project $environment -- npm run dev
    ;;
  docker)
    exec vibeCodingSecretManager run $project $environment -- docker compose up
    ;;
  build)
    exec vibeCodingSecretManager run $project $environment -- npm run build
    ;;
  test)
    exec npm test
    ;;
  lint)
    exec npm run lint
    ;;
  check-secrets)
    exec vibeCodingSecretManager check $project $environment
    ;;
  list-secrets)
    exec vibeCodingSecretManager list $project $environment
    ;;
  *)
    echo "Usage: ./scripts/secret-dev {up|docker|build|test|lint|check-secrets|list-secrets}"
    exit 1
    ;;
esac
EOF
chmod +x scripts/secret-dev

touch .env.example .gitignore
if ! grep -qxF ".env" .gitignore; then
  {
    say ""
    say "# Secret files"
    say ".env"
    say ".env.*"
    say "!.env.example"
    say "*.kdbx"
    say "*.kdbx.lock"
  } >> .gitignore
fi

old_ifs="$IFS"
IFS=","
for secret in $secrets_csv; do
  trimmed="$(printf '%s' "$secret" | tr -d '[:space:]')"
  [ -n "$trimmed" ] || continue
  if ! grep -q "^$trimmed=" .env.example; then
    say "$trimmed=replace_me" >> .env.example
  fi
done
IFS="$old_ifs"

if [ ! -f CLAUDE.md ]; then
  cat > CLAUDE.md <<EOF
# AI Coding Secret Rules

Secrets are managed by KeePassXC and injected through vibeCodingSecretManager.

You may run:

\`\`\`bash
./scripts/secret-dev up
./scripts/secret-dev docker
./scripts/secret-dev build
./scripts/secret-dev test
./scripts/secret-dev lint
./scripts/secret-dev check-secrets
./scripts/secret-dev list-secrets
\`\`\`

You must not read, print, retrieve, inspect, summarize, modify, or infer real secret values.

Do not run: \`env\`, \`printenv\`, \`set\`, \`export\`, \`keepassxc-cli\`, \`cat .env*\`, \`rg .env*\`, cloud secret read commands, or clipboard/history reads.

Use \`.env.example\` only for placeholder names.
EOF
fi

if [ -n "$cleanup" ]; then
  rm -rf "$cleanup"
fi

say ""
say "Installed vibeCodingSecretManager."
say "Wrapper created: ./scripts/secret-dev"
say "Config path: $config_file"
say ""
say "Next human-only steps:"
say "1. Open KeePassXC and create database entries for the listed secret paths."
say "2. Put real values only in KeePassXC entry Password fields."
say "3. Run: ./scripts/secret-dev list-secrets"
say "4. Run: ./scripts/secret-dev check-secrets"
say ""
say "The master password must be typed by the human into the local hidden prompt."
