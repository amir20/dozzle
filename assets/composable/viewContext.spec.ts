/** @vitest-environment jsdom */
import { describe, expect, test } from "vitest";
import { narrowedLevels, routeKind } from "./viewContext";
import type { Level } from "@/models/LogEntry";

describe("routeKind", () => {
  test("names the shape of the page, not the route", () => {
    expect(routeKind("/container/[id]")).toBe("container");
    expect(routeKind("/host/[id]")).toBe("host");
    expect(routeKind("/merged/[ids]")).toBe("merged");
    expect(routeKind("/stack/[name]")).toBe("stack");
    expect(routeKind("/notifications")).toBe("notifications");
  });

  test("collapses the time-travel route onto the container it shows", () => {
    expect(routeKind("/container/[id].time.[datetime]")).toBe("container");
  });

  test("treats home and unnamed routes as the index", () => {
    expect(routeKind("/")).toBe("index");
    expect(routeKind(undefined)).toBe("index");
    expect(routeKind(null)).toBe("index");
    expect(routeKind(Symbol("anon"))).toBe("index");
  });
});

describe("narrowedLevels", () => {
  const all: Level[] = ["info", "debug", "warn", "error", "fatal", "trace", "unknown"];

  test("reports nothing when every level is shown, because that is not a filter", () => {
    expect(narrowedLevels(new Set(all))).toBeUndefined();
  });

  test("reports the narrowed set in a stable order", () => {
    expect(narrowedLevels(new Set(["error", "warn"] as Level[]))).toEqual(["warn", "error"]);
  });

  test("treats an empty set as no filter rather than as hiding everything", () => {
    expect(narrowedLevels(new Set())).toBeUndefined();
  });
});
