// <Since version="X.Y.Z">: a passage that arrived in a patch release, flagged for a reader whose kit is older.

import type { ReactNode } from "react";

import { useReaderKit } from "../reader";
import { sinceState } from "../version";

export function Since({ version, children }: { version: string; children: ReactNode }) {
  const kit = useReaderKit();
  const state = sinceState(version, kit);
  return (
    <div className={`since since-${state}`}>
      <p className="since-mark">{state === "newer" ? `Needs kit ${version}; you have ${kit}.` : `Since ${version}`}</p>
      {children}
    </div>
  );
}
