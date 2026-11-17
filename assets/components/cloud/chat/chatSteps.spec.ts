/**
 * @vitest-environment jsdom
 */
import { readdirSync, readFileSync } from "node:fs";
import { resolve } from "node:path";
import { parse } from "yaml";
import { describe, expect, test } from "vitest";
import type { ChatStep } from "@/composable/cloud/cloudChat";
import { stepKeys, stepResult, stepText, toolGroup } from "./chatSteps";

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
    expect(stepText(step({ tool: "brand_new_tool", label: "Doing something new" }))).toBe("Doing something new");
    expect(stepText(step({ tool: "toString", label: "Not a key" }))).toBe("Not a key");
    expect(stepText(step({ phase: "pondering", label: "Pondering" }))).toBe("Pondering");
  });

  test("uses its own words for a known one", () => {
    expect(stepText(step({ tool: "search_logs", label: "ignored" }))).toBe("Searching logs");
    expect(stepText(step({ phase: "writing", label: "ignored" }))).toBe("Writing the answer");
    expect(stepText(step({ tool: "show_cpu", label: "ignored" }))).toBe("Drawing CPU over time");
    expect(stepText(step({ tool: "rpc_restart_container", label: "ignored" }))).toBe("Restarting the container");
  });
});

// Every tool cloud's agent sends a step for (agent.go toolStatusMessage, plus
// the rpc_ tools it proxies to Dozzle).
const cloudTools = [
  "search_logs",
  "search_log_stats",
  "history_get_events",
  "history_container_changes",
  "history_get_alerts",
  "history_get_findings",
  "history_list_containers",
  "docs_list",
  "docs_read",
  "web_search",
  "metrics_container_stats",
  "metrics_container_timeseries",
  "ignore_finding",
  "mute_pattern",
  "unmute_pattern",
  "list_mute_rules",
  "remember",
  "forget_note",
  "schedule_check",
  "list_dozzle_instances",
  "rpc_find_containers",
  "rpc_list_all_containers",
  "rpc_list_running_containers",
  "rpc_list_hosts",
  "rpc_get_running_container_stats",
  "rpc_inspect_container",
  "rpc_inspect_containers",
  "rpc_fetch_container_logs",
  "rpc_start_container",
  "rpc_stop_container",
  "rpc_restart_container",
  "rpc_remove_container",
  "rpc_update_container",
  "rpc_rollback_container",
  "rpc_check_image_updates",
  "rpc_list_notifications",
  "rpc_create_log_notification",
  "rpc_create_metric_notification",
  "rpc_create_event_notification",
  "show_memory_vs_limit",
  "show_cpu",
  "show_pattern_count",
  "show_container_ranking",
  "show_logs",
];
const cloudPhases = ["reading", "reviewing", "writing", "retrying"];

const localesDir = resolve(__dirname, "../../../../locales");
const locales = readdirSync(localesDir)
  .filter((f) => f.endsWith(".yml"))
  .map((f) => [f, parse(readFileSync(resolve(localesDir, f), "utf8"))["cloud-chat"]] as const);

describe("every cloud step has words in every locale", () => {
  test("all locale files are read", () => {
    expect(locales.length).toBeGreaterThan(10);
  });

  test.each(cloudTools)("%s", (tool) => {
    const key = stepKeys[tool];
    expect(key).toBeTruthy();
    for (const [file, chat] of locales) {
      expect(chat?.step?.[key], `${file} cloud-chat.step.${key}`).toEqual(expect.any(String));
    }
    expect(stepText(step({ tool, label: "cloud label" }))).not.toBe("cloud label");
  });

  test.each(cloudPhases)("phase %s", (phase) => {
    for (const [file, chat] of locales) {
      expect(chat?.phase?.[phase], `${file} cloud-chat.phase.${phase}`).toEqual(expect.any(String));
    }
    expect(stepText(step({ phase, label: "cloud label" }))).not.toBe("cloud label");
  });

  test("no step key is left without a tool", () => {
    const used = new Set(Object.values(stepKeys));
    for (const [file, chat] of locales) {
      expect(
        Object.keys(chat.step).filter((k) => !used.has(k)),
        file,
      ).toEqual([]);
    }
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
