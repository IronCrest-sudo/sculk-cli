#!/usr/bin/env bash
# Records the showcase sessions used on the website.
#
# Everything here runs the real `sculk` binary against a throwaway world, so
# the GIFs are recordings of actual output rather than mock-ups.
#
#   ./record-showcase.sh all          record + render every showcase
#   ./record-showcase.sh init         just the init session
#   SCULK=/path/to/sculk ./record-showcase.sh all
#
# Output goes to website/public/showcase/. Needs asciinema and Pillow.
set -euo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
WEBSITE="$(dirname "$HERE")"
OUT_DIR="$WEBSITE/public/showcase"
CAST_DIR="$(mktemp -d)"
SCULK="${SCULK:-sculk}"
FPS="${FPS:-10}"

mkdir -p "$OUT_DIR"

# A throwaway world + config so a recording never touches the real one, and so
# the output is identical on every machine.
new_world() {
    local name="$1"
    local root="$CAST_DIR/$name"
    rm -rf "$root"
    mkdir -p "$root/world/datapacks/my_pack" "$root/cfg"

    export XDG_CONFIG_HOME="$root/cfg"
    export HOME="$root"
    export NO_COLOR=
    export TERM=xterm-256color

    # Pre-seed the config: the "creating a new base config" notice is a
    # first-run detail, not part of the story each showcase tells.
    "$SCULK" config author "IronCrest" >/dev/null 2>&1 || true
    "$SCULK" config doMerge true >/dev/null 2>&1 || true

    cd "$root/world/datapacks/my_pack"
}

# say <cmd>  - print a command as though it were typed
say() { printf '$ %s\n' "$1"; sleep 0.5; }
# sc <args...> - show "sculk <args>", then really run the configured binary
sc() { printf '$ sculk %s\n' "$*"; sleep 0.5; "$SCULK" "$@"; sleep 0.7; }
# head_of <file> [n] - show a short file listing
head_of() { say "cat $1"; head -n "${2:-12}" "$1"; sleep 0.7; }

session_init() {
    new_world init
    sc init --dp my_pack 26.2
    say "find . -type f | sort"
    find . -type f | sort
    sleep 1.0
}

session_libraries() {
    new_world libraries
    "$SCULK" init --dp my_pack 26.2 >/dev/null 2>&1
    sc add id-system
    head_of "data/minecraft/tags/function/load.json" 8
    sleep 1.0
}

session_track() {
    new_world track
    "$SCULK" init --dp my_pack 26.2 >/dev/null 2>&1
    "$SCULK" add id-system >/dev/null 2>&1
    sc list --installed
    head_of "libraries.json" 14
    sleep 1.0
}

record() {
    local name="$1"
    local cast="$CAST_DIR/$name.cast"
    local gif="$OUT_DIR/$name.gif"
    # The session is written to a file rather than inlined into `bash -c`,
    # because the helper bodies contain single quotes.
    local script="$CAST_DIR/$name.sh"

    {
        echo "SCULK=$(printf '%q' "$SCULK")"
        echo "CAST_DIR=$(printf '%q' "$CAST_DIR")"
        declare -f say sc head_of new_world
        declare -f "session_$name"
        echo "session_$name"
    } > "$script"

    echo "▶ recording $name ..."
    asciinema rec -q --cols 92 --rows 18 -c "bash $script" "$cast"

    echo "▶ rendering $name ..."
    python3 "$HERE/cast2gif.py" "$cast" "$gif" --fps "$FPS" --max-frames 80
}

case "${1:-all}" in
    init)      record init ;;
    libraries) record libraries ;;
    track)     record track ;;
    all)
        record init
        record libraries
        record track
        ;;
    *)
        echo "usage: $0 [all|init|libraries|track]" >&2
        exit 2
        ;;
esac

echo
echo "done. GIFs in ${OUT_DIR#$WEBSITE/}"
