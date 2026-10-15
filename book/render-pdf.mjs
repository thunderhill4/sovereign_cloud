#!/usr/bin/env node
// render-pdf.mjs — print the assembled HTML (build-pdf.py's output) to a PDF
// with real page numbers, via the system Chrome (no network, no bundled
// Chromium download: puppeteer-core drives whatever CHROME_PATH points at).
//
// Usage: node render-pdf.mjs [in.html] [out.pdf]

import puppeteer from "puppeteer-core";
import { pathToFileURL } from "node:url";
import path from "node:path";

const inFile = path.resolve(process.argv[2] ?? "dist/book.html");
const outFile = path.resolve(process.argv[3] ?? "dist/Own-Your-Cloud.pdf");
const chromePath = process.env.CHROME_PATH ?? "/usr/bin/google-chrome-stable";

const browser = await puppeteer.launch({
  executablePath: chromePath,
  headless: true,
  args: ["--no-sandbox", "--disable-dev-shm-usage"],
});
try {
  const page = await browser.newPage();
  await page.goto(pathToFileURL(inFile).href, { waitUntil: "networkidle0" });
  await page.pdf({
    path: outFile,
    printBackground: true,
    preferCSSPageSize: true, // honour the @page size/margin rules in the CSS
    displayHeaderFooter: true,
    headerTemplate: "<span></span>", // blank; @page margin still reserves the space
    footerTemplate: `
      <div style="width:100%; font-family:'Liberation Serif',Georgia,serif;
                  font-size:8.5px; color:#737C86; text-align:center;
                  padding-top:2px;">
        <span class="pageNumber"></span>
      </div>`,
  });
} finally {
  await browser.close();
}
console.log(`wrote ${outFile}`);
