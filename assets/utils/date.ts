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
 * A day heading: "Today", "Yesterday", the weekday for the rest of the past week,
 * then a short date, with the year only when it is not this one. Every word comes
 * from Intl, so it is translated without a locale key.
 */
export function dayLabel(date: Date, locale: string | undefined, now: Date = new Date()): string {
  // Rounded, because a day across a DST change is 23 or 25 hours long.
  const diff = Math.round((startOfDay(date) - startOfDay(now)) / 86400000);
  let label: string;
  if (diff === 0 || diff === -1) {
    label = new Intl.RelativeTimeFormat(locale, { numeric: "auto" }).format(diff, "day");
  } else if (diff > -7 && diff < 0) {
    label = new Intl.DateTimeFormat(locale, { weekday: "long" }).format(date);
  } else {
    label = new Intl.DateTimeFormat(locale, {
      weekday: "short",
      month: "short",
      day: "numeric",
      ...(date.getFullYear() === now.getFullYear() ? {} : { year: "numeric" }),
    }).format(date);
  }
  // RelativeTimeFormat answers in lowercase ("today") and this starts a heading.
  return label.charAt(0).toLocaleUpperCase(locale) + label.slice(1);
}

/** Splits items into local calendar days, keeping their order inside each day. */
export function groupByDay<T>(
  items: readonly T[],
  dateOf: (item: T) => Date,
  locale: string | undefined,
  now: Date = new Date(),
): { key: number; label: string; items: T[] }[] {
  const groups = new Map<number, { key: number; label: string; items: T[] }>();
  for (const item of items) {
    const date = dateOf(item);
    const key = startOfDay(date);
    let group = groups.get(key);
    if (!group) {
      group = { key, label: dayLabel(date, locale, now), items: [] };
      groups.set(key, group);
    }
    group.items.push(item);
  }
  return [...groups.values()];
}
