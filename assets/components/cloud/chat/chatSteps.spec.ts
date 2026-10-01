/**
 * @vitest-environment jsdom
 */
import { describe, expect, test } from "vitest";
import type { ChatStep } from "@/composable/cloud/cloudChat";
import { stepResult, stepText, toolGroup } from "./chatSteps";

const step = (s: Partial<ChatStep>): ChatStep => ({ id: 1, label: "", state: "done", count: -1, durationMs: 0, ...s });

describe("toolGroup", () => {
  test.each([
    ["search_logs", "logs"],
    ["rpc_fetch_container_logs", "logs"],
    ["rpc_find_containers", "containers"],
    ["rpc_list_hosts", "containers"],
    ["rpc_get_running_container_stats", "stats"],
    ["metrics_container_timeseries", "stats"],
    ["rpc_inspect_container", "inspect"],
    ["rpc_restart_container", "action"],
    ["rpc_create_log_notification", "notifications"],
    ["history_get_alerts", "alerts"],
    ["history_get_findings", "findings"],
    ["history_container_changes", "events"],
    ["docs_read", "docs"],
  ])("%s → %s", (tool, group) => {
    expect(toolGroup(tool)).toBe(group);
  });

  test("a tool this build never heard of has no group", () => {
    expect(toolGroup("mute_pattern")).toBeUndefined();
  });
});

describe("stepText", () => {
  test("falls back to cloud's label for an unknown tool or phase", () => {
    expect(stepText(step({ tool: "mute_pattern", label: "Muting" }))).toBe("Muting");
    expect(stepText(step({ phase: "pondering", label: "Pondering" }))).toBe("Pondering");
  });

  test("uses its own words for a known one", () => {
    expect(stepText(step({ tool: "search_logs", label: "ignored" }))).toBe("Searching logs");
    expect(stepText(step({ phase: "writing", label: "ignored" }))).toBe("Writing the answer");
  });
});

describe("stepResult", () => {
  test("counts only finished steps", () => {
    expect(stepResult(step({ count: 42 }))).toBe("42 found");
    expect(stepResult(step({ count: 0 }))).toBe("nothing found");
    expect(stepResult(step({ count: -1, summary: "1.2 GB" }))).toBe("");
    expect(stepResult(step({ count: 42, state: "running" }))).toBe("");
  });
});
