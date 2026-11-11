// What the browser's entry does with the root: hydrate a pre-rendered page, render into dev's empty one, or leave it.

// entry says what to do with the page the browser holds. A page the build pre-rendered, at an address within the base,
// is hydrated. An empty root, as `npm run dev` serves, is rendered. A pre-rendered page at an address outside the
// base is the 404.html that Pages serves for another copy of the site, and it stands as built: its links are to the
// site it was built for.
export function entry(prerendered: boolean, pathname: string, base: string): "hydrate" | "render" | "leave" {
  if (!prerendered) {
    return "render";
  }
  return pathname.startsWith(base) ? "hydrate" : "leave";
}
