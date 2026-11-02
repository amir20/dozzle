export function formatDuration(seconds: number, locale: string | undefined): string {
  const hours = Math.floor(seconds / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  const secs = seconds % 60;
  const duration = { hours, minutes, seconds: secs };

  if (typeof Intl !== "undefined" && "DurationFormat" in Intl) {
    return new Intl.DurationFormat(locale, { style: "narrow" }).format(duration);
  }

  if (hours > 0) return `${hours}h ${minutes ? `${minutes}m` : ""}`.trim();
  if (minutes > 0) return `${minutes}m ${secs ? `${secs}s` : ""}`.trim();
  return `${secs}s`;
}

const units: [Intl.RelativeTimeFormatUnit, number][] = [
  ["year", 31536000],
  ["month", 2592000],
  ["week", 604800],
  ["day", 86400],
  ["hour", 3600],
  ["minute", 60],
  ["second", 1],
];

export function toRelativeTime(
  date: Date,
  locale: string | undefined,
  style: Intl.RelativeTimeFormatStyle = "long",
): string {
  const diffInSeconds = (date.getTime() - new Date().getTime()) / 1000;
  const rtf = new Intl.RelativeTimeFormat(locale, { numeric: "auto", style });

  for (const [unit, seconds] of units) {
    const value = Math.round(diffInSeconds / seconds);
    if (Math.abs(value) >= 1) {
      return rtf.format(value, unit);
    }
  }

  return rtf.format(0, "second");
}

/**
 * How long ago `date` was as a bare duration in its largest whole unit, "8 minutes"
 * rather than "8 minutes ago", for a sentence that supplies its own verb ("Up 8 minutes").
 */
export function toElapsed(date: Date, locale: string | undefined): string {
  const elapsed = Math.max(0, (Date.now() - date.getTime()) / 1000);
  for (const [unit, seconds] of units) {
    const value = Math.floor(elapsed / seconds);
    if (value >= 1) {
      return new Intl.NumberFormat(locale, { style: "unit", unit, unitDisplay: "long" }).format(value);
    }
  }
  return new Intl.NumberFormat(locale, { style: "unit", unit: "second", unitDisplay: "long" }).format(0);
}

function startOfDay(date: Date): number {
  return new Date(date.getFullYear(), date.getMonth(), date.getDate()).getTime();
}

/**
 * The heading an item files under: "Today", "Yesterday", the weekday for the rest of
 * the past week, then one "Last week" bucket. Anything older gets none and is hidden.
 * Every word comes from Intl, so it is translated without a locale key.
 */
export function dayBucket(
  date: Date,
  locale: string | undefined,
  now: Date = new Date(),
): { key: number; label: string } | undefined {
  // Rounded, because a day across a DST change is 23 or 25 hours long. Capped at 0 so
  // a timestamp a clock ahead of ours files under today rather than vanishing.
  const diff = Math.min(0, Math.round((startOfDay(date) - startOfDay(now)) / 86400000));
  const relative = new Intl.RelativeTimeFormat(locale, { numeric: "auto" });
  let key = diff;
  let label: string;
  if (diff >= -1) {
    label = relative.format(diff, "day");
  } else if (diff > -7) {
    label = new Intl.DateTimeFormat(locale, { weekday: "long" }).format(date);
  } else if (diff > -14) {
    key = -7;
    label = relative.format(-1, "week");
  } else {
    return undefined;
  }
  // RelativeTimeFormat answers in lowercase ("today") and this starts a heading.
  return { key, label: label.charAt(0).toLocaleUpperCase(locale) + label.slice(1) };
}

/** Splits items into day buckets, keeping their order inside each and dropping older ones. */
export function groupByDay<T>(
  items: readonly T[],
  dateOf: (item: T) => Date,
  locale: string | undefined,
  now: Date = new Date(),
): { key: number; label: string; items: T[] }[] {
  const groups = new Map<number, { key: number; label: string; items: T[] }>();
  for (const item of items) {
    const bucket = dayBucket(dateOf(item), locale, now);
    if (!bucket) continue;
    let group = groups.get(bucket.key);
    if (!group) {
      group = { ...bucket, items: [] };
      groups.set(bucket.key, group);
    }
    group.items.push(item);
  }
  return [...groups.values()];
}
