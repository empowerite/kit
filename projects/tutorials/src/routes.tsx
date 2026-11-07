// The site's pages, one route for each MDX file, and the app that serves them.

import type { MDXComponents, MDXContent } from "mdx/types";
import { type ReactElement, type ReactNode, useEffect } from "react";
import { Route, Routes } from "react-router";

import { Since } from "./components/Since";
import { Layout } from "./Layout";
import { SiteLink } from "./reader";

// Frontmatter is a page's front matter; frontmatter.schema.json holds it to these fields.
export interface Frontmatter {
  title: string;
  top?: boolean;
  summary?: string;
  glyph?: string;
  audience?: string[];
  order?: number;
}

interface Page {
  default: MDXContent;
  frontmatter: Frontmatter;
}

// A route's path is relative to the site's base: "" is the home page.
export interface SiteRoute {
  path: string;
  title: string;
  element: ReactElement;
}

const pages = import.meta.glob<Page>(["./content/**/*.mdx", "./whats-new.mdx", "./changelog/*.mdx"], {
  eager: true,
});

const components: MDXComponents = { a: SiteLink, Since };

// contentPath gives the path a file under content/ is served at: <id>/index.mdx at <id>/, a top-level tutorial,
// and <id>/<page>.mdx at <id>/<page>/, a deeper page.
export function contentPath(file: string): string {
  return file
    .replace(/^\.\/content\//, "")
    .replace(/(^|\/)index\.mdx$/, "$1")
    .replace(/\.mdx$/, "/");
}

function Titled({ title, children }: { title: string; children: ReactNode }) {
  useEffect(() => {
    document.title = `${title} · kit tutorials`;
  }, [title]);
  return <article>{children}</article>;
}

function page(file: string): ReactElement {
  const { default: Content, frontmatter } = pages[file];
  return (
    <Titled title={frontmatter.title}>
      <h1>{frontmatter.title}</h1>
      <Content components={components} />
    </Titled>
  );
}

const tutorials = Object.keys(pages)
  .filter((file) => file.startsWith("./content/"))
  .map((file) => ({ file, path: contentPath(file), ...pages[file].frontmatter }));

// byOrder sorts tutorials as the catalogue does: by order (1000 when a page has none), then by path, which is its id.
export function byOrder(a: { order?: number; path: string }, b: { order?: number; path: string }): number {
  return (a.order ?? 1000) - (b.order ?? 1000) || (a.path < b.path ? -1 : a.path > b.path ? 1 : 0);
}

function Home() {
  const top = tutorials
    .filter((tutorial) => tutorial.top)
    .sort(byOrder);
  return (
    <Titled title="Tutorials">
      <h1>Tutorials</h1>
      <ul className="tutorials">
        {top.map((tutorial) => (
          <li key={tutorial.path}>
            <SiteLink href={tutorial.path}>
              <span className="glyph">{tutorial.glyph}</span> {tutorial.title}
            </SiteLink>
            <p>{tutorial.summary}</p>
          </li>
        ))}
      </ul>
    </Titled>
  );
}

// The changelog is one entry per release, changelog/<version>.mdx, newest first.
function Changelog() {
  const entries = Object.keys(pages)
    .filter((file) => file.startsWith("./changelog/"))
    .sort((a, b) => b.localeCompare(a, "en", { numeric: true }));
  return (
    <Titled title="Changelog">
      <h1>Changelog</h1>
      {entries.map((file) => {
        const { default: Content, frontmatter } = pages[file];
        return (
          <section key={file}>
            <h2>{frontmatter.title}</h2>
            <Content components={components} />
          </section>
        );
      })}
    </Titled>
  );
}

export const routes: SiteRoute[] = [
  { path: "", title: "Tutorials", element: <Home /> },
  ...tutorials.map((tutorial) => ({ path: tutorial.path, title: tutorial.title, element: page(tutorial.file) })),
  { path: "whats-new/", title: pages["./whats-new.mdx"].frontmatter.title, element: page("./whats-new.mdx") },
  { path: "changelog/", title: "Changelog", element: <Changelog /> },
];

export function App() {
  return (
    <Routes>
      <Route element={<Layout />}>
        {routes.map((route) => (
          <Route key={route.path} path={route.path} element={route.element} />
        ))}
      </Route>
    </Routes>
  );
}
