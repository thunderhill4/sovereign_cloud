#!/usr/bin/env python3
"""build-pdf.py — assemble the book's markdown chapters into one styled HTML
file, ready to be printed to PDF (see render-pdf.mjs, which drives Chrome).

Usage: python3 build-pdf.py [--out dist/book.html]

No network access, no external fonts: everything renders from fonts already
installed on this host (Linux Libertine for body text, matching the serif
stack the figure SVGs already declare; Liberation Mono for code).
"""
import argparse
import html
import re
from pathlib import Path

import markdown

BOOK_DIR = Path(__file__).parent
CHAPTERS_DIR = BOOK_DIR / "chapters"
FIGURES_DIR = BOOK_DIR / "figures"

# Reading order: prologue, 1-20, epilogue, appendices. Matches book/README.md's
# own contents table and the chapters/ directory's own sort order.
CHAPTER_FILES = [
    "00-prologue-the-invoice.md",
    "01-the-great-cloud-roi-myth.md",
    "02-the-landlord-problem.md",
    "03-the-sovereign-imperative.md",
    "04-kubernetes-that-runs-kubernetes.md",
    "05-the-first-cluster.md",
    "06-ten-minutes-becomes-two.md",
    "07-thirty-six-percent-before-the-kernel.md",
    "08-two-lines-and-a-cpu-quota.md",
    "09-warmth-is-mandatory.md",
    "10-hardcoded-ips-and-other-confessions.md",
    "11-one-label.md",
    "12-the-model-is-just-another-service.md",
    "13-an-agent-as-a-first-class-object.md",
    "14-rules-the-model-cannot-break.md",
    "15-trust-but-verify.md",
    "16-the-fleet.md",
    "17-the-same-cluster-twice.md",
    "18-from-the-lab-to-the-data-centre.md",
    "19-the-day-we-upgraded-everything.md",
    "20-reclaiming-the-margin.md",
    "99-epilogue-own-your-cloud.md",
    "appendices.md",
]

IMG_RE = re.compile(r"!\[(?P<alt>[^\]]*)\]\((?P<src>\.\./figures/[^)]+)\)")


def inline_figures(md_text: str) -> str:
    """Replace ![alt](../figures/x.svg) with a <figure> holding the literal
    inline SVG, so the assembled HTML is self-contained (no relative paths to
    resolve, and Chrome print rasterizes vector art crisply either way)."""

    def repl(m: re.Match) -> str:
        alt = m.group("alt")
        src = FIGURES_DIR / Path(m.group("src")).name
        svg = src.read_text()
        # Drop the XML prolog if present; not valid inside an HTML fragment.
        svg = re.sub(r"^\s*<\?xml[^>]*\?>\s*", "", svg)
        return (
            f'\n<figure class="book-figure">{svg}'
            f"<figcaption>{html.escape(alt)}</figcaption></figure>\n"
        )

    return IMG_RE.sub(repl, md_text)


SCENE_RE = re.compile(r"^<p><em>(Meridian ·[^<]*)</em></p>$", re.MULTILINE)


def mark_scene_setters(chapter_html: str) -> str:
    """Meridian's scene-setter lines ("*Meridian · a Wednesday...*") are a
    single italic paragraph with no other markup to hook a style onto —
    give them a class so CSS can set them off from body text."""
    return SCENE_RE.sub(r'<p class="scene-setter">\1</p>', chapter_html)


def render_chapter(path: Path) -> str:
    raw = path.read_text()
    raw = inline_figures(raw)
    body = markdown.markdown(raw, extensions=["extra", "sane_lists"])
    body = mark_scene_setters(body)
    slug = path.stem
    return f'<article class="chapter" id="ch-{slug}">\n{body}\n</article>\n'


def build_toc_page(chapter_htmls: list[tuple[str, str]]) -> str:
    """chapter_htmls: list of (slug, first h1 text) already extracted."""
    items = "\n".join(
        f'<li><a href="#ch-{slug}">{html.escape(title)}</a></li>'
        for slug, title in chapter_htmls
    )
    return f"""
<section class="frontmatter toc-page">
  <h1>Contents</h1>
  <ol class="toc-list">
  {items}
  </ol>
</section>
"""


TITLE = "Own Your Cloud"
SUBTITLE = "From Cloud Tenant to Infrastructure Owner"
STRAPLINE = "A sovereign-Kubernetes story told in chapters"
STATUS = "Second draft · September 2026"

