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
