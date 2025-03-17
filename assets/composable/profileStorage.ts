import { Profile } from "@/stores/config";

interface SerializerTransformer<T, U> {
  from: (raw: U) => T;
  to: (value: T) => U;
}

export function useProfileStorage<K extends keyof Profile>(
  key: K,
  defaultValue: NonNullable<Profile[K]>,
  transformer?: SerializerTransformer<NonNullable<Profile[K]>, any>,
) {
  const storageKey = "DOZZLE_" + key.toUpperCase();
  const storage = useStorage<NonNullable<Profile[K]>>(storageKey, defaultValue, undefined, {
    writeDefaults: false,
    mergeDefaults: true,
    serializer: transformer
      ? {
          read: (raw) => transformer.from(JSON.parse(raw)),
          write: (value) => JSON.stringify(transformer.to(value)),
        }
      : undefined,
    onError: (e) => {
      console.error(`Failed to read ${storageKey} from storage`, e);
    },
  });

  if (config.profile?.[key]) {
    if (transformer) {
      storage.value = transformer.from(config.profile[key]);
    } else if (storage.value instanceof Set && config.profile[key] instanceof Array) {
      storage.value = new Set(config.profile[key] as Iterable<any>) as unknown as NonNullable<Profile[K]>;
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
        const transformedValue = transformer ? transformer.to(value) : value;

        fetch(withBase("/api/profile"), {
          method: "PATCH",
          body: JSON.stringify({ [key]: transformedValue }, (_, value) => {
            if (value instanceof Set) {
              return Array.from(value);
            } else if (value instanceof Map) {
              return Array.from(value.entries());
            } else {
              return value;
            }
          }),
        });
      },
      { deep: true },
    );
  }

  return storage;
}
