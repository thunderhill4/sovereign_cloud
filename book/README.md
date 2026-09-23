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
html/                  rendered pages (html/index.html is the table of contents)
tools/build_chapter.py renders one chapter to a styled HTML page
tools/build_all.sh     renders every chapter with its part label
```

## Rendering

```bash
pip install markdown
./tools/build_all.sh          # one styled HTML page per chapter, in html/
./tools/build_pdf.sh          # the whole book as a single PDF
```

Each chapter page is self-contained HTML with light and dark themes; fonts load from Google Fonts.

`build_pdf.sh` assembles every chapter into one print-styled document (figures inlined as data URIs, so it is a single self-contained file), renders it with headless Chrome, then uses Ghostscript to add page numbers and nested bookmarks — Chrome implements neither. The result is **`own-your-cloud.pdf`**: A4, 195 pages, a clickable contents page, 27 bookmarks, and vector figures. It needs `google-chrome` or `chromium`; `gs` and `poppler-utils` are optional and only affect page numbers and bookmarks.

## License

All rights reserved — see [`LICENSE`](LICENSE).
