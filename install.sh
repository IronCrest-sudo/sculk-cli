#!/usr/bin/env sh
# Builds sculk and installs it as `sculk` on your PATH.
# The binary is built in a temp directory, so nothing ends up in the repository.
set -eu

cd "$(dirname "$0")"

if ! command -v go >/dev/null 2>&1; then
	echo "Go is not installed. Get it from https://go.dev/dl/ (version listed in go.mod)." >&2
	exit 1
fi

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

echo "Building sculk ..."
go build -o "$tmp/sculk" .

# Prefer a system-wide location, fall back to a per-user one.
sudo=""
if [ -w /usr/local/bin ]; then
	dest=/usr/local/bin
elif command -v sudo >/dev/null 2>&1 && sudo -n true 2>/dev/null; then
	dest=/usr/local/bin
	sudo="sudo"
else
	dest="$HOME/.local/bin"
	mkdir -p "$dest"
fi

$sudo install -m 0755 "$tmp/sculk" "$dest/sculk"
echo "Installed: $dest/sculk"

case ":$PATH:" in
*":$dest:"*) ;;
*)
	echo
	echo "$dest is not on your PATH yet. Add this line to ~/.bashrc (or ~/.zshrc) and reopen the terminal:"
	echo "  export PATH=\"$dest:\$PATH\""
	;;
esac

echo
echo "Done. Try: sculk --help"
