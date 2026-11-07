// The site's build: MDX pages in React, pre-rendered to dist/<path>/index.html for the base TUTORIALS_BASE names.

import { mkdir, readFile, rm, writeFile } from "node:fs/promises";
import { join, resolve } from "node:path";
import { pathToFileURL } from "node:url";

import mdx from "@mdx-js/rollup";
import react from "@vitejs/plugin-react";
import remarkFrontmatter from "remark-frontmatter";
import remarkMdxFrontmatter from "remark-mdx-frontmatter";
import { defineConfig } from "vitest/config";

const dist = resolve(import.meta.dirname, "dist");
const server = join(dist, ".server");

interface Server {
  routes: { path: string; title: string }[];
  render(path: string): string;
}

function escape(text: string): string {
  return text.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;");
}

// prerender writes each route's page into the shell the client build left in dist/index.html. A page that throws,
// as one with <Since version="0.4"> does, fails the build. The replacements are functions, so a `$$`, `$&`, or `$'` in
// a page's text is not read as a replacement pattern.
async function prerender(): Promise<void> {
  const shell = await readFile(join(dist, "index.html"), "utf8");
  const { routes, render }: Server = await import(pathToFileURL(join(server, "server.js")).href);
  for (const route of routes) {
    const html = shell
      .replace("<title>kit tutorials</title>", () => `<title>${escape(route.title)} · kit tutorials</title>`)
      .replace("<!--page-->", () => render(route.path));
    await mkdir(join(dist, route.path), { recursive: true });
    await writeFile(join(dist, route.path, "index.html"), html);
  }
  await rm(server, { recursive: true });
}

export default defineConfig({
  // A build is for one host's path, as /kit/tutorials/main/ or /tutorials/0.4/; `npm run dev` serves at /.
  base: process.env.TUTORIALS_BASE || "/",
  // TUTORIALS_LINE, the line of kit the site is for, reaches the pages as import.meta.env.TUTORIALS_LINE.
  envPrefix: "TUTORIALS_",
  plugins: [
    { enforce: "pre", ...mdx({ remarkPlugins: [remarkFrontmatter, remarkMdxFrontmatter] }) },
    react({ include: /\.(mdx|tsx|ts)$/ }),
  ],
  environments: {
    ssr: { build: { outDir: server, rolldownOptions: { input: "src/server.tsx" } } },
  },
  builder: {
    async buildApp(builder) {
      await builder.build(builder.environments.client);
      await builder.build(builder.environments.ssr);
      await prerender();
    },
  },
  test: { environment: "jsdom" },
});
