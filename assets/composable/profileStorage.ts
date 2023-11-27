import { Profile } from "@/stores/config";

export function useProfileStorage<K extends keyof Profile>(key: K, defaultValue: NonNullable<Profile[K]>) {
  const storageKey = "DOZZLE_" + key.toUpperCase();
  const storage = useStorage<NonNullable<Profile[K]>>(storageKey, defaultValue, undefined, {
    writeDefaults: false,
    mergeDefaults: true,
  });

  if (config.profile?.[key]) {
    if (storage.value instanceof Set && config.profile[key] instanceof Array) {
      storage.value = new Set([...(config.profile[key] as Iterable<any>)]) as unknown as NonNullable<Profile[K]>;
    } else if (config.profile[key] instanceof Array) {
      storage.value = config.profile[key] as NonNullable<Profile[K]>;
    } else if (config.profile[key] instanceof Object) {
      Object.assign(storage.value, config.profile[key]);
    } else {
      storage.value = config.profile[key] as NonNullable<Profile[K]>;
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
