import { describe, expect, test, vi } from "vitest";

vi.mock("@/stores/config", () => ({
  default: { base: "" },
  withBase: (path: string) => path,
}));

import { selfUpdateHeadline } from "./selfUpdate";

describe("selfUpdateHeadline", () => {
  // The bug this replaced: a container on :master was offered the newest
  // release, which is not what pulling that tag would ever give it.
  test("the tag's answer wins over the release feed", () => {
    expect(selfUpdateHeadline("update-available", true)).toBe("image");
    expect(selfUpdateHeadline("update-available", false)).toBe("image");
    expect(selfUpdateHeadline("up-to-date", true)).toBe("current");
  });

  test("a release is named only when the tag cannot answer", () => {
    expect(selfUpdateHeadline("pinned", true)).toBe("release");
    expect(selfUpdateHeadline("skipped", true)).toBe("release");
    expect(selfUpdateHeadline(undefined, true)).toBe("release");
    expect(selfUpdateHeadline("pinned", false)).toBe("current");
    expect(selfUpdateHeadline(undefined, false)).toBe("current");
  });
});
