// Every page's frame: the site's name and the top navigation, then the page.

import { Outlet } from "react-router";

import { SiteLink } from "./reader";

// line is the line of kit the site was built for: `main`, a minor line as `0.4`, or a pre-release as `0.5.0-rc.1`.
function line(): string {
  return import.meta.env.TUTORIALS_LINE || "main";
}

export function Layout() {
  const built = line();
  return (
    <>
      <header className="site">
        <SiteLink href="" className="home">
          🎓 kit tutorials
        </SiteLink>
        <nav>
          <SiteLink href="whats-new/">{built === "main" ? "What's new" : `What's new in ${built}`}</SiteLink>
          <SiteLink href="changelog/">Changelog</SiteLink>
        </nav>
      </header>
      <main>
        <Outlet />
      </main>
    </>
  );
}
