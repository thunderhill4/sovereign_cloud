# Own Your Cloud

*From Cloud Tenant to Infrastructure Owner — a sovereign-Kubernetes story told in chapters.*

A book for two readers: the executive who signs the cloud bill, and the engineer who would build the alternative. It follows a composite enterprise, **Meridian**, from a cloud invoice to a working sovereign platform — and checks every claim along the way against a real one.

**Status: second draft, September 2026.** The *Draft notes* sections have been stripped from the manuscript; every open question they held is collected in [`draft-notes-archive.md`](draft-notes-archive.md), and the manuscript that still carried them inline is preserved at the git tag `with-draft-notes`.

## How the book keeps score

- **Meridian** scenes are composite fiction.
- **Build log** sections come from the companion platform repository, [`sovereign_cloud`](https://github.com/thunderhill4/sovereign_cloud), and every number in them was measured on it — most of them again on 17 September 2026.
- **Public record** and **Briefing** sections quote named sources: published reports, or the strategy deck behind the book. Where a source gives no evidence for a number, the text says so.

Every chapter closes with **The ledger** (what was built, measured or found), **Ask your team** (three questions for a CxO), and **Open the repo** (the files and commands that reproduce the chapter).

## Contents

| | Title | Part |
|---|---|---|
| Prologue | The Invoice | |
| 1 | The Great Cloud ROI Myth | I — The Bill |
| 2 | The Landlord Problem | I — The Bill |
| 3 | The Sovereign Imperative | I — The Bill |
| 4 | Kubernetes That Runs Kubernetes | I — The Bill |
| 5 | The First Cluster | II — The Factory |
| 6 | Ten Minutes Becomes Two | II — The Factory |
| 7 | Thirty-Six Percent Before the Kernel | II — The Factory |
| 8 | Two Lines and a CPU Quota | II — The Factory |
| 9 | Warmth Is Mandatory | II — The Factory |
| 10 | Hardcoded IPs and Other Confessions | III — The Fabric |
| 11 | One Label | III — The Fabric |
| 12 | The Model Is Just Another Service | III — The Fabric |
| 13 | An Agent as a First-Class Object | IV — The Brain |
| 14 | Rules the Model Cannot Break | IV — The Brain |
| 15 | Trust, but Verify | IV — The Brain |
| 16 | The Fleet | IV — The Brain |
| 17 | The Same Cluster Twice | V — The Dividend |
| 18 | From the Lab to the Data Centre | V — The Dividend |
| 19 | The Day We Upgraded Everything | V — The Dividend |
| 20 | Reclaiming the Margin | V — The Dividend |
| Epilogue | Own Your Cloud | |
| Appendices | The Owner's Reference (A–H) | |

About 59,000 words of narrative, plus about 7,800 words of appendices.

## Repository layout

```
TABLE_OF_CONTENTS.md   the book's plan, with a blurb and sources for each chapter
chapters/              the manuscript, one Markdown file per chapter
figures/               figure SVGs, plus a brief for the ones not yet drawn
measurements/          raw data behind the measured claims, and how it was taken
draft-notes-archive.md every open question, stripped out of the manuscript
build-pdf.py           assembles the chapters into one print-styled HTML file
render-pdf.mjs          prints that HTML to PDF via headless Chrome
dist/                  build output (gitignored): book.html, Own-Your-Cloud.pdf
```

## Rendering

```bash
python3 build-pdf.py                  # chapters -> dist/book.html
npm install --no-save puppeteer-core  # once; drives the system Chrome, no download
node render-pdf.mjs                   # dist/book.html -> dist/Own-Your-Cloud.pdf
```

`build-pdf.py` (stdlib `markdown` only, no network) walks the chapters in reading
order — prologue, 1–20, epilogue, appendices — inlines each figure's SVG directly
(so the HTML is one self-contained file, no relative paths to resolve), and wraps
the result in a print stylesheet: 6×9in trim, `Linux Libertine` body text and
`Liberation Mono` code, both installed locally, so nothing is fetched from Google
Fonts or anywhere else. `render-pdf.mjs` drives the **system** Chrome via
`puppeteer-core` (`CHROME_PATH`, default `/usr/bin/google-chrome-stable`; no
Chromium download) and adds real page numbers via Chrome's own footer template.
Current output: 201 pages. No clickable outline/bookmarks yet — `qpdf` and
`pypdf` weren't available on the machine this was built on, and installing
either wasn't worth reaching for `pip --break-system-packages`; a later pass can
add them without touching the two scripts above.

## License

All rights reserved — see [`LICENSE`](LICENSE).
