#!/usr/bin/env bash
# render-demo-video.sh — turn a scene transcript into an MP4.
#
# Reads a directory of NN.txt scene files (first line = title, rest = body),
# renders each as a terminal-styled frame with headless Chrome, and stitches
# them with ffmpeg. Scene hold time scales with how much text the scene has,
# so dense scenes stay on screen long enough to read.
#
# There is no asciinema/vhs/agg on this host, which is why frames are rendered
# from HTML rather than replayed from a terminal cast.
#
# Usage:
#   scripts/render-demo-video.sh <scene-dir> <output.mp4> [title]
#
# Example:
#   SCENE_DIR=/tmp/scenes ./06-sympozium/demo-mesh-sre.sh
#   scripts/render-demo-video.sh /tmp/scenes docs/demo/mesh-sre-agent-demo.mp4
set -euo pipefail

SCENE_DIR="${1:?usage: render-demo-video.sh <scene-dir> <output.mp4> [title]}"
OUT="${2:?usage: render-demo-video.sh <scene-dir> <output.mp4> [title]}"
TITLE="${3:-mesh-sre-agent · Sympozium}"

WIDTH="${WIDTH:-1600}"
HEIGHT="${HEIGHT:-900}"
# Seconds a scene holds = BASE + PER_LINE * lines, clamped to MAX.
BASE_HOLD="${BASE_HOLD:-2.5}"
PER_LINE="${PER_LINE:-0.42}"
MAX_HOLD="${MAX_HOLD:-13}"
FPS="${FPS:-25}"
# Characters per rendered line at the CSS font size — used to estimate how many
# on-screen lines a long paragraph wraps into.
WRAP_COLS="${WRAP_COLS:-95}"

CHROME="$(command -v google-chrome || command -v chromium || true)"
[[ -z "$CHROME" ]] && { echo "ERROR: no chrome/chromium found" >&2; exit 1; }
command -v ffmpeg >/dev/null || { echo "ERROR: ffmpeg not found" >&2; exit 1; }

