import { cleanup, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { afterEach, expect, test } from "vitest";

import { PageLink, SiteLink } from "./reader";

afterEach(cleanup);

test("a page's link resolves against the page, the layout's against the base, and both keep ?kit=", () => {
  render(
    <MemoryRouter initialEntries={["/kit-tutorials/?kit=0.4.7"]}>
      <PageLink href="marks/">marks</PageLink>
      <PageLink href="../changelog/#0.4.7">changelog</PageLink>
      <PageLink href="https://example.test/">out</PageLink>
      <SiteLink href="whats-new/">new</SiteLink>
    </MemoryRouter>,
  );
  const href = (name: string) => screen.getByRole("link", { name }).getAttribute("href");
  expect(["marks", "changelog", "out", "new"].map(href)).toEqual([
    "/kit-tutorials/marks/?kit=0.4.7",
    "/changelog/?kit=0.4.7#0.4.7",
    "https://example.test/",
    "/whats-new/?kit=0.4.7",
  ]);
});
