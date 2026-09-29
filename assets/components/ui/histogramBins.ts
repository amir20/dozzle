export interface HistogramBin {
  start: number;
  end: number;
  count: number;
}

export interface Distribution {
  count: number;
  min: number;
  max: number;
  avg: number;
  p50: number;
  p95: number;
  bins: HistogramBin[];
}

const MIN_BINS = 5;
const MAX_BINS = 40;

/**
 * Linear interpolation between the two closest ranks, the same definition as
 * DuckDB's `quantile_cont`, so a number read here matches the one a query returns.
 */
export function quantile(sorted: ArrayLike<number>, q: number): number {
  const position = (sorted.length - 1) * q;
  const lower = Math.floor(position);
  const upper = Math.ceil(position);
  return sorted[lower] + (sorted[upper] - sorted[lower]) * (position - lower);
}

/**
 * Stats and equal-width bins for one numeric column. Bin width follows
 * Freedman-Diaconis, which sizes bins by the spread of the middle half, so a long
 * tail (the point of charting latency) does not squash the bulk into two bars.
 * Integer columns get whole-number widths, so no bin claims a range between two
 * values the column can never hold.
 */
export function distribution(values: Float64Array, integer = false): Distribution | null {
  const count = values.length;
  if (count === 0) return null;

  const sorted = values.slice().sort();
  const min = sorted[0];
  const max = sorted[count - 1];
  let sum = 0;
  for (const value of sorted) sum += value;

  const stats = { count, min, max, avg: sum / count, p50: quantile(sorted, 0.5), p95: quantile(sorted, 0.95) };
  if (min === max) return { ...stats, bins: [{ start: min, end: max, count }] };

  const range = max - min;
  const iqr = quantile(sorted, 0.75) - quantile(sorted, 0.25);
  // A zero IQR (most rows identical) has no spread to size by: fall back to the
  // Rice rule, which only looks at the row count.
  const ideal = iqr > 0 ? Math.ceil(range / ((2 * iqr) / Math.cbrt(count))) : Math.ceil(2 * Math.cbrt(count));
  let binCount = Math.min(MAX_BINS, Math.max(MIN_BINS, ideal));
  let width = range / binCount;
  if (integer) {
    width = Math.max(1, Math.ceil((range + 1) / binCount));
    binCount = Math.ceil((range + 1) / width);
  }

  const bins: HistogramBin[] = Array.from({ length: binCount }, (_, i) => ({
    start: min + i * width,
    // Inclusive for integers, so a width-1 bin over 3 reads "3", not "3 – 4".
    end: min + (i + 1) * width - (integer ? 1 : 0),
    count: 0,
  }));
  for (const value of sorted) {
    // The max lands exactly on the last edge; it belongs to the last bin.
    bins[Math.min(binCount - 1, Math.floor((value - min) / width))].count++;
  }

  return { ...stats, bins };
}
