// The reader's kit, from the ?kit=<version> that kit puts in every link it opens, and the links that keep it.

import { type ComponentPropsWithoutRef, useSyncExternalStore } from "react";
import { Link, useLocation } from "react-router";

const unchanging = () => () => {};

// useReaderKit returns the version in the page's ?kit=, or null. A pre-rendered page cannot know it, so while React
// hydrates that page it returns null, as the build did, and then what the address says.
export function useReaderKit(): string | null {
  const { search } = useLocation();
  const inBrowser = useSyncExternalStore(
    unchanging,
    () => true,
    () => false,
  );
  return inBrowser ? new URLSearchParams(search).get("kit") : null;
}

const external = /^([a-z][a-z0-9+.-]*:|\/|#)/i;

// linked renders `href`, resolved against `from`, a path within the router's base, as a router link that carries the
// reader's ?kit= to the next page. An href with a scheme, a fragment alone, or a leading / is left as written;
// scripts/verify-dist.mjs fails a build whose root-relative href is outside the base.
function linked(href: string, from: string, kit: string | null, rest: ComponentPropsWithoutRef<"a">) {
  if (external.test(href)) {
    return <a href={href} {...rest} />;
  }
  const url = new URL(href, `https://site.invalid${from}`);
  if (kit !== null) {
    url.searchParams.set("kit", kit);
  }
  return <Link to={{ pathname: url.pathname, search: url.search, hash: url.hash }} {...rest} />;
}

// SiteLink is a link whose href is relative to the site's base, as `changelog/` or `kit-tutorials/marks/`: the
// layout's and the home page's.
export function SiteLink({ href = "", ...rest }: ComponentPropsWithoutRef<"a">) {
  return linked(href, "/", useReaderKit(), rest);
}

// PageLink is MDX's `a`: a link whose relative href resolves against the page it is on, as on the web, so
// `marks/` from `kit-tutorials/` is `kit-tutorials/marks/` and `../changelog/` is `changelog/`.
export function PageLink({ href = "", ...rest }: ComponentPropsWithoutRef<"a">) {
  const { pathname } = useLocation();
  return linked(href, pathname, useReaderKit(), rest);
}
