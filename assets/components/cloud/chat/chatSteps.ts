import type { Component } from "vue";
import type { ChatStep } from "@/composable/cloud/cloudChat";
import { i18n } from "@/modules/i18n";
import mdiMagnify from "~icons/mdi/magnify";
import mdiPackageVariant from "~icons/mdi/package-variant-closed";
import mdiChartLine from "~icons/mdi/chart-line";
import mdiFileSearchOutline from "~icons/mdi/file-search-outline";
import mdiBellOutline from "~icons/mdi/bell-outline";
import mdiClipboardTextSearchOutline from "~icons/mdi/clipboard-text-search-outline";
import mdiClockOutline from "~icons/mdi/clock-outline";
import mdiBookOpenVariant from "~icons/mdi/book-open-variant";
import mdiBellCogOutline from "~icons/mdi/bell-cog-outline";
import mdiPlayCircleOutline from "~icons/mdi/play-circle-outline";
import mdiWrenchOutline from "~icons/mdi/wrench-outline";

const { t } = i18n.global;

/** The model-round phases this build has words for. */
const PHASES = ["reading", "reviewing", "writing", "retrying"] as const;

type Group =
  "logs" | "containers" | "stats" | "inspect" | "alerts" | "findings" | "events" | "docs" | "notifications" | "action";

/** Which kind of lookup a tool is, by cloud's name for it. Cloud names the
 *  tools it proxies to Dozzle `rpc_<dozzle name>`. Undefined for a tool this
 *  build has never heard of, which then shows cloud's own label. */
export function toolGroup(tool = ""): Group | undefined {
  const name = tool.replace(/^rpc_/, "");
  if (name === "search_logs" || name.includes("logs")) return "logs";
  if (name.includes("notification")) return "notifications";
  if (/^(start|stop|restart|remove|update)_container$/.test(name)) return "action";
  if (name === "inspect_container") return "inspect";
  if (name.includes("stats") || name.startsWith("metrics_")) return "stats";
  if (name === "history_get_alerts") return "alerts";
  if (name === "history_get_findings") return "findings";
  if (name === "history_get_events" || name === "history_container_changes") return "events";
  if (name.startsWith("docs_")) return "docs";
  if (name.includes("container") || name.includes("hosts") || name === "list_dozzle_instances") return "containers";
  return undefined;
}

const icons: Record<Group, Component> = {
  logs: mdiMagnify,
  containers: mdiPackageVariant,
  stats: mdiChartLine,
  inspect: mdiFileSearchOutline,
  alerts: mdiBellOutline,
  findings: mdiClipboardTextSearchOutline,
  events: mdiClockOutline,
  docs: mdiBookOpenVariant,
  notifications: mdiBellCogOutline,
  action: mdiPlayCircleOutline,
};

export function stepIcon(tool?: string): Component {
  const group = toolGroup(tool);
  return group ? icons[group] : mdiWrenchOutline;
}

/** The words for each tool cloud sends, by cloud's name for it, as a key under
 *  `cloud-chat.step`. Cloud names the tools it proxies to Dozzle
 *  `rpc_<dozzle name>`. */
export const stepKeys: Record<string, string> = {
  search_logs: "logs",
  search_log_stats: "count-logs",
  history_get_events: "events",
  history_container_changes: "image-history",
  history_get_alerts: "alerts",
  history_get_findings: "findings",
  history_list_containers: "containers",
  docs_list: "docs-index",
  docs_read: "docs",
  web_search: "web",
  metrics_container_stats: "stats",
  metrics_container_timeseries: "stats",
  ignore_finding: "ignore-finding",
  mute_pattern: "mute",
  unmute_pattern: "unmute",
  list_mute_rules: "mute-rules",
  remember: "remember",
  forget_note: "forget",
  schedule_check: "schedule",
  list_dozzle_instances: "instances",
  rpc_find_containers: "containers",
  rpc_list_all_containers: "containers",
  rpc_list_running_containers: "containers",
  rpc_list_hosts: "hosts",
  rpc_get_running_container_stats: "stats",
  rpc_inspect_container: "inspect",
  rpc_inspect_containers: "inspect",
  rpc_fetch_container_logs: "logs",
  rpc_start_container: "start",
  rpc_stop_container: "stop",
  rpc_restart_container: "restart",
  rpc_remove_container: "remove",
  rpc_update_container: "update",
  rpc_rollback_container: "rollback",
  rpc_check_image_updates: "image-updates",
  rpc_list_notifications: "notifications",
  rpc_create_log_notification: "create-alert",
  rpc_create_metric_notification: "create-alert",
  rpc_create_event_notification: "create-alert",
  show_memory_vs_limit: "draw-memory",
  show_cpu: "draw-cpu",
  show_pattern_count: "draw-count",
  show_container_ranking: "draw-ranking",
  show_logs: "draw-logs",
};

/** The step in the reader's language, or cloud's English when this build has
 *  no words for it. */
export function stepText(step: ChatStep): string {
  if (step.tool) {
    const key = Object.hasOwn(stepKeys, step.tool) ? stepKeys[step.tool] : undefined;
    return key ? t(`cloud-chat.step.${key}`) : step.label;
  }
  const phase = PHASES.find((p) => p === step.phase);
  return phase ? t(`cloud-chat.phase.${phase}`) : step.label;
}

/** What came back: a count in the reader's language, cloud's summary when the
 *  result was not a count, nothing when it was neither. */
export function stepResult(step: ChatStep): string {
  if (step.state !== "done") return "";
  if (step.count === 0) return t("cloud-chat.nothing-found");
  if (step.count > 0) return t("cloud-chat.found", { n: step.count });
  return "";
}
