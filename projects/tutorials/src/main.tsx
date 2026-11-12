// The browser's entry: it hydrates the pre-rendered page, or renders into the empty one `npm run dev` serves.

import "./site.css";

import { StrictMode } from "react";
import { createRoot, hydrateRoot } from "react-dom/client";
import { BrowserRouter } from "react-router";

import { entry } from "./entry";
import { App } from "./routes";

const root = document.getElementById("root")!;
const app = (
  <StrictMode>
    <BrowserRouter basename={import.meta.env.BASE_URL}>
      <App />
    </BrowserRouter>
  </StrictMode>
);
switch (entry(root.firstElementChild !== null, location.pathname, import.meta.env.BASE_URL)) {
  case "hydrate":
    hydrateRoot(root, app);
    break;
  case "render":
    createRoot(root).render(app);
    break;
}
