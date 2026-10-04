#!/usr/bin/env python3
"""Render an asciicast v2 recording into an animated GIF.

asciinema records the session, this turns it into something that can sit on
the website:

    asciinema rec -q --cols 96 --rows 20 -c ./session.sh out.cast
    python3 cast2gif.py out.cast out.gif

Why not `agg`: it is a separate Rust binary that is awkward to pin in CI, and
we only need a fixed-width text grid. Pillow, which is already a dependency of
the recording workflow, is enough.

Usage:
    cast2gif.py IN.cast OUT.gif [--fps 10] [--max-frames 90]
                                [--font PATH] [--font-size 14]
                                [--bg '#0b0a12'] [--idle 0.6]
"""

from __future__ import annotations

import argparse
import io
import json
import re
import sys
from pathlib import Path

from PIL import Image, ImageDraw, ImageFont

DEFAULT_FONT = "/usr/share/fonts/truetype/dejavu/DejaVuSansMono.ttf"
EMOJI_FONT = "/usr/share/fonts/truetype/noto/NotoColorEmoji.ttf"

# sculk prefixes every log line with "YYYY-MM-DD HH:MM:SS ". It is accurate in
# a terminal but it is noise (and a spoiler for the recording date) in a
# showcase, so it is stripped on the way in.
TIMESTAMP = re.compile(r"^\d{4}[-/]\d{2}[-/]\d{2} \d{2}:\d{2}:\d{2} ")

# Minimal palette for the syntax colouring pass. The CLI itself writes plain
# text, so the colour is applied here by looking at the start of each line.
C_PROMPT = (126, 231, 235)   # a typed command
C_OK = (134, 227, 166)       # success markers
C_WARN = (240, 200, 120)     # warnings
C_DIM = (140, 140, 160)      # everything else
C_TEXT = (226, 226, 236)

OK_MARKERS = ("✅", "🎉", "🍀", "⚙", "📩", "📦", "🚧", "📂", "🎯", "🔍", "📥", "📄", "🗑", "🚮")
WARN_MARKERS = ("⚠", "⛔", "🚫", "ℹ")


def parse_cast(path: Path):
    """Return (header, events) from an asciicast v2 file."""
    with path.open(encoding="utf-8") as fh:
        lines = [ln for ln in fh if ln.strip()]
    if not lines:
        raise SystemExit(f"{path} is empty")

    header = json.loads(lines[0])
    events = []
    for ln in lines[1:]:
        try:
            at, kind, data = json.loads(ln)
        except json.JSONDecodeError:
            continue
        if kind == "o":
            events.append((float(at), data))
    return header, events


class Screen:
    """A fixed-size text grid with the little bit of terminal behaviour that
    line-oriented CLI output actually needs."""

    def __init__(self, cols: int, rows: int):
        self.cols = cols
        self.rows = rows
        self.lines: list[str] = [""]
        self.col = 0

    def write(self, data: str) -> None:
        for ch in data:
            if ch == "\n":
                self.lines.append("")
                self.col = 0
            elif ch == "\r":
                self.col = 0
            elif ch == "\b":
                self.col = max(0, self.col - 1)
            elif ch == "\t":
                self.col = min(self.cols, self.col + 8 - (self.col % 8))
            elif ord(ch) < 32:
                continue  # ignore other control characters
            else:
                line = self.lines[-1]
                if self.col < len(line):
                    line = line[: self.col] + ch + line[self.col + 1 :]
                else:
                    line = line.ljust(self.col) + ch
                self.lines[-1] = line
                self.col += 1
                if self.col >= self.cols:
                    self.lines.append("")
                    self.col = 0
        # keep only the visible window
        if len(self.lines) > self.rows:
            del self.lines[: len(self.lines) - self.rows]

    def visible(self) -> list[str]:
        out = [TIMESTAMP.sub("", ln.rstrip()) for ln in self.lines]
        while out and out[-1] == "":
            out.pop()
        return out

    def snapshot(self) -> tuple[str, ...]:
        return tuple(self.visible())