shopt -s nullglob
SCENES=("$SCENE_DIR"/*.txt)
[[ ${#SCENES[@]} -eq 0 ]] && { echo "ERROR: no scenes in $SCENE_DIR" >&2; exit 1; }

WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT
mkdir -p "$(dirname "$OUT")"

echo "Rendering ${#SCENES[@]} scenes at ${WIDTH}x${HEIGHT}..."

# Build one HTML frame per scene. Python owns the escaping so arbitrary
# kubectl/agent output can't break out of the markup.
python3 - "$WORK" "$TITLE" "${SCENES[@]}" <<'PY'
import html, sys, pathlib

work = pathlib.Path(sys.argv[1]); deck_title = sys.argv[2]; scenes = sys.argv[3:]

CSS = """
*{margin:0;padding:0;box-sizing:border-box}
body{width:1600px;height:900px;background:#0d1117;overflow:hidden;
  font-family:'DejaVu Sans Mono','Liberation Mono',monospace}
.win{position:absolute;inset:38px;background:#161b22;border:1px solid #30363d;
  border-radius:10px;box-shadow:0 18px 60px rgba(0,0,0,.6);display:flex;flex-direction:column}
.bar{height:38px;background:#21262d;border-bottom:1px solid #30363d;
  border-radius:9px 9px 0 0;display:flex;align-items:center;padding:0 14px;gap:8px;flex:none}
.dot{width:11px;height:11px;border-radius:50%}
.r{background:#ff5f57}.y{background:#febc2e}.g{background:#28c840}
.wt{color:#8b949e;font-size:12.5px;margin-left:12px;letter-spacing:.3px}
.body{padding:26px 34px;flex:1;overflow:hidden;display:flex;flex-direction:column}
.n{color:#484f58;font-size:12px;letter-spacing:2.5px;text-transform:uppercase}
h1{color:#58a6ff;font-size:27px;font-weight:700;margin:7px 0 18px;letter-spacing:-.2px}
pre{color:#c9d1d9;font-size:15.5px;line-height:1.58;white-space:pre-wrap;
  word-break:break-word;flex:1}
.cmd{color:#7ee787;font-weight:700}
.q{color:#d2a8ff;font-weight:700}
.hl{color:#ffa657}
.foot{color:#484f58;font-size:11.5px;display:flex;justify-content:space-between;
  border-top:1px solid #21262d;padding-top:11px;margin-top:14px;flex:none}
"""

def render_line(raw):
    e = html.escape(raw)
    s = raw.lstrip()
    if s.startswith("$"):
        return f'<span class="cmd">{e}</span>'
    if s.startswith(">"):
        return f'<span class="q">{e}</span>'
    if s.startswith("·"):
        return f'<span class="hl">{e}</span>'
    return e

for i, path in enumerate(scenes, 1):
    lines = pathlib.Path(path).read_text().split("\n")
    title, body = lines[0], lines[1:]
    while body and not body[-1].strip():
        body.pop()
    inner = "\n".join(render_line(l) for l in body)
    (work / f"f{i:02d}.html").write_text(f"""<!doctype html><meta charset=utf-8>
<style>{CSS}</style><div class=win>
<div class=bar><div class="dot r"></div><div class="dot y"></div><div class="dot g"></div>
<div class=wt>{html.escape(deck_title)}</div></div>
<div class=body><div class=n>scene {i:02d} / {len(scenes):02d}</div>
<h1>{html.escape(title)}</h1><pre>{inner}</pre>
<div class=foot><span>KubeVirt multi-cluster platform · Istio 1.30 ambient</span>
<span>captured live</span></div></div></div>""")
    # Hold time is decided here, next to the text it is measured from.
    print(f"{i:02d}\t{len(body)}")
PY

# Screenshot each frame, computing its hold from the line count Python reported.
: > "$WORK/concat.txt"
i=0
for scene in "${SCENES[@]}"; do
    i=$((i + 1))
    f="$(printf 'f%02d' "$i")"
    "$CHROME" --headless --no-sandbox --disable-gpu --hide-scrollbars \
        --virtual-time-budget=2500 --window-size="${WIDTH},${HEIGHT}" \
        --screenshot="$WORK/$f.png" "file://$WORK/$f.html" >/dev/null 2>&1
    [[ -f "$WORK/$f.png" ]] || { echo "ERROR: frame $f failed to render" >&2; exit 1; }
    # Count RENDERED lines, not newlines: the <pre> soft-wraps, so a scene whose
    # body is three long paragraphs occupies far more than three lines on screen.
    # Counting newlines gave a 3.7s hold to scenes that need ~10s to read.
    read -r n_lines hold < <(python3 -c "
import math,sys
body=open(sys.argv[1]).read().split(chr(10))[1:]
cols=$WRAP_COLS
n=sum(max(1, math.ceil(len(l)/cols)) for l in body if l.strip()) or 1
print(n, min($MAX_HOLD, $BASE_HOLD + $PER_LINE*n))" "$scene")
    printf "file '%s'\nduration %s\n" "$WORK/$f.png" "$hold" >> "$WORK/concat.txt"
    printf '  scene %02d  %-5s lines  %5ss hold\n' "$i" "$n_lines" "$hold"
done
# concat demuxer ignores the final entry's duration unless the file is repeated.
printf "file '%s'\n" "$WORK/$(printf 'f%02d' "$i").png" >> "$WORK/concat.txt"

echo "Encoding $OUT..."
ffmpeg -y -loglevel error -f concat -safe 0 -i "$WORK/concat.txt" \
    -vf "fps=${FPS},format=yuv420p" -c:v libx264 -preset medium -crf 20 \
    -movflags +faststart "$OUT"

dur=$(ffprobe -v error -show_entries format=duration -of csv=p=0 "$OUT" 2>/dev/null || echo "?")
size=$(du -h "$OUT" | cut -f1)
echo "Done: $OUT  (${dur}s, $size)"
