// The build's checks: the base a site is built for, a page put into the shell, and what dist/ holds once it is built.
// vite.config.ts is the only importer besides the tests.

import { readdir, readFile, stat } from "node:fs/promises";
import { join } from "node:path";

// siteBase returns the base path for a TUTORIALS_BASE value: "/" for nothing, else one leading and one trailing slash.
// A value that is a URL, or has a query, a fragment, or a space, is not a path and throws.
export function siteBase(value: string | undefined): string {
  if (!value) {
    return "/";
  }
  if (/[:?#\s]/.test(value)) {
    throw new Error(`TUTORIALS_BASE is a path, as /kit/tutorials/main/, not ${JSON.stringify(value)}`);
  }
  const segments = value.split("/").filter((segment) => segment !== "");
  return segments.length === 0 ? "/" : `/${segments.join("/")}/`;
}

function escape(text: string): string {
  return text.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;");
}

// filled returns the shell with the route's title and the rendered page put in. The replacements are functions, so a
// `$$`, `$&`, or `$'` in a page's text is not read as a replacement pattern. A page with no <article throws.
export function filled(shell: string, route: { path: string; title: string }, page: string): string {
  if (!page.includes("<article")) {
    throw new Error(`${route.path}: the page rendered empty`);
  }
  return shell
    .replace("<title>kit tutorials</title>", () => `<title>${escape(route.title)} · kit tutorials</title>`)
    .replace("<!--page-->", () => page);
}

async function isFile(path: string): Promise<boolean> {
  try {
    return (await stat(path)).isFile();
  } catch {
    return false;
  }
}

// verify returns the problems of dist/ built for the base, each once and naming its file: a route with no page or
// an empty one, and an href or src that is outside the base, is no URL, or names no file. A link is resolved against
// its page's address.
export async function verify(dist: string, base: string, paths: string[]): Promise<string[]> {
  const problems = new Set<string>();
  for (const path of paths) {
    const file = path.endsWith(".html") ? path : `${path}index.html`;
    if (!(await isFile(join(dist, file)))) {
      problems.add(`no dist/${file}`);
    } else if (!(await readFile(join(dist, file), "utf8")).includes("<article")) {
      problems.add(`dist/${file}: an empty page`);
    }
  }
  for (const file of await readdir(dist, { recursive: true })) {
    if (!file.endsWith(".html")) {
      continue;
    }
    const html = await readFile(join(dist, file), "utf8");
    const address = new URL(`${base}${file}`, "http://site.invalid");
    const attributes = /\s(href|src)=(?:"([^"]*)"|'([^']*)'|([^\s"'<>`=]+))/g;
    for (const [, attribute, quoted, single, bare] of html.matchAll(attributes)) {
      const url = quoted ?? single ?? bare;
      if (url.startsWith("#") || url.startsWith("//") || /^[a-z][a-z0-9+.-]*:/i.test(url)) {
        continue;
      }
      let target: string;
      try {
        target = decodeURIComponent(new URL(url, address).pathname);
      } catch {
        problems.add(`dist/${file}: ${attribute}="${url}" is not a URL`);
        continue;
      }
      if (!target.startsWith(base)) {
        problems.add(`dist/${file}: ${attribute}="${url}" is outside ${base}`);
        continue;
      }
      const name = target.slice(base.length);
      if (!(await isFile(join(dist, name === "" || name.endsWith("/") ? `${name}index.html` : name)))) {
        problems.add(`dist/${file}: ${attribute}="${url}" has no file`);
      }
    }
  }
  return [...problems];
}