def draw_line(d, img, x, y, line, font, char_w, line_h):
    """Draw one line, falling back to colour emoji glyphs for characters the
    monospace font cannot render."""
    colour = colour_for(line)
    cx = x
    size = int(line_h * 0.92)
    for ch in line:
        if is_emoji(ch):
            e = emoji_image(ch, size)
            if e is not None:
                img.paste(e, (int(cx), y + (line_h - size) // 2 + 1), e)
                cx += size  # emoji are full-width
                continue
            # no glyph available: skip the character rather than draw a box
        d.text((cx, y), ch, font=font, fill=colour)
        cx += char_w


def colour_for(line: str):
    if line.startswith("$ "):
        return C_PROMPT
    stripped = TIMESTAMP.sub("", line).strip()
    if any(stripped.startswith(m) or m in stripped[:6] for m in WARN_MARKERS):
        return C_WARN
    if any(stripped.startswith(m) or m in stripped[:6] for m in OK_MARKERS):
        return C_OK
    if stripped == "":
        return C_DIM
    return C_TEXT


# --- emoji rendering -----------------------------------------------------
# The CLI writes real unicode emoji, which the terminal shows in colour. Pillow
# cannot rasterise NotoColorEmoji (it is a fixed-size CBDT bitmap font), so we
# pull the embedded PNGs straight out of the font with fontTools and paste them.
_EMOJI: dict[str, Image.Image] = {}
_EMOJI_CMAP: dict[int, str] | None = None
_EMOJI_STRIKE: dict | None = None


def _emoji_tables():
    global _EMOJI_CMAP, _EMOJI_STRIKE
    if _EMOJI_CMAP is None:
        try:
            from fontTools.ttLib import TTFont

            font = TTFont(EMOJI_FONT)
            _EMOJI_CMAP = font.getBestCmap() or {}
            _EMOJI_STRIKE = font["CBDT"].strikeData[0]
        except Exception:
            _EMOJI_CMAP = {}
            _EMOJI_STRIKE = {}
    return _EMOJI_CMAP, _EMOJI_STRIKE


def is_emoji(ch: str) -> bool:
    o = ord(ch)
    return (
        0x1F000 <= o <= 0x1FAFF  # emoticons, symbols & pictographs
        or 0x2600 <= o <= 0x27BF  # misc symbols, dingbats
        or 0xFE00 <= o <= 0xFE0F  # variation selectors
        or 0x2B00 <= o <= 0x2BFF
        or 0x2190 <= o <= 0x21FF
    )


def emoji_image(ch: str, size: int) -> Image.Image | None:
    key = f"{ch}@{size}"
    if key in _EMOJI:
        return _EMOJI[key]
    cmap, strike = _emoji_tables()
    name = cmap.get(ord(ch))
    glyph = strike.get(name) if name else None
    if glyph is None:
        _EMOJI[key] = None
        return None
    try:
        glyph.ensureDecompiled()
        raw = getattr(glyph, "imageData", None) or getattr(glyph, "data", None)
        if raw is None:
            _EMOJI[key] = None
            return None
        img = Image.open(io.BytesIO(raw)).convert("RGBA")
    except Exception:
        _EMOJI[key] = None
        return None
    img = img.resize((size, size), Image.LANCZOS)
    _EMOJI[key] = img
    return img


def render_frames(args, header, events):
    """Replay the cast and return (frames, durations_ms, cols).

    Each entry is a *distinct* screen state together with how long the real
    session spent showing it, so the GIF plays back at the recorded pace
    instead of racing through at the sampling rate.
    """
    cols = int(header.get("width", args.cols))
    rows = int(header.get("height", args.rows))

    screen = Screen(cols, rows)

    # Boundaries are the event times themselves: the screen can only change
    # when something is written, so sampling any finer would only add
    # duplicates.
    boundaries = [0.0] + [at for at, _ in events]
    total = (events[-1][0] if events else 0.0) + args.idle
    boundaries.append(total)

    frames: list[tuple[str, ...]] = []
    durations: list[int] = []
    idx = 0

    for i, at in enumerate(boundaries):
        while idx < len(events) and events[idx][0] <= at:
            screen.write(events[idx][1])
            idx += 1

        snap = screen.snapshot()
        held = boundaries[i + 1] - at if i + 1 < len(boundaries) else 0.0

        if frames and frames[-1] == snap:
            # same picture, keep accumulating its real dwell time
            durations[-1] += max(0, int(held * 1000))
            continue

        frames.append(snap)
        durations.append(max(args.min_frame_ms, int(held * 1000)))

    # Collapse the longest holds so a slow session does not become a slideshow,
    # but never below the time the viewer needs to read the line.
    durations = [min(d, args.max_frame_ms) for d in durations]

    # rest on the final state
    if durations:
        durations[-1] = max(durations[-1], args.hold_last_ms)

    # a hard cap keeps pathological recordings from producing huge GIFs
    if args.max_frames and len(frames) > args.max_frames:
        step = len(frames) / args.max_frames
        kept = [frames[int(i * step)] for i in range(args.max_frames)]
        kept_d = [durations[int(i * step)] for i in range(args.max_frames)]
        frames, durations = kept, kept_d

    return frames, durations, cols


def build_gif(frames, durations, cols, args, out: Path) -> None:
    font = ImageFont.truetype(args.font, args.font_size)

    pad_x, pad_y = 18, 16
    line_h = int(args.font_size * 1.45)

    # measure a wide glyph to size the canvas
    probe = Image.new("RGB", (10, 10))
    draw = ImageDraw.Draw(probe)
    char_w = draw.textlength("M", font=font)
    width = int(char_w * cols) + pad_x * 2
    max_lines = max((len(f) for f in frames), default=1)
    height = line_h * max(max_lines, args.rows // 2) + pad_y * 2

    bg = tuple(int(args.bg[i : i + 2], 16) for i in (1, 3, 5))

    images = []
    for frame in frames:
        img = Image.new("RGB", (width, height), bg)
        d = ImageDraw.Draw(img)

        # window chrome, so it reads as a terminal on the page
        for i, dot in enumerate([(255, 95, 86), (255, 189, 46), (39, 201, 63)]):
            cx = pad_x + 6 + i * 20
            d.ellipse([cx - 5, pad_y - 8, cx + 5, pad_y + 2], fill=dot)

        y = pad_y + 10
        for line in frame:
            draw_line(d, img, pad_x, y, line, font, char_w, line_h)
            y += line_h
        images.append(img)

    if not images:
        raise SystemExit("no frames rendered")

    # quantise to a shared palette so the GIF is smaller and does not flicker
    palette = images[0].quantize(colors=64, method=Image.Quantize.MEDIANCUT)
    quantised = [im.quantize(colors=64, method=Image.Quantize.MEDIANCUT, palette=palette) for im in images]

    quantised[0].save(
        out,
        save_all=True,
        append_images=quantised[1:],
        duration=durations[: len(quantised)],
        loop=0,
        optimize=True,
        disposal=2,
    )


def main() -> int:
    p = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    p.add_argument("cast", type=Path)
    p.add_argument("out", type=Path)
    p.add_argument("--fps", type=int, default=10, help="kept for CLI compatibility; frames are event-driven")
    p.add_argument("--max-frames", type=int, default=120)
    p.add_argument("--min-frame-ms", type=int, default=120, help="floor for how long any state is shown")
    p.add_argument("--max-frame-ms", type=int, default=1100, help="ceiling, so long pauses do not stall the clip")
    p.add_argument("--hold-last-ms", type=int, default=2200, help="how long the final state is held")
    p.add_argument("--font", default=DEFAULT_FONT)
    p.add_argument("--font-size", type=int, default=14)
    p.add_argument("--bg", default="#0b0a12")
    p.add_argument("--idle", type=float, default=0.6, help="seconds of padding after the last event")
    p.add_argument("--cols", type=int, default=96, help="fallback width if the cast has none")
    p.add_argument("--rows", type=int, default=20, help="fallback height if the cast has none")
    args = p.parse_args()

    header, events = parse_cast(args.cast)
    if not events:
        raise SystemExit(f"{args.cast} contains no output events")

    frames, durations, cols = render_frames(args, header, events)
    build_gif(frames, durations, cols, args, args.out)

    size_kb = args.out.stat().st_size / 1024
    print(f"{args.out}: {len(frames)} frames, {sum(durations) / 1000:.1f}s, {size_kb:.0f} KiB")
    return 0


if __name__ == "__main__":
    sys.exit(main())
