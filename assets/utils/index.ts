export function formatBytes(bytes: number, decimals = 2) {
  if (bytes === 0) return "0 Bytes";
  const k = 1024;
  const dm = decimals < 0 ? 0 : decimals;
  const sizes = ["Bytes", "KB", "MB", "GB", "TB", "PB", "EB", "ZB", "YB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(dm)) + " " + sizes[i];
}

export function getDeep(obj: Record<string, any>, path: string[]) {
  return path.reduce((acc, key) => acc?.[key], obj);
}

export function isObject(value: any): value is Record<string, any> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

export function flattenJSON(obj: Record<string, any>, path: string[] = []) {
  const result: Record<string, any> = {};
  Object.keys(obj).forEach((key) => {
    const value = obj[key];
    const newPath = path.concat(key);
    if (isObject(value)) {
      Object.assign(result, flattenJSON(value, newPath));
    } else {
      result[newPath.join(".")] = value;
    }
  });
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
