export function formatBytes(
  bytes: number,
  { decimals = 2, short = false }: { decimals?: number; short?: boolean } = { decimals: 2, short: false },
) {
  if (bytes === 0) return short ? "0B" : "0 Bytes";
  const k = 1024;
  const dm = decimals < 0 ? 0 : decimals;
  const sizes = ["Bytes", "KB", "MB", "GB", "TB", "PB", "EB", "ZB", "YB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));

  const value = parseFloat((bytes / Math.pow(k, i)).toFixed(dm));
  if (short) {
    return value + sizes[i].charAt(0);
  } else {
    return value + " " + sizes[i];
  }
}

// Bytes per second, at three significant figures at most so a fixed-width column
// can reserve the worst case ("1023K/s" is 7 characters).
export function formatRate(bytes: number) {
  // Below one byte the log would go negative and formatBytes would index past the
  // start of its unit table. A cloud bucket's rate is an average, so it can be.
  if (bytes < 1) return "0B/s";
  // Drop the decimal once the scaled value reaches three digits: "105K/s" reads
  // the same as "105.3K/s" at this size and costs two fewer characters.
  const scaled = bytes / 1024 ** Math.floor(Math.log(bytes) / Math.log(1024));
  return formatBytes(bytes, { short: true, decimals: scaled >= 100 ? 0 : 1 }) + "/s";
}

export function stripVersion(label: string) {
  const [name, _] = label.split(":");
  return name;
}

export function hashCode(str: string) {
  let hash = 0;
  for (let i = 0; i < str.length; i++) {
    hash = (hash << 5) - hash + str.charCodeAt(i);
    hash |= 0;
  }
  return hash;
}

/**
 * Escapes a string for safe inclusion in HTML. Needed wherever untrusted text
 * ends up inside a `v-html` binding, such as toast messages, since vue-i18n
 * does not escape interpolated values.
 */
export function escapeHtml(value: string) {
  return value
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#39;");
}
