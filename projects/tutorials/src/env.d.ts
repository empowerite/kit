/// <reference types="vite/client" />

interface ImportMetaEnv {
  // The line of kit the site is built for; vite.config.ts exposes every TUTORIALS_ variable to the pages.
  readonly TUTORIALS_LINE?: string;
}
