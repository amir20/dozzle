import { describe, expect, test } from "vitest";
import { numericFieldQuery } from "./sql";

describe("numericFieldQuery", () => {
  test("top-level key is a bare column", () => {
    expect(numericFieldQuery(["latency"])).toBe(
      `SELECT TRY_CAST("latency" AS DOUBLE) AS "latency"\nFROM logs\nWHERE TRY_CAST("latency" AS DOUBLE) IS NOT NULL`,
    );
  });

  test("nested key goes through the table so it reads as a struct field", () => {
    expect(numericFieldQuery(["http", "duration"])).toContain(
      `TRY_CAST("logs"."http"."duration" AS DOUBLE) AS "http.duration"`,
    );
  });

  test("quotes are escaped", () => {
    expect(numericFieldQuery(['a"b'])).toContain(`"a""b"`);
  });
});
