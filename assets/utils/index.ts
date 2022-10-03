import { Container } from "@/models/Container";
import { useStorage } from "@vueuse/core";
import { computed, ComputedRef } from "vue";

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

export function persistentVisibleKeys(container: ComputedRef<Container>) {
  return computed(() => useStorage(stripVersion(container.value.image) + ":" + container.value.command, []));
}

export function stripVersion(label: string) {
  const [name, _] = label.split(":");
  return name;
}
