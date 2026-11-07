// verify-dist.mjs - checks that dist/ serves every page under src/content/ and links only inside its base
//
// A build is for one base, as /kit/tutorials/main/ or /tutorials/0.4/, and its links must work under that base
// alone. The script prints each problem, naming the file, and exits 1; it prints nothing when there is none.
//
//   node scripts/verify-dist.mjs <base>

import { access, readdir, readFile } from "node:fs/promises";
import { join, resolve } from "node:path";

const base = process.argv[2];
if (!base?.startsWith("/") || !base.endsWith("/")) {
  console.error("usage: node scripts/verify-dist.mjs <base>, a path that starts and ends with /");
  process.exit(2);
}

const site = resolve(import.meta.dirname, "..");
const problems = [];

// A top-level tutorial, <id>/index.mdx, is served at <id>/; a deeper page, <id>/<page>.mdx, at <id>/<page>/.
for (const file of await readdir(join(site, "src/content"), { recursive: true })) {
  if (!file.endsWith(".mdx")) {
    continue;
  }
  const page = join("dist", file.replace(/(^|\/)index\.mdx$/, "$1").replace(/\.mdx$/, "/"), "index.html");
  try {
    await access(join(site, page));
  } catch {
    problems.push(`src/content/${file}: no ${page}`);
  }
}

// A root-relative href or src outside the base breaks on every host but one.
for (const file of await readdir(join(site, "dist"), { recursive: true })) {
  if (!file.endsWith(".html")) {
    continue;
  }
  const html = await readFile(join(site, "dist", file), "utf8");
  for (const [, attribute, url] of html.matchAll(/\s(href|src)="(\/[^"]*)"/g)) {
    if (!url.startsWith("//") && !url.startsWith(base)) {
      problems.push(`dist/${file}: ${attribute}="${url}" is outside ${base}`);
    }
  }
}

// A page's text reaches the shell through a string replacement in the build; `$$`, `$&`, and `$'` in a page must survive
// it. React escapes & and ' in text, so dist/ holds them escaped.
try {
  const dollars = await readFile(join(site, "dist/kit-tutorials/dollar-signs/index.html"), "utf8");
  const expected = ["echo $$", "echo $&amp;", "echo $&#x27;x&#x27;"];
  for (const text of expected) {
    if (!dollars.includes(text)) {
      problems.push(`dist/kit-tutorials/dollar-signs/index.html: lost "${text}" in the build`);
    }
  }
} catch {
  // a missing page is reported above
}

if (problems.length > 0) {
  console.error(problems.join("\n"));
  process.exit(1);
}
