/**
 * @vitest-environment jsdom
 */
import { describe, expect, test } from "vitest";
import { DeployLogEntry, LoadMoreLogEntry, SimpleLogEntry, type LogEntry, type LogMessage } from "@/models/LogEntry";
import type { ContainerUpdate } from "@/models/ContainerUpdate";
import { mergeByTime } from "./logLoader";
import {
  MARKER_START_SLACK_MS,
  firstLineByContainer,
  insertMarker,
  repositionMarkers,
  windowReachesUpdate,
} from "./updateMarkers";

const T0 = Date.UTC(2026, 9, 3, 3, 0, 0);
const line = (container: string, ms: number, id = ms) =>
  new SimpleLogEntry(`line ${ms}`, container, id, new Date(T0 + ms), "info", "stdout", `line ${ms}`);
const marker = (container: string, ms = 0) =>
  new DeployLogEntry({ newId: container, at: new Date(T0 + ms).toISOString() } as ContainerUpdate, new Date(T0 + ms));
const loader = () => new LoadMoreLogEntry(new Date(), async () => {});
const kinds = (list: LogEntry<LogMessage>[]) =>
  list.map((e) =>
    e instanceof DeployLogEntry ? `M:${e.containerID}` : e instanceof LoadMoreLogEntry ? "L" : e.message,
  );

describe("windowReachesUpdate", () => {
  test("a container whose first line came right after the update", () => {
    const first = firstLineByContainer([marker("a"), line("a", 3_000), line("a", 5_000)]);
    expect(windowReachesUpdate(marker("a"), first)).toBe(true);
  });

  // The review's case: updated yesterday, and the opening tail is today's lines.
  test("not a container whose lines on screen are long after the update", () => {
    const first = firstLineByContainer([line("a", 24 * 3600_000)]);
    expect(windowReachesUpdate(marker("a"), first)).toBe(false);
  });

  test("not a container with no line on screen", () => {
    expect(windowReachesUpdate(marker("a"), firstLineByContainer([line("b", 1_000)]))).toBe(false);
  });

  test("the marker itself does not count as a line", () => {
    const first = firstLineByContainer([marker("a"), line("a", MARKER_START_SLACK_MS + 1)]);
    expect(windowReachesUpdate(marker("a"), first)).toBe(false);
  });
});

describe("insertMarker", () => {
  test("goes above the container's first line, under the load-more row", () => {
    const list = insertMarker([loader(), line("a", 2_000), line("a", 3_000)], marker("a"));
    expect(kinds(list)).toEqual(["L", "M:a", "line 2000", "line 3000"]);
  });

  test("goes after another container's lines from before the update", () => {
    const list = insertMarker([loader(), line("b", -5_000), line("a", 2_000), line("b", 1_000)], marker("a"));
    expect(kinds(list)).toEqual(["L", "line -5000", "M:a", "line 2000", "line 1000"]);
  });
});

describe("a marker put back after a trim", () => {
  // The same steps the stream runs: put back on what the trim left, then older lines
  // merge in and markers reposition.
  test("lands in time among a merged view's reloaded lines", () => {
    const m = marker("a", 1_000);
    const after = insertMarker([loader(), line("b", 1_500), line("a", 2_000)], m);
    expect(kinds(after)).toEqual(["L", "M:a", "line 1500", "line 2000"]);

    const [head, ...rest] = after;
    const reloaded = repositionMarkers([head, ...mergeByTime(rest, [line("b", 500), line("a", 1_200)])]);
    expect(kinds(reloaded)).toEqual(["L", "line 500", "M:a", "line 1200", "line 1500", "line 2000"]);
  });
});

describe("repositionMarkers", () => {
  // A marker placed before its container's lines arrived, then older lines loaded
  // above it.
  test("moves a marker back over its container's first line", () => {
    const list = repositionMarkers([loader(), line("a", 1_000), marker("a"), line("a", 9_000)]);
    expect(kinds(list)).toEqual(["L", "M:a", "line 1000", "line 9000"]);
  });

  test("leaves a list in order alone", () => {
    const list = [loader(), marker("a"), line("a", 1_000)];
    expect(repositionMarkers(list)).toBe(list);
  });
});
