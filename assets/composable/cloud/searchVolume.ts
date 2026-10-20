export interface VolumeBucket {
  /** Bucket start, unix nanoseconds. */
  ts: number;
  level: string;
  count: number;
}

export interface VolumeColumn {
  /** Bucket start, unix milliseconds. */
  start: number;
  error: number;
  warn: number;
  /** Every other level, which the chart draws as one neutral band. */
  other: number;
  total: number;
}

// Past this many columns the bars are thinner than a pixel at page width.
const MAX_COLUMNS = 400;

/**
 * Lay Cloud's sparse (bucket, level) counts out as one column per bucket, with the
 * empty buckets filled in so a quiet hour reads as a gap instead of vanishing. The
 * axis runs from the first bucket with anything in it to the bucket holding `now`:
 * the search window is 14 days, but a plan that keeps a day of logs would otherwise
 * draw thirteen days of nothing.
 */
export function volumeColumns(buckets: VolumeBucket[], stepNs: number, now: number): VolumeColumn[] {
  if (!buckets.length || stepNs <= 0) return [];
  const step = stepNs / 1e6;
  const first = Math.min(...buckets.map((b) => b.ts / 1e6));
  const last = Math.max(first, ...buckets.map((b) => b.ts / 1e6), Math.floor(now / step) * step);
  const count = Math.min(MAX_COLUMNS, Math.floor((last - first) / step) + 1);
  const origin = last - (count - 1) * step;

  const columns: VolumeColumn[] = Array.from({ length: count }, (_, i) => ({
    start: origin + i * step,
    error: 0,
    warn: 0,
    other: 0,
    total: 0,
  }));
  for (const bucket of buckets) {
    const i = Math.round((bucket.ts / 1e6 - origin) / step);
    if (i < 0 || i >= count) continue;
    const column = columns[i];
    column[bandOf(bucket.level)] += bucket.count;
    column.total += bucket.count;
  }
  return columns;
}

function bandOf(level: string): "error" | "warn" | "other" {
  switch (level.toLowerCase()) {
    case "error":
    case "fatal":
      return "error";
    case "warn":
    case "warning":
      return "warn";
    default:
      return "other";
  }
}
