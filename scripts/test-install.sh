#!/usr/bin/env bash
# Exercise downloads and installation without network access or sudo.
set -euo pipefail
PROJECT_DIR=$(cd "$(dirname "$0")/.." && pwd)
FIXTURE_DIR=$(mktemp -d)
trap 'rm -rf "$FIXTURE_DIR"' EXIT
export FIXTURE_DIR
mkdir -p "$FIXTURE_DIR/bin" "$FIXTURE_DIR/assets"

cat > "$FIXTURE_DIR/bin/uname" <<'MOCK'
#!/usr/bin/env bash
case "$1" in -s) echo Linux ;; -m) echo x86_64 ;; esac
MOCK
cat > "$FIXTURE_DIR/bin/curl" <<'MOCK'
#!/usr/bin/env bash
set -euo pipefail
while [ "$#" -gt 0 ]; do
    case "$1" in
        -o) DEST="$2"; shift 2 ;;
        -w) shift 2 ;;
        -*) shift ;;
        *) URL="$1"; shift ;;
    esac
done
if [[ "$URL" == */releases/latest ]]; then
    printf 'https://github.com/corywilkerson/webp2ascii/releases/tag/v1.1.0'
else
    cp "$FIXTURE_DIR/assets/${URL##*/}" "$DEST"
fi
MOCK
chmod +x "$FIXTURE_DIR/bin/"*
export PATH="$FIXTURE_DIR/bin:$PATH"

make_archive() {
    local name="webp2ascii-$1-linux-amd64"
    printf '#!/usr/bin/env bash\necho %s\n' "$1" > "$FIXTURE_DIR/$name"
    tar -czf "$FIXTURE_DIR/assets/$name.tar.gz" -C "$FIXTURE_DIR" "$name"
}
make_archive 1.1.0
(cd "$FIXTURE_DIR/assets" && shasum -a 256 ./*.tar.gz > SHA256SUMS)

INSTALL_DIR="$FIXTURE_DIR/latest install" VERSION=latest bash "$PROJECT_DIR/install.sh" >/dev/null
test "$("$FIXTURE_DIR/latest install/webp2ascii")" = 1.1.0
INSTALL_DIR="$FIXTURE_DIR/pinned" VERSION=v1.1.0 bash "$PROJECT_DIR/install.sh" >/dev/null
test "$("$FIXTURE_DIR/pinned/webp2ascii")" = 1.1.0

# Existing releases have differently versioned filenames and no checksums.
make_archive 1.0.0
INSTALL_DIR="$FIXTURE_DIR/legacy" VERSION=v1.0.1 bash "$PROJECT_DIR/install.sh" >/dev/null
test "$("$FIXTURE_DIR/legacy/webp2ascii")" = 1.0.0

# Reject corrupt archives before replacing an existing installation.
printf 'corrupted' >> "$FIXTURE_DIR/assets/webp2ascii-1.1.0-linux-amd64.tar.gz"
if INSTALL_DIR="$FIXTURE_DIR/pinned" VERSION=v1.1.0 bash "$PROJECT_DIR/install.sh" >"$FIXTURE_DIR/log" 2>&1; then
    echo 'Corrupt archive was accepted' >&2
    exit 1
fi
grep -q 'checksum verification failed' "$FIXTURE_DIR/log"
test "$("$FIXTURE_DIR/pinned/webp2ascii")" = 1.1.0
echo 'Installer tests passed (latest, pinned, legacy, checksum failure)'
