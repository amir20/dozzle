import { describe, expect, test, vi } from "vitest";
import type { Container } from "@/models/Container";

import { analyticsLogsParams, createLogsTableSql } from "./analyticsTable";

vi.mock("@/stores/config", () => ({
  __esModule: true,
  default: { base: "" },
  withBase: (path: string) => `/base${path}`,
}));

const container = (id: string, host = "local") => ({ id, name: id, host }) as unknown as Container;

describe("analyticsLogsParams", () => {
  test("asks for every source's JSON lines in one request, in order", () => {
    const params = analyticsLogsParams([container("b"), container("a", "remote")]);
    expect(params.get("format")).toBe("jsonl");
    expect(params.get("stdout")).toBe("1");
    expect(params.get("stderr")).toBe("1");
    expect(params.getAll("container")).toEqual(["local/b", "remote/a"]);
  });
});

describe("createLogsTableSql", () => {
  test("one container is just its fields", () => {
    const sql = createLogsTableSql("logs.json", false);
    expect(sql).toContain("SELECT unnest(m) FROM read_json('logs.json'");
    expect(sql).not.toContain("_container");
    expect(sql).not.toContain("sample_size");
  });

  test("several containers name each row's container from c and read the whole file for its schema", () => {
    const sql = createLogsTableSql("logs.json", true);
    expect(sql).toContain("c AS _container");
    expect(sql).toContain("epoch_ms(ts) AS _ts");
    expect(sql).toContain("sample_size = -1");
  });
});
