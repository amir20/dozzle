/** @vitest-environment jsdom */
import { describe, expect, test } from "vitest";
import { similarSearch } from "./similarSearch";

// The regex runs in Go on the server, but every construct used here means the
// same thing in JS, so matching it locally is a fair check of what it finds.
function finds(source: string, other: string, whole = false) {
  const result = similarSearch(source, whole);
  if (!result) throw new Error("no pattern for " + source);
  return new RegExp(result.regex).test(other);
}

describe("similarSearch", () => {
  test("wildcards numbers and keeps the words", () => {
    expect(similarSearch("Deleted 62 expired alerts")).toEqual({
      regex: "Deleted \\d+(?:\\.\\d+)? expired alerts",
      phrase: "expired alerts",
    });
    expect(finds("Deleted 62 expired alerts", "Deleted 11628 expired alerts")).toBe(true);
    expect(finds("Deleted 62 expired alerts", "Deleted 62 expired rows")).toBe(false);
  });

  test("claims timestamps, UUIDs, IPs and hashes whole", () => {
    const line =
      "2026-10-19T21:07:54.123Z request 3f2b8c1e-9a4d-4e6f-8b2a-1c3d5e7f9a0b from 10.0.0.12:5432 commit a1b2c3d4e5 done";
    const other =
      "2026-10-20T01:00:00Z request 00000000-1111-2222-3333-444444444444 from 192.168.1.1:80 commit ffff00001 done";
    expect(finds(line, other)).toBe(true);
    expect(similarSearch(line)?.regex).not.toContain("2026");
  });

  test("leaves words with digits in them alone", () => {
    expect(finds("user123 logged in from v2 client", "user456 logged in from v2 client")).toBe(false);
  });

  test("escapes regex characters in the fixed text", () => {
    expect(finds("GET /api/users?id=5 (200)", "GET /api/users?id=19 (200)")).toBe(true);
    expect(finds("GET /api/users?id=5 (200)", "GET /api/usersXid=19 (200)")).toBe(false);
  });

  test("anchors a whole field value at both ends", () => {
    expect(similarSearch("Deleted 5 rows", true)?.regex).toBe("^Deleted \\d+(?:\\.\\d+)? rows$");
    expect(finds("Deleted 5 rows", "Deleted 5 rows and more", true)).toBe(false);
  });

  test("ignores colour codes and search highlight markers", () => {
    expect(similarSearch("\u001b[31mDeleted\u001b[0m 5 rows")?.regex).toBe("Deleted \\d+(?:\\.\\d+)? rows");
  });

  test("cuts a long line on a space and does not anchor its end", () => {
    const line = "starting worker " + "pool ".repeat(40) + "with 8 threads";
    const result = similarSearch(line, true)!;
    expect(result.regex.endsWith("$")).toBe(false);
    expect(result.regex.endsWith(" ")).toBe(false);
    expect(finds(line, line.replace("8 threads", "16 threads"), true)).toBe(true);
  });

  test("gives up on a line with no words to hold on to", () => {
    expect(similarSearch("12 34 56")).toBeNull();
    expect(similarSearch("   ")).toBeNull();
  });
});
