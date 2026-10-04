import { describe, expect, test } from "vitest";
import { awaitingVerdict, imageTag, shortDigest, updateLabels } from "./ContainerUpdate";

describe("imageTag", () => {
  test.each([
    ["ghcr.io/immich-app/immich-server:1.4.2", "1.4.2"],
    ["localhost:5000/app", "latest"],
    ["nginx", "latest"],
    ["nginx:1.27@sha256:abc", "1.27"],
    ["", ""],
  ])("%s", (ref, tag) => expect(imageTag(ref)).toBe(tag));
});

describe("shortDigest", () => {
  test("drops the repository and the algorithm", () => {
    expect(shortDigest("ghcr.io/app@sha256:0123456789abcdef")).toBe("0123456789ab");
    expect(shortDigest("sha256:0123456789abcdef")).toBe("0123456789ab");
    expect(shortDigest(undefined)).toBe("");
  });
});

describe("updateLabels", () => {
  test("names the tag alone when no digest is known", () => {
    expect(updateLabels({ imageRef: "app:1.4" })).toEqual({ from: "1.4", to: "1.4" });
  });

  test("adds each side's digest to the tag", () => {
    expect(
      updateLabels({
        imageRef: "app:latest",
        fromDigest: "app@sha256:aaaaaaaaaaaaaaaa",
        toDigest: "sha256:bbbbbbbbbbbbbbbb",
      }),
    ).toEqual({ from: "latest (aaaaaaaaaaaa)", to: "latest (bbbbbbbbbbbb)" });
  });

  test("falls back to image ids for an image built locally", () => {
    expect(
      updateLabels({
        imageRef: "app",
        fromImageId: "sha256:111111111111aa",
        toImageId: "sha256:222222222222bb",
      }),
    ).toEqual({ from: "latest (111111111111)", to: "latest (222222222222)" });
  });
});

describe("awaitingVerdict", () => {
  const now = Date.parse("2026-10-03T08:00:00Z");
  const at = new Date(now - 60_000);

  test("waits for a verdict, a pending one, and an undecided regression", () => {
    expect(awaitingVerdict(at, undefined, now)).toBe(true);
    expect(awaitingVerdict(at, { deployId: "d", verdict: "pending" }, now)).toBe(true);
    expect(awaitingVerdict(at, { deployId: "d", verdict: "regressed" }, now)).toBe(true);
  });

  test("stops once settled", () => {
    expect(awaitingVerdict(at, { deployId: "d", verdict: "clean" }, now)).toBe(false);
    expect(awaitingVerdict(at, { deployId: "d", verdict: "regressed", decision: "kept" }, now)).toBe(false);
    expect(awaitingVerdict(at, { deployId: "d", verdict: "rolled_back_by_dozzle" }, now)).toBe(false);
  });

  test("stops asking about an old update", () => {
    expect(awaitingVerdict(new Date(now - 7 * 60 * 60 * 1000), undefined, now)).toBe(false);
  });
});