CSS = """
@page { size: 6in 9in; margin: 0.95in 0.75in 0.85in 0.75in; }
@page :first { margin: 0; }

:root {
  --ink:#1B2430; --ink2:#4A5563; --ink3:#737C86; --rule:#D6DAD3;
  --accent:#1F5F4A; --accent-soft:#E3EDE7;
}

* { box-sizing: border-box; }
html { -webkit-print-color-adjust: exact; print-color-adjust: exact; }

body {
  font-family: 'Linux Libertine O', 'Liberation Serif', Georgia, serif;
  font-size: 11.3pt;
  line-height: 1.5;
  color: var(--ink);
  margin: 0;
  hyphens: auto;
  -webkit-hyphens: auto;
  text-rendering: optimizeLegibility;
}

p { margin: 0 0 0.85em 0; text-align: justify; orphans: 3; widows: 3; }

code, pre, kbd {
  font-family: 'Liberation Mono', 'DejaVu Sans Mono', monospace;
  font-size: 0.85em;
}
code { background: var(--accent-soft); padding: 0.05em 0.3em; border-radius: 3px; }
pre {
  background: #f4f4f2; border: 1px solid var(--rule); border-radius: 4px;
  padding: 0.7em 0.9em; overflow-wrap: break-word; white-space: pre-wrap;
  page-break-inside: avoid;
}
pre code { background: none; padding: 0; }

blockquote {
  margin: 1.1em 0.2em; padding: 0 0 0 0.9em;
  border-left: 2.5px solid var(--accent);
  font-style: italic; color: var(--ink2);
  page-break-inside: avoid;
}
blockquote p { text-align: left; margin: 0 0 0.3em 0; }
blockquote p:last-child { text-align: right; font-style: normal; font-size: 0.92em; }

hr {
  border: none; text-align: center; margin: 1.6em 0;
  color: var(--ink3);
}
hr::before { content: "*  *  *"; letter-spacing: 0.4em; }

table { border-collapse: collapse; width: 100%; margin: 1em 0; font-size: 0.92em; }
th, td { border: 1px solid var(--rule); padding: 0.35em 0.6em; text-align: left; vertical-align: top; }
th { background: var(--accent-soft); font-weight: 700; }
tr { page-break-inside: avoid; }

a { color: var(--accent); text-decoration: none; }

ul, ol { margin: 0 0 0.85em 1.3em; padding: 0; }
li { margin-bottom: 0.3em; }

.scene-setter {
  text-align: center; font-style: italic; color: var(--ink3);
  letter-spacing: 0.02em; margin: 1.3em 0 1.1em 0;
}

figure.book-figure {
  margin: 1.3em 0; text-align: center; page-break-inside: avoid;
}
figure.book-figure svg { max-width: 100%; height: auto; }
figure.book-figure figcaption {
  font-size: 0.82em; color: var(--ink3); font-style: italic; margin-top: 0.4em;
}

/* ── Title page ── */
.titlepage {
  height: 9in; display: flex; flex-direction: column; align-items: center;
  justify-content: center; text-align: center; page-break-after: always;
  padding: 0 0.9in;
}
.titlepage .kicker {
  font-family: 'Liberation Mono', monospace; font-size: 0.8em;
  letter-spacing: 0.25em; text-transform: uppercase; color: var(--accent);
  margin-bottom: 1.4em;
}
.titlepage h1 {
  font-size: 2.55em; margin: 0 0 0.35em 0; line-height: 1.1;
  border: none; padding: 0;
}
.titlepage .subtitle { font-size: 1.2em; color: var(--ink2); margin-bottom: 0.3em; }
.titlepage .strapline {
  font-style: italic; color: var(--ink3); margin-top: 0.6em; font-size: 1em;
}
.titlepage .status {
  margin-top: 3.5em; font-size: 0.85em; color: var(--ink3);
  font-family: 'Liberation Mono', monospace;
}

/* ── Front matter / TOC ── */
.frontmatter { page-break-after: always; }
.toc-page h1 { text-align: center; }
.toc-list { list-style: none; margin: 2em 0 0 0; padding: 0; columns: 1; }
.toc-list li {
  padding: 0.35em 0; border-bottom: 1px dotted var(--rule);
  font-size: 0.98em;
}
.toc-list a { color: var(--ink); }

/* ── Chapters ── */
article.chapter { page-break-before: always; }
article.chapter > h1:first-child {
  font-size: 1.55em; margin: 0 0 0.15em 0; line-height: 1.25;
  border-bottom: 1.5px solid var(--accent); padding-bottom: 0.35em;
}
h1 { font-size: 1.5em; margin: 1.4em 0 0.5em 0; }
h2 { font-size: 1.2em; margin: 1.3em 0 0.45em 0; color: var(--ink); }
h3 { font-size: 1.02em; margin: 1.1em 0 0.4em 0; font-style: italic; }

.backmatter-title {
  height: 9in; display: flex; align-items: center; justify-content: center;
  text-align: center; page-break-after: always;
}
.backmatter-title h1 { font-size: 2em; }
"""


def main() -> None:
    ap = argparse.ArgumentParser()
    ap.add_argument("--out", default=str(BOOK_DIR / "dist" / "book.html"))
    args = ap.parse_args()

    chapters_html = []
    toc_entries = []
    h1_re = re.compile(r"<h1[^>]*>(.*?)</h1>")
    for fname in CHAPTER_FILES:
        path = CHAPTERS_DIR / fname
        if not path.exists():
            raise SystemExit(f"missing chapter file: {path}")
        chapter_html = render_chapter(path)
        chapters_html.append(chapter_html)
        m = h1_re.search(chapter_html)
        title = re.sub(r"<[^>]+>", "", m.group(1)) if m else fname
        toc_entries.append((path.stem, title))

    titlepage = f"""
<section class="titlepage">
  <div class="kicker">A Sovereign-Kubernetes Story</div>
  <h1>{html.escape(TITLE)}</h1>
  <div class="subtitle">{html.escape(SUBTITLE)}</div>
  <div class="strapline">{html.escape(STRAPLINE)}</div>
  <div class="status">{html.escape(STATUS)}</div>
</section>
"""
    toc_page = build_toc_page(toc_entries)
    backmatter = """
<section class="backmatter-title">
  <h1>End of manuscript</h1>
</section>
"""

    doc = f"""<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>{html.escape(TITLE)}</title>
<style>{CSS}</style>
</head>
<body>
{titlepage}
{toc_page}
{''.join(chapters_html)}
{backmatter}
</body>
</html>
"""

    out_path = Path(args.out)
    out_path.parent.mkdir(parents=True, exist_ok=True)
    out_path.write_text(doc)
    print(f"wrote {out_path} ({len(doc):,} bytes, {len(CHAPTER_FILES)} chapters)")


if __name__ == "__main__":
    main()
