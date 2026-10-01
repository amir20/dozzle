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

/** The step in the reader's language, or cloud's English when this build has
 *  no words for it. */
export function stepText(step: ChatStep): string {
  if (step.tool) {
    const group = toolGroup(step.tool);
    return group ? t(`cloud-chat.step.${group}`) : step.label;
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
