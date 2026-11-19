/**
 * @vitest-environment jsdom
 */
import { describe, expect, test } from "vitest";
import { bucketEnd, buildTimeline, problemAt, rowAt } from "./alertsTimeline";
import type { CloudAlert, CloudEvent } from "./cloudAlerts";
import type { RecentDeploy } from "./recentDeploys";

// Noon UTC (the suite runs with TZ=UTC), so a few hours either way stays on one day.
const NOW = new Date("2026-11-18T12:00:00Z");
const at = (iso: string) => new Date(iso).getTime() * 1e6;

function hit(overrides: Partial<CloudAlert> = {}): CloudAlert {
  return {
    alertId: "a1",
    containerId: "api",
    hostId: "h",
    ts: at("2026-11-18T09:00:00Z"),
    headline: "Pool exhausted",
    level: "error",
    eventCount: 1,
    createdAt: 0,
    isOrigin: true,
    subscriptionId: "7",
    ...overrides,
  };
}

function match(overrides: Partial<CloudEvent> = {}): CloudEvent {
  return {
    ts: at("2026-11-18T09:00:00Z"),
    containerId: "api",
    message: "pool exhausted",
    alertId: "a1",
    suppressed: true,
    ...overrides,
  };
}

const build = (hits: CloudAlert[], events: CloudEvent[]) => buildTimeline(hits, events, "en", NOW);

describe("buildTimeline", () => {
  // One problem, many matches, one notification: the reader sees a row that
  // says how often it happened and that they were told once.
  test("folds an alert's matches into one row: happened N times, notified once", () => {
    const events = [
      match({ ts: at("2026-11-18T09:00:00Z"), logId: 1 }),
      match({ ts: at("2026-11-18T10:00:00Z"), logId: 2 }),
      match({ ts: at("2026-11-18T11:00:00Z"), logId: 3 }),
    ];
    const [today] = build([hit()], events);

    expect(today.label).toBe("Today");
    expect(today.rows).toHaveLength(1);
    const row = today.rows[0];
    expect(row).toMatchObject({ kind: "problem", headline: "Pool exhausted", happened: 3, notified: 1 });
    expect(row.containers).toEqual(["api"]);
    // Newest first under the row.
    expect(row.occurrences.map((o) => o.logId)).toEqual([3, 2, 1]);
  });

  // The same rule firing twice on the same headline is the same problem, told twice.
  test("two alerts of the same rule and headline share a row: notified N times", () => {
    const hits = [hit(), hit({ alertId: "a2", containerId: "worker", ts: at("2026-11-18T10:00:00Z") })];
    const events = [match(), match({ alertId: "a2", containerId: "worker", ts: at("2026-11-18T10:00:00Z") })];
    const [row] = build(hits, events)[0].rows;

    expect(row).toMatchObject({ happened: 2, notified: 2 });
    expect(row.containers.sort()).toEqual(["api", "worker"]);
  });

  // An alert that first fired before the window still has matches in it. The
  // reader was told, just not today.
  test("a problem with only follow-up anchors was notified earlier", () => {
    const [row] = build([hit({ isOrigin: false })], [match()])[0].rows;
    expect(row).toMatchObject({ kind: "problem", notified: 0, happened: 1 });
  });

  test("the worst level across a problem's alerts wins", () => {
    const hits = [hit({ level: "warn" }), hit({ alertId: "a2", level: "fatal" })];
    expect(build(hits, [])[0].rows[0]).toMatchObject({ level: "fatal" });
  });

  // Matches that told nobody anything are one quiet row per day, at the bottom.
  test("matches that reached no alert become one quiet row per day", () => {
    const events = [
      match(),
      match({ alertId: undefined, containerId: "a", ts: at("2026-11-18T08:00:00Z") }),
      match({ alertId: undefined, containerId: "b", ts: at("2026-11-18T07:00:00Z") }),
      match({ alertId: undefined, containerId: "b", ts: at("2026-11-18T11:30:00Z") }),
      match({ alertId: undefined, containerId: "c", ts: at("2026-11-17T08:00:00Z") }),
    ];
    const [today, yesterday] = build([hit()], events);

    expect(today.rows.map((r) => r.kind)).toEqual(["problem", "quiet"]);
    expect(today.rows[1]).toMatchObject({ happened: 3 });
    expect([...today.rows[1].containers].sort()).toEqual(["a", "b"]);
    expect(yesterday.label).toBe("Yesterday");
    expect(yesterday.rows).toEqual([expect.objectContaining({ kind: "quiet", happened: 1, containers: ["c"] })]);
  });

  // A deploy verdict rides the same response but is not something a rule caught.
  test("deploy verdicts are not matches", () => {
    expect(build([], [match({ alertId: undefined, type: "deploy" })])).toEqual([]);
  });

  // An older Cloud sends alerts but no matches; the alert's own count stands in.
  test("without matches, an alert's own count is how often it happened", () => {
    const [row] = build([hit({ eventCount: 12 })], [])[0].rows;
    expect(row).toMatchObject({ happened: 12, notified: 1 });
    expect(row.occurrences).toHaveLength(1);
  });

  test("a match on an alert with no anchor in the window still gets a row", () => {
    const [row] = build([], [match({ alertId: "gone", message: "disk full" })])[0].rows;
    expect(row).toMatchObject({ kind: "problem", headline: "disk full", notified: 0 });
  });

  test("days come newest first and rows inside a day by their newest match", () => {
    const hits = [
      hit({ alertId: "x", headline: "early", ts: at("2026-11-18T01:00:00Z") }),
      hit({ alertId: "y", headline: "late", ts: at("2026-11-18T11:00:00Z") }),
      hit({ alertId: "z", headline: "old", ts: at("2026-11-17T11:00:00Z") }),
    ];
    const days = build(hits, []);
    expect(days.map((d) => d.rows.map((r) => (r.kind === "problem" ? r.headline : "")))).toEqual([
      ["late", "early"],
      ["old"],
    ]);
  });
});

