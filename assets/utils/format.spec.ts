import { describe, expect, test } from "vitest";
import { formatBytes, stripVersion, hashCode } from "./format";

describe("formatBytes", () => {
  test("zero bytes", () => {
    expect(formatBytes(0)).toBe("0 Bytes");
    expect(formatBytes(0, { short: true })).toBe("0B");
  });

  test("bytes under 1KB", () => {
    expect(formatBytes(512)).toBe("512 Bytes");
    expect(formatBytes(512, { short: true })).toBe("512B");
  });

  test("scales to KB/MB/GB", () => {
    expect(formatBytes(1024)).toBe("1 KB");
    expect(formatBytes(1024, { short: true })).toBe("1K");
    expect(formatBytes(1024 * 1024)).toBe("1 MB");
    expect(formatBytes(1024 * 1024, { short: true })).toBe("1M");
    expect(formatBytes(1024 * 1024 * 1024)).toBe("1 GB");
  });

  test("honors decimals option", () => {
    expect(formatBytes(1500)).toBe("1.46 KB");
    expect(formatBytes(1500, { decimals: 0 })).toBe("1 KB");
    expect(formatBytes(1500, { decimals: 1 })).toBe("1.5 KB");
  });

  test("negative decimals clamp to zero", () => {
    expect(formatBytes(1500, { decimals: -3 })).toBe("1 KB");
  });
});

describe("stripVersion", () => {
  test("removes tag", () => {
    expect(stripVersion("nginx:1.25")).toBe("nginx");
  });

  test("leaves untagged image unchanged", () => {
    expect(stripVersion("nginx")).toBe("nginx");
  });

  test("splits on first colon", () => {
    expect(stripVersion("registry:5000/img:tag")).toBe("registry");
  });
});

describe("hashCode", () => {
  test("empty string is zero", () => {
    expect(hashCode("")).toBe(0);
  });

  test("is deterministic", () => {
    expect(hashCode("dozzle")).toBe(hashCode("dozzle"));
  });

  test("differs for different input", () => {
    expect(hashCode("a")).not.toBe(hashCode("b"));
  });
});
