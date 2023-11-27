import { Profile } from "@/stores/config";

export function usePersistedStorage<T>(key: keyof Profile, defaultValue: T) {
  const storageKey = "DOZZLE_" + key.toUpperCase();
  const storage = useStorage(storageKey, defaultValue);

  if (defaultValue instanceof Object && !(defaultValue instanceof Array) && !(defaultValue instanceof Set)) {
    storage.value = { ...defaultValue, ...storage.value };
  }

  if (config.profile?.[key]) {
    if (storage.value instanceof Set) {
      storage.value = new Set(config.profile[key] as Array<any>) as T;
    } else if (storage.value instanceof Array) {
      storage.value = config.profile[key] as T;
    } else if (storage.value instanceof Object) {
      storage.value = { ...storage.value, ...config.profile[key] };
    } else {
      storage.value = config.profile[key] as T;
    }
  }

  if (config.user) {
    watch(
      storage,
      (value) => {
        fetch(withBase("/api/profile"), {
          method: "PATCH",
          body: JSON.stringify({ [key]: value }, (_, value) => (value instanceof Set ? [...value] : value)),
        });
      },
      { deep: true },
    );
  }

  return storage;
}
