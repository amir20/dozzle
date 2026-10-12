import { describe, expect, test } from "vitest";
import { shortImageRef, stripImageDigest } from "./image";

const digest = "sha256:" + "a".repeat(64);

describe("stripImageDigest", () => {
  test("drops the digest and keeps the rest", () => {
    expect(stripImageDigest(`ghcr.io/amir20/dozzle:v11.3.1@${digest}`)).toBe("ghcr.io/amir20/dozzle:v11.3.1");
    expect(stripImageDigest("nginx:1.27")).toBe("nginx:1.27");
  });
});

describe("shortImageRef", () => {
  test.each([
    ["ghcr.io/amir20/dozzle:v11.3.1", "dozzle:v11.3.1"],
    [`ghcr.io/amir20/dozzle:v11.3.1@${digest}`, "dozzle:v11.3.1"],
    [`nginx@${digest}`, "nginx"],
    ["docker.io/library/nginx:1.27", "nginx:1.27"],
    ["victoriametrics/victoria-metrics:v1.153.0", "victoria-metrics:v1.153.0"],
    ["registry:5000/team/app:2", "app:2"],
    ["localhost:5001/testapp:latest", "testapp:latest"],
    ["amir20/doligence-ui", "doligence-ui"],
    ["postgres:18.6-alpine", "postgres:18.6-alpine"],
  ])("%s -> %s", (image, expected) => {
    expect(shortImageRef(image)).toBe(expected);
  });

  test("shortens a bare image id like docker ps does", () => {
    const id = "87b64a8d47e58e8610089d4f76b7d4a0f908bd7717112f03610e6ae6b046d849";
    expect(shortImageRef(`sha256:${id}`)).toBe("87b64a8d47e5");
    expect(shortImageRef(id)).toBe("87b64a8d47e5");
  });

  test("returns empty for a missing image", () => {
    expect(shortImageRef("")).toBe("");
    expect(shortImageRef(undefined)).toBe("");
  });
});
