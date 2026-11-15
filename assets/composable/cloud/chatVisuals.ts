import type { VolumeBucket } from "@/composable/cloud/searchVolume";
import { i18n } from "@/modules/i18n";

const { t } = i18n.global;

/** One bucket of a container's memory, as the metrics panel receives it. */
export type ChatVisualPoint = {
  ts: number;
  memory: number;
  memoryUsage: number;
  memoryUsageMax?: number;
};

export type ChatVisualFacts = {
  exits: number;
  exitCodes?: { code: string; count: number }[];
  ooms: number;
  restarts: number;
  /** False when no lifecycle events are stored for the container. */
  eventsRecorded: boolean;
  /** Cloud's stored events hit its row cap, so the counts are a floor. */
  capped: boolean;
  memoryPeak?: number;
};

/**
 * One picture under an answer. Cloud measured every number in it and the
 * assistant only chose which to show, so it is drawn as it came. See
 * cloud.proto (ChatVisual) for the fields each type sets.
 */
export type ChatVisual = {
  type: "memory_vs_limit" | "pattern_count" | "container_ranking" | "logs" | "container_events";
  container?: string;
  hostId?: string;
  hostName?: string;
  query?: string;
  metric?: "cpu" | "memory" | "restarts";
  /** Window, unix milliseconds. */
  from: number;
  to: number;
  facts?: ChatVisualFacts;
  total?: number;
  rows?: { container: string; hostId?: string; hostName?: string; value: number }[];
  /** Newest first. A line cloud stored without a time has no timestamp. */
  lines?: { timestamp?: string; level?: string; message: string }[];
  volume?: VolumeBucket[];
  stepNs?: number;
  points?: ChatVisualPoint[];
  /** Bytes; 0 or absent when no limit is set or it could not be read. */
  memoryLimit?: number;
};

/**
 * What happened to a container, in one line: exits by code, out-of-memory
 * kills, restarts. "No out-of-memory kill" is said only where cloud has events
 * for the container at all, and never from a capped list, because silence
 * there proves nothing.
 */
export function factsLine(facts: ChatVisualFacts): string {
  const parts: string[] = [];
  if (facts.exits) {
    const codes = (facts.exitCodes ?? []).map((c) => (facts.exitCodes!.length > 1 ? `${c.code} (${c.count})` : c.code));
    parts.push(
      codes.length
        ? `${t("cloud-chat.visual.exits", { n: facts.exits })}, ${t("cloud-chat.visual.exit-codes", { codes: codes.join(", ") })}`
        : t("cloud-chat.visual.exits", { n: facts.exits }),
    );
  }
  if (facts.ooms) parts.push(t("cloud-chat.visual.ooms", { n: facts.ooms }));
  else if (facts.eventsRecorded && !facts.capped && facts.exits) parts.push(t("cloud-chat.visual.no-oom"));
  if (facts.restarts) parts.push(t("cloud-chat.visual.restarts", { n: facts.restarts }));
  return parts.join(" · ");
}

/**
 * The memory chart's bars. With a limit, each bucket is its share of it and the
 * chart is drawn against 100%, so a container near its limit looks near it.
 * Without one there is no ceiling to draw against, so the bars are bytes and
 * the chart scales to its own peak, as the metrics panel does.
 */
export function memoryBars(points: ChatVisualPoint[], limit: number) {
  return points.map((p) => {
    const peak = p.memoryUsageMax ?? p.memoryUsage;
    if (limit <= 0) return { percent: p.memoryUsage, value: p.memoryUsage, peak, peakPercent: peak };
    return { percent: (p.memoryUsage / limit) * 100, value: p.memoryUsage, peak, peakPercent: (peak / limit) * 100 };
  });
}
