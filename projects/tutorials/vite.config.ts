// The site's build: MDX pages in React, pre-rendered to dist/<path>/index.html for the base TUTORIALS_BASE names.

import { mkdir, readFile, rm, writeFile } from "node:fs/promises";
import { join, resolve } from "node:path";
import { pathToFileURL } from "node:url";

import mdx from "@mdx-js/rollup";
import react from "@vitejs/plugin-react";
import remarkFrontmatter from "remark-frontmatter";
import remarkMdxFrontmatter from "remark-mdx-frontmatter";
import { defineConfig } from "vitest/config";

import { filled, siteBase, verify } from "./src/build";

const dist = resolve(import.meta.dirname, "dist");
const server = join(dist, ".server");

interface Server {
  routes: { path: string; title: string }[];
  render(path: string): string;
}

// prerender writes each route's page into the shell the client build left in dist/index.html, then checks dist/. A
// page that throws, as one with <Since version="0.4"> does, fails the build under its route's path, and so do the
// problems verify finds.
async function prerender(base: string): Promise<void> {
  const shell = await readFile(join(dist, "index.html"), "utf8");
  const { routes, render }: Server = await import(pathToFileURL(join(server, "server.js")).href);
  for (const route of routes) {
    let page: string;
    try {
      page = render(route.path);
    } catch (error) {
      throw new Error(`${route.path}: ${error instanceof Error ? error.message : String(error)}`, { cause: error });
    }
    await mkdir(join(dist, route.path), { recursive: true });
    await writeFile(join(dist, route.path, "index.html"), filled(shell, route, page));
  }
  await rm(server, { recursive: true });
  const problems = await verify(dist, base, routes.map((route) => route.path));
  if (problems.length > 0) {
    throw new Error(problems.join("\n"));
  }
}

const base = siteBase(process.env.TUTORIALS_BASE);

export default defineConfig({
  // A build is for one host's path, as /kit/tutorials/main/ or /tutorials/0.4/; `npm run dev` serves at /.
  base,
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
      await prerender(base);
    },
  },
  test: { environment: "jsdom" },
});