describe("chart helpers", () => {
  const hour = 3600e9;
  const start = at("2026-11-18T08:00:00Z");
  const days = build(
    [hit({ ts: start + hour / 2 })],
    [
      match({ ts: start + hour / 2 }),
      match({ ts: start + hour / 3 }),
      match({ alertId: undefined, ts: start + hour + 1 }),
      match({ alertId: undefined, ts: start + hour + 2 }),
      match({ alertId: undefined, ts: start + hour + 3 }),
    ],
  );

  test("a bucket runs to the next one's start, the last to the window's end", () => {
    const buckets = [
      { startTs: 0, matched: 0, notified: 0 },
      { startTs: 10, matched: 0, notified: 0 },
    ];
    expect(bucketEnd(buckets, 0, 20)).toBe(10);
    expect(bucketEnd(buckets, 1, 20)).toBe(20);
  });

  // Hover names the problem a bar is about, and only a problem: the quiet
  // matches have no headline to name.
  test("hover names the problem inside a bar", () => {
    expect(problemAt(days, start, start + hour)).toBe("Pool exhausted");
    expect(problemAt(days, start + hour, start + 2 * hour)).toBeUndefined();
  });

  // Click lands on the row with the most matches in the bar, else the nearest.
  test("a bar's row is the one with the most matches in it, or the nearest", () => {
    expect(rowAt(days, start, start + hour)?.kind).toBe("problem");
    expect(rowAt(days, start + hour, start + 2 * hour)?.kind).toBe("quiet");
    expect(rowAt(days, start + 5 * hour, start + 6 * hour)?.kind).toBe("quiet");
    expect(rowAt(days, start - 5 * hour, start - 4 * hour)?.kind).toBe("problem");
  });
});

describe("buildTimeline with updates", () => {
  const deploy = (overrides: Partial<RecentDeploy> = {}): RecentDeploy => ({
    host: "h",
    name: "api",
    oldId: "",
    newId: "api2",
    imageRef: "ghcr.io/x/api:v2",
    at: "2026-11-18T08:00:00Z",
    source: "external",
    ...overrides,
  });

  test("an update sits among the problems in time order and counts no matches", () => {
    const days = buildTimeline(
      [hit({ ts: at("2026-11-18T09:00:00Z") })],
      [match({ ts: at("2026-11-18T09:00:00Z") }), match({ alertId: undefined, ts: at("2026-11-18T10:00:00Z") })],
      "en",
      NOW,
      [deploy({ at: "2026-11-18T08:00:00Z" }), deploy({ at: "2026-11-18T11:00:00Z", newId: "api3", imageRef: "x:v3" })],
    );
    expect(days).toHaveLength(1);
    expect(days[0].rows.map((r) => r.kind)).toEqual(["update", "problem", "update", "quiet"]);
    const outside = days[0].rows[2];
    expect(outside).toMatchObject({ kind: "update", name: "api", label: "updated", from: "", to: "v2", happened: 0 });
    expect(outside.occurrences).toEqual([{ ts: at("2026-11-18T08:00:00Z"), containerId: "api2", text: "v2" }]);
  });

  test("Dozzle's own update names both images", () => {
    const [day] = buildTimeline([], [], "en", NOW, [
      deploy({
        source: "schedule",
        imageRef: "nginx:latest",
        fromDigest: "sha256:aaaaaaaaaaaa1",
        toDigest: "sha256:bbbbbbbbbbbb2",
      }),
    ]);
    expect(day.rows[0]).toMatchObject({ from: "latest (aaaaaaaaaaaa)", to: "latest (bbbbbbbbbbbb)" });
  });

  test("a bar is never about an update", () => {
    const days = buildTimeline([], [match({ alertId: undefined, ts: at("2026-11-18T10:00:00Z") })], "en", NOW, [
      deploy({ at: "2026-11-18T09:00:00Z" }),
    ]);
    expect(rowAt(days, at("2026-11-18T08:30:00Z"), at("2026-11-18T09:30:00Z"))?.kind).toBe("quiet");
  });
});
