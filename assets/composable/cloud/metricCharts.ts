import type { BarDataPoint } from "@/components/ui/BarChart.vue";
import { formatBytes, formatRate } from "@/utils/format";

export type Readout = { main: string; aside: string };
export type Chart = {
  key: string;
  label: string;
  colorClass: string;
  data: BarDataPoint[];
  // A fixed top for the bars (100 when they are percent of a limit); left
  // unset, the chart scales to its own peak.
  ceiling?: number;
  // The window's average, and the limit the bars are drawn against when there
  // is one.
  summary: string;
  limit?: string;
  // What the header says with the pointer elsewhere, and on a hovered bar.
  rest: Readout;
  at: (span: [number, number]) => Readout;
};

// What every chart in one panel shares: the translator, the bucket width in ms,
// and the window's name for the I/O totals.
export type ChartContext = { t: (key: string) => string; bucket: number; windowLabel: string };

const bytes = (v: number) => formatBytes(v, { short: true, decimals: 1 });
const mean = (xs: number[], [from, to]: [number, number]) =>
  xs.slice(from, to).reduce((a, b) => a + b, 0) / Math.max(1, to - from);

// A level (CPU, memory) rests on its peak: the highest sample cloud saw, not the
// highest average, which shrank as the buckets widened and read a week lower than
// the day inside it. An older cloud sends no max, and the average stands in.
export function levelChart<P>(
  points: P[],
  { t }: ChartContext,
  key: string,
  label: string,
  colorClass: string,
  format: (v: number) => string,
  avgOf: (p: P) => number,
  maxOf: (p: P) => number | undefined,
  // Draws the bars as a share of a limit instead of against their own peak:
  // each point's height (0-100) and, when it has a max, the max's height.
  share?: { of: (p: P) => { percent: number; peakPercent?: number }; label: string },
): Chart {
  const values = points.map(avgOf);
  const max = points.map(maxOf);
  const peaks = values.map((v, i) => max[i] ?? v);
  const hasMax = max.some((m) => m !== undefined);
  const peakWord = t("cloud-rail.metrics-peak");
  const heights = share ? points.map(share.of) : undefined;
  return {
    key,
    label,
    colorClass,
    data: values.map((v, i) => {
      const percent = heights ? heights[i].percent : v;
      if (!hasMax) return { percent, value: v };
      return {
        percent,
        value: v,
        peakPercent: heights ? (heights[i].peakPercent ?? percent) : peaks[i],
        peak: peaks[i],
      };
    }),
    ceiling: share ? 100 : undefined,
    // The header already carries the peak, so the summary adds the average.
    summary: `${t("analytics.avg")} ${format(mean(values, [0, values.length]))}`,
    limit: share?.label,
    rest: { main: format(Math.max(0, ...peaks)), aside: ` / ${peakWord}` },
    at: (span) => ({
      main: format(mean(values, span)),
      aside: hasMax ? ` · ${peakWord} ${format(Math.max(...peaks.slice(...span)))}` : "",
    }),
  };
}

// I/O rests on what moved over the whole window: a peak rate says little about
// a counter, and the total is the number people ask for. A bar is both
// directions together; the readout splits them, out first.
export function ioChart<P>(
  points: P[],
  { t, bucket, windowLabel }: ChartContext,
  key: string,
  label: string,
  colorClass: string,
  inOf: (p: P) => number | undefined,
  outOf: (p: P) => number | undefined,
  [outMark, inMark]: [string, string] = ["↑", "↓"],
): Chart | undefined {
  // Absent on an older cloud: hidden, rather than a flat line that reads as idle.
  if (!points.some((p) => inOf(p) !== undefined || outOf(p) !== undefined)) return undefined;
  const ins = points.map((p) => inOf(p) ?? 0);
  const outs = points.map((p) => outOf(p) ?? 0);
  const total = (xs: number[]) => xs.reduce((a, b) => a + b, 0) * (bucket / 1000);
  return {
    key,
    label,
    colorClass,
    data: ins.map((v, i) => ({ percent: v + outs[i], value: v + outs[i] })),
    summary: `${t("analytics.avg")} ${outMark}${formatRate(mean(outs, [0, outs.length]))} ${inMark}${formatRate(mean(ins, [0, ins.length]))}`,
    rest: { main: `${outMark}${bytes(total(outs))} ${inMark}${bytes(total(ins))}`, aside: ` / ${windowLabel}` },
    at: (span) => ({
      main: `${outMark}${formatRate(mean(outs, span))} ${inMark}${formatRate(mean(ins, span))}`,
      aside: "",
    }),
  };
}

// Said out loud so a flat line reads as an average rather than a missing sample.
export function formatBucket(ms: number): string {
  const seconds = Math.round(ms / 1000);
  if (seconds < 60) return `${seconds}s`;
  if (seconds < 3600) return `${Math.round(seconds / 60)}m`;
  return `${Math.round(seconds / 3600)}h`;
}
