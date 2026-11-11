import { cleanup, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { afterEach, expect, test } from "vitest";

import { Since } from "./Since";

afterEach(cleanup);

test("Since tells a reader whose kit is older which version the passage needs", () => {
  render(
    <MemoryRouter initialEntries={["/kit-tutorials/?kit=0.4.7"]}>
      <Since version="0.4.9">x</Since>
    </MemoryRouter>,
  );
  expect(screen.getByText("Needs kit 0.4.9; you have 0.4.7.")).toBeTruthy();
  expect(screen.getByText(/To upgrade, install kit 0\.4\.9 or later, then run/)).toBeTruthy();
  expect(screen.getByText("x")).toBeTruthy();
});

test("Since marks the passage plainly when the page does not know the reader's kit", () => {
  render(
    <MemoryRouter initialEntries={["/kit-tutorials/"]}>
      <Since version="0.4.9">x</Since>
    </MemoryRouter>,
  );
  expect(screen.getByText("Since 0.4.9")).toBeTruthy();
  expect(screen.getByText("x")).toBeTruthy();
  expect(screen.queryByText(/Needs/)).toBeNull();
});
