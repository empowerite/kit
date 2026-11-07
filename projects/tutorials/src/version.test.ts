import { expect, test } from "vitest";

import { sinceState } from "./version";

test("sinceState compares the reader's kit with the passage's version", () => {
  expect(sinceState("0.4.9", "0.4.7")).toBe("newer");
  expect(sinceState("0.4.9", "0.4.9")).toBe("met");
  expect(sinceState("0.4.9", "0.4.12")).toBe("met");
  expect(sinceState("0.4.9", null)).toBe("plain");
  expect(sinceState("0.4.9", "unreleased")).toBe("plain");
  expect(sinceState("0.4.9", "garbage")).toBe("plain");
  expect(sinceState("0.5.0", "0.5.0-rc.1")).toBe("newer");
  expect(() => sinceState("0.4", "0.4.7")).toThrow();
});
