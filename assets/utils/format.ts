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
