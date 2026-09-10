import { describe, expect, test } from "vitest";
import { logMomentRoute } from "./logJump";

describe("logMomentRoute", () => {
  const date = new Date("2026-09-09T14:02:44.123Z");

  test("targets the historical view of the container at that moment", () => {
    expect(logMomentRoute({ containerId: "abc123", date })).toEqual({
      name: "/container/[id].time.[datetime]",
      params: { id: "abc123", datetime: "2026-09-09T14:02:44.123Z" },
      query: {},
    });
  });

  test("carries a log id so the view scrolls to the exact line", () => {
    const route = logMomentRoute({ containerId: "abc123", date, logId: 42 });
    expect(route).toMatchObject({ query: { logId: "42" } });
  });

  test("drops a zero log id, which means the alert was never anchored to a line", () => {
    // Metric and event alerts carry no log line, and Cloud sends 0 for them.
    // Passing it through would target a line that does not exist.
    expect(logMomentRoute({ containerId: "abc123", date, logId: 0 })).toMatchObject({ query: {} });
    expect(logMomentRoute({ containerId: "abc123", date, logId: "0" })).toMatchObject({ query: {} });
  });

  test("pre-fills the search box when the caller has a term", () => {
    const route = logMomentRoute({ containerId: "abc123", date, logId: 7, query: "connection refused" });
    expect(route).toMatchObject({ query: { logId: "7", q: "connection refused" } });
  });

  test("omits an empty search term rather than sending a blank filter", () => {
    expect(logMomentRoute({ containerId: "abc123", date, query: "" })).toMatchObject({ query: {} });
  });
});
