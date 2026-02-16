export function formatDuration(seconds: number, locale: string | undefined): string {
  const hours = Math.floor(seconds / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  const secs = seconds % 60;
  const duration = { hours, minutes, seconds: secs };

  if (typeof Intl !== "undefined" && "DurationFormat" in Intl) {
    // @ts-expect-error DurationFormat is not yet in all TS lib types
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

export function toRelativeTime(date: Date, locale: string | undefined): string {
  const diffInSeconds = (date.getTime() - new Date().getTime()) / 1000;
  const rtf = new Intl.RelativeTimeFormat(locale, { numeric: "auto" });

  for (const [unit, seconds] of units) {
    const value = Math.round(diffInSeconds / seconds);
    if (Math.abs(value) >= 1) {
      return rtf.format(value, unit);
    }
  }

  return rtf.format(0, "second");
}
