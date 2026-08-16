/**
 * Alert markers for the scroll gutter.
 *
 * Turns measured element positions into fractional offsets down the scrollable
 * content, merging any that would render on top of each other. A storm of
 * alerts must not become a solid bar of overlapping ticks — the gutter's job is
 * to tell you *where* to scroll, and a solid bar tells you nothing.
 */

export interface AlertMeasurement {
  alertId: number;
  /** Distance from the top of the scrollable content, in pixels. */
  top: number;
  level: string;
  headline: string;
}

export interface GutterTick {
  /** Fraction down the content, 0..1. */
  offset: number;
  /** The alert to scroll to when the tick is clicked — the topmost of a merge. */
  alertId: number;
  level: string;
  headline: string;
  /** How many alerts this tick stands for. 1 unless neighbours were merged. */
  count: number;
}

/** Ranked most severe first, so a merged tick takes its worst level. */
const SEVERITY = ["fatal", "critical", "severe", "error", "warn", "warning", "info", "debug", "trace"];

function severityOf(level: string): number {
  const i = SEVERITY.indexOf(level);
  return i === -1 ? SEVERITY.length : i;
}

/**
 * @param measurements alerts with their pixel offset within the content
 * @param contentHeight total scrollable content height, in pixels
 * @param gutterHeight rendered height of the gutter, in pixels — merging is
 *   judged in gutter space, since that's where overlap actually happens
 * @param minGapPx smallest visually distinguishable gap between two ticks
 */
export function buildTicks(
  measurements: AlertMeasurement[],
  contentHeight: number,
  gutterHeight: number,
  minGapPx = 6,
): GutterTick[] {
  // A zero height means the content hasn't been laid out yet. Dividing by it
  // would put every tick at the top, which reads as real data rather than as
  // "not measured".
  if (contentHeight <= 0 || gutterHeight <= 0 || measurements.length === 0) return [];

  const sorted = [...measurements].sort((a, b) => a.top - b.top);
  const ticks: GutterTick[] = [];

  for (const m of sorted) {
    const offset = Math.min(Math.max(m.top / contentHeight, 0), 1);
    const last = ticks[ticks.length - 1];

    if (last && (offset - last.offset) * gutterHeight < minGapPx) {
      // Merge into the previous tick. It keeps the topmost alert as its scroll
      // target — clicking should land you at the start of the cluster, not in
      // the middle of it — but takes the worst level, so one error inside a run
      // of warnings still colours the marker.
      last.count += 1;
      if (severityOf(m.level) < severityOf(last.level)) last.level = m.level;
      continue;
    }

    ticks.push({ offset, alertId: m.alertId, level: m.level, headline: m.headline, count: 1 });
  }

  return ticks;
}
