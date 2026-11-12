// The build's entry for pre-rendering: vite.config.ts renders every route to dist/<path>/index.html with it.

import { StrictMode } from "react";
import { renderToString } from "react-dom/server";
import { StaticRouter } from "react-router";

import { App, routes } from "./routes";

export { routes };

// render returns the HTML of the page at `path`, relative to the site's base.
export function render(path: string): string {
  const base = import.meta.env.BASE_URL;
  return renderToString(
    <StrictMode>
      <StaticRouter basename={base} location={`${base}${path}`}>
        <App />
      </StaticRouter>
    </StrictMode>,
  );
}
