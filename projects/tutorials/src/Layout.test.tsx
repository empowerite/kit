import { cleanup, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { afterEach, expect, test, vi } from "vitest";

import { Layout } from "./Layout";

afterEach(() => {
  cleanup();
  vi.unstubAllEnvs();
});

function navigation(line: string): string[] {
  vi.stubEnv("TUTORIALS_LINE", line);
  render(
    <MemoryRouter>
      <Layout />
    </MemoryRouter>,
  );
  return screen.getAllByRole("link").map((link) => link.textContent ?? "");
}

test("a minor line's navigation names its What's new and the Changelog", () => {
  expect(navigation("0.4")).toEqual(expect.arrayContaining(["What's new in 0.4", "Changelog"]));
});

test("main's navigation has What's new and the Changelog", () => {
  expect(navigation("main")).toEqual(expect.arrayContaining(["What's new", "Changelog"]));
});
