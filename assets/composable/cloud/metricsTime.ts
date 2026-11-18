import { hourStyle } from "@/stores/settings";

/** A hovered bar: its index and how many bars the chart drew. BarChart reports both. */
export type MetricsHover = { index: number; bars: number };

/**
 * The points behind a hovered bar. BarChart folds a series into equal runs of
 * ceil(points / bars), so this is the same run it averaged to draw the bar.
 */
export function hoverSpan(length: number, hover: MetricsHover): [number, number] {
  const size = Math.ceil(length / Math.max(1, hover.bars));
  const from = hover.index * size;
  return [from, Math.min(length, from + size)];
}

/** The middle point of a span: the honest moment for a bar that covers a run. */
export const spanMiddle = ([from, to]: [number, number]) => Math.floor((from + to - 1) / 2);

/**
 * How a metrics window names its times. Past a few hours the clock alone is
 * ambiguous ("09:14" on a week of bars could be any of seven mornings), so the
 * date comes along.
 */
export const windowClock = (window: string): Intl.DateTimeFormatOptions =>
  window === "1h" || window === "6h"
    ? { hour: "2-digit", minute: "2-digit" }
    : { month: "short", day: "numeric", hour: "2-digit", minute: "2-digit" };

/** A formatter for those times that follows the reader's own clock preference. */
export function useMetricsClock(options: MaybeRefOrGetter<Intl.DateTimeFormatOptions>) {
  const hourCycle = computed(() => (hourStyle.value === "12" ? "h12" : hourStyle.value === "24" ? "h23" : undefined));
  return computed(() => new Intl.DateTimeFormat(undefined, { ...toValue(options), hourCycle: hourCycle.value }));
}
