import { mkdirSync, mkdtempSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { dirname, join } from "node:path";

import { expect, test } from "vitest";

import { filled, siteBase, verify } from "./build";

const shell = "<title>kit tutorials</title><div id=\"root\"><!--page--></div>";

// dist writes the files, less those that are null, under a fresh directory and returns it.
function dist(files: Record<string, string | null>): string {
  const root = mkdtempSync(join(tmpdir(), "tutorials-dist-"));
  for (const [name, text] of Object.entries(files)) {
    if (text === null) {
      continue;
    }
    mkdirSync(dirname(join(root, name)), { recursive: true });
    writeFileSync(join(root, name), text);
  }
  return root;
}

test("verify passes a whole dist, and names each break", async () => {
  const page = (body = "") => `<link href="/b/assets/site.css"><div id="root"><article>${body}</article></div>`;
  const whole = {
    "index.html": page('<a href="/b/a/?kit=0.1.0#top">a</a>'),
    "a/index.html": page(),
    "assets/site.css": "",
  };
  const broken = async (change: Record<string, string | null>) =>
    (await verify(dist({ ...whole, ...change }), "/b/", ["", "a/"])).join("\n");
  expect(await broken({})).toBe("");
  expect(await broken({ "assets/site.css": null })).toContain('href="/b/assets/site.css" has no file');
  expect(await broken({ "a/index.html": '<div id="root"></div>' })).toContain("dist/a/index.html: an empty page");
  expect(await broken({ "index.html": null })).toContain("no dist/index.html");
  expect(await broken({ "a/index.html": page('<a href="/b/gone/">x</a>') })).toContain('href="/b/gone/" has no file');
  expect(await broken({ "a/index.html": page('<img src="/out.png">') })).toContain('src="/out.png" is outside /b/');
  expect(await broken({ "a/index.html": page("<a href='/b/gone/'>x</a>") })).toContain('href="/b/gone/" has no file');
  expect(await broken({ "a/index.html": page("<a href=/b/gone/>x</a>") })).toContain('href="/b/gone/" has no file');
  expect(await broken({ "a/index.html": page('<a href="/b/%zz">x</a>') })).toContain(
    'dist/a/index.html: href="/b/%zz" is not a URL',
  );
  expect(await broken({ "a/index.html": page('<a href="/b/gone/">x</a><a href="/b/gone/">y</a>') })).toBe(
    'dist/a/index.html: href="/b/gone/" has no file',
  );
  expect(await broken({ "a/index.html": page('<a href="/b/">home</a><a href="../../up">up</a>') })).toBe(
    'dist/a/index.html: href="../../up" is outside /b/',
  );
  expect(
    await broken({ "a/index.html": page('<a href="mailto:a@b.test">m</a><a href="//host.test/x">h</a>') }),
  ).toBe("");
});

test("siteBase gives a base its slashes, and refuses what is not a path", () => {
  expect([undefined, "/k/main", "k/main/", "//k//main//"].map(siteBase)).toEqual([
    "/",
    "/k/main/",
    "/k/main/",
    "/k/main/",
  ]);
  expect(["", "/"].map(siteBase)).toEqual(["/", "/"]);
  for (const bad of ["https://x.test/k/", "/k:1/", "/k?x", "/k#x", "/k m/"]) {
    expect(() => siteBase(bad)).toThrow("TUTORIALS_BASE is a path");
  }
});

test("filled keeps a page's dollar signs, and refuses an empty page", () => {
  const html = filled(shell, { path: "x/", title: "$&" }, "<article>echo $$ $& $'x'</article>");
  expect(html).toContain("<article>echo $$ $& $'x'</article>");
  expect(html).toContain("<title>$&amp; · kit tutorials</title>");
  expect(() => filled(shell, { path: "x/", title: "X" }, "")).toThrow("x/: the page rendered empty");
});
