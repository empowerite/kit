import { cleanup, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { afterEach, expect, test } from "vitest";

import { App, byOrder } from "./routes";

afterEach(cleanup);

test("tutorials sort by order, then by path rather than title, as the catalogue does", () => {
  const tutorials = [
    { path: "b/", title: "A first title", order: 2 },
    { path: "z/", title: "Alpha" },
    { path: "a/", title: "Zulu", order: 2 },
    { path: "c/", title: "Middle", order: 1 },
  ];
  expect(tutorials.sort(byOrder).map((tutorial) => tutorial.path)).toEqual(["c/", "a/", "b/", "z/"]);
});

test("an address that is no page says so", () => {
  render(
    <MemoryRouter initialEntries={["/nonesuch/?kit=0.4.7"]}>
      <App />
    </MemoryRouter>,
  );
  expect(screen.getByRole("heading", { name: "No such page" })).toBeTruthy();
  expect(screen.getByRole("link", { name: "All tutorials" }).getAttribute("href")).toBe("/?kit=0.4.7");
});
