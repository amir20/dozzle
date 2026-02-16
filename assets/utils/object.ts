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
