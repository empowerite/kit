import { expect, test } from "vitest";

import { byOrder } from "./routes";

test("tutorials sort by order, then by path rather than title, as the catalogue does", () => {
  const tutorials = [
    { path: "b/", title: "A first title", order: 2 },
    { path: "z/", title: "Alpha" },
    { path: "a/", title: "Zulu", order: 2 },
    { path: "c/", title: "Middle", order: 1 },
  ];
  expect(tutorials.sort(byOrder).map((tutorial) => tutorial.path)).toEqual(["c/", "a/", "b/", "z/"]);
});
