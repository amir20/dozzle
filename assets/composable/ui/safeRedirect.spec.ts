import { describe, expect, test } from "vitest";
import { safeRedirect } from "./safeRedirect";

const ORIGIN = "https://dozzle.example.com";

describe("safeRedirect", () => {
  test.each([
    ["/", "/"],
    ["/container/abc", "/container/abc"],
    ["/settings?tab=1", "/settings?tab=1"],
    ["/a#frag", "/a#frag"],
  ])("keeps same-origin path %s", (input, expected) => {
    expect(safeRedirect(input, "", ORIGIN)).toBe(expected);
  });

  // Each of these navigates off-origin once a browser parses it. The tab and
  // newline cases are the ones a startsWith("//") guard lets through, because
  // the parser strips the control character only after that check has passed.
  test.each(["//evil.com", "/\\evil.com", "/\t/evil.com", "/\n/evil.com", "/\r/evil.com", "https://evil.com"])(
    "rejects %j",
    (hostile) => {
      expect(safeRedirect(hostile, "", ORIGIN)).toBe("/");
    },
  );

  test("falls back when empty", () => {
    expect(safeRedirect(null, "", ORIGIN)).toBe("/");
    expect(safeRedirect("", "", ORIGIN)).toBe("/");
  });

  test("keeps the base path", () => {
    expect(safeRedirect("/container/abc", "/dozzle", ORIGIN)).toBe("/dozzle/container/abc");
    expect(safeRedirect("//evil.com", "/dozzle", ORIGIN)).toBe("/dozzle/");
  });
});
