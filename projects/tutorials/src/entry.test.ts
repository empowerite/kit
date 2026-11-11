import { expect, test } from "vitest";

import { entry } from "./entry";

test("a built page hydrates, dev's empty root renders, and Pages' 404 for another copy stands as built", () => {
  expect(entry(true, "/kit/tutorials/main/x/", "/kit/tutorials/main/")).toBe("hydrate");
  expect(entry(false, "/", "/")).toBe("render");
  expect(entry(true, "/kit/tutorials/0.1/gone/", "/kit/tutorials/main/")).toBe("leave");
});
