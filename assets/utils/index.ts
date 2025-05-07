export function formatBytes(
  bytes: number,
  { decimals = 2, short = false }: { decimals?: number; short?: boolean } = { decimals: 2, short: false },
) {
  if (bytes === 0) return "0 Bytes";
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

export function getDeep(obj: Record<string, any>, path: string[]) {
  return path.reduce((acc, key) => acc?.[key], obj);
}

export function isObject(value: any): value is Record<string, any> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

export function flattenJSON(obj: Record<string, any>, path: string[] = []) {
  const map = flattenJSONToMap(obj);
  const result = {} as Record<string, any>;
  for (const [key, value] of map) {
    result[key.join(".")] = value;
  }
  return result;
}

export function flattenJSONToMap(obj: Record<string, any>, path: string[] = []): Map<string[], any> {
  const result = new Map<string[], any>();
  for (const key of Object.keys(obj)) {
    const value = obj[key];
    const newPath = path.concat(key);
    if (isObject(value)) {
      for (const [k, v] of flattenJSONToMap(value, newPath)) {
        result.set(k, v);
      }
    } else {
      result.set(newPath, value);
    }
  }

  return result;
}

export function arrayEquals(a: string[], b: string[]): boolean {
  return Array.isArray(a) && Array.isArray(b) && a.length === b.length && a.every((val, index) => val === b[index]);
}

export function stripVersion(label: string) {
  const [name, _] = label.split(":");
  return name;
}

export function useExponentialMovingAverage<T extends Record<string, number>>(source: Ref<T>, alpha: number = 0.2) {
  const ema = ref<T>(source.value) as Ref<T>;

  watch(source, (value) => {
    const newValue = {} as Record<string, number>;
    for (const key in value) {
      newValue[key] = alpha * value[key] + (1 - alpha) * ema.value[key];
    }
    ema.value = newValue as T;
  });

  return ema;
}

interface UseSimpleRefHistoryOptions<T> {
  capacity: number;
  deep?: boolean;
  initial?: T[];
}

export function useSimpleRefHistory<T>(source: Ref<T>, options: UseSimpleRefHistoryOptions<T>) {
  const { capacity, deep = true, initial = [] as T[] } = options;
  const history = ref<T[]>(initial) as Ref<T[]>;

  watch(
    source,
    (value) => {
      history.value.push(value);
      if (history.value.length > capacity) {
        history.value.shift();
      }
    },
    { deep },
  );

  const reset = ({ initial = [] }: Pick<UseSimpleRefHistoryOptions<T>, "initial">) => {
    history.value = initial;
  };

  return { history, reset };
}

export function hashCode(str: string) {
  let hash = 0;
  for (let i = 0; i < str.length; i++) {
    hash = (hash << 5) - hash + str.charCodeAt(i);
    hash |= 0;
  }
  return hash;
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
