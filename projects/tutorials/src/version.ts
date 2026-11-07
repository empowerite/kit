// How a passage marked <Since version="…"> reads to the kit the reader has.

export type SinceState = "plain" | "met" | "newer";

const releaseVersion = /^(\d+)\.(\d+)\.(\d+)$/;
const anyVersion = /^(\d+)\.(\d+)\.(\d+)(-[0-9A-Za-z.-]+)?$/;

// sinceState says how a passage that arrived in `since` reads to a reader on kit `reader`: "newer" when the
// reader's kit is older than the passage, "met" when it has it, and "plain" when the page does not know the
// reader's kit: no ?kit=, `unreleased`, or anything that is not a version. A pre-release is older than its release.
// It throws when `since` is not X.Y.Z, so a typo in a page fails the build.
export function sinceState(since: string, reader: string | null): SinceState {
  const passage = releaseVersion.exec(since);
  if (!passage) {
    throw new Error(`<Since version="${since}">: the version must be X.Y.Z`);
  }
  const kit = reader === null ? null : anyVersion.exec(reader);
  if (!kit) {
    return "plain";
  }
  for (const part of [1, 2, 3]) {
    const difference = Number(kit[part]) - Number(passage[part]);
    if (difference !== 0) {
      return difference > 0 ? "met" : "newer";
    }
  }
  return kit[4] ? "newer" : "met";
}
