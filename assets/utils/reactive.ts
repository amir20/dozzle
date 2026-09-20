export function useExponentialMovingAverage<T extends Record<string, number>>(source: Ref<T>, alpha: number = 0.2) {
  const ema = ref<T>(source.value) as Ref<T>;

  watch(source, (value) => {
    const newValue = {} as Record<string, number>;
    for (const key in value) {
      newValue[key] = alpha * value[key] + (1 - alpha) * ema.value[key];
    }
    ema.value = newValue as T;
  });

  return { movingAverage: ema, reset: (value: T) => (ema.value = value) };
}

interface UseSimpleRefHistoryOptions<T> {
  capacity: number;
  deep?: boolean;
  initial?: T[];
}

export function useSimpleRefHistory<T>(source: Ref<T>, options: UseSimpleRefHistoryOptions<T>) {
  const { capacity, deep = true, initial = [] as T[] } = options;
  // Shallow, and pushed into in place: a deep `ref` would proxy the window and
  // every entry in it, and the charts downstream read all `capacity` of them back
  // once a second. Nothing edits an entry after it lands, so `triggerRef` below is
  // the whole of the tracking this needs. Callers must treat `history` as read-only.
  const history = shallowRef<T[]>(initial) as Ref<T[]>;

  watch(
    source,
    (value) => {
      history.value.push(value);
      if (history.value.length > capacity) {
        history.value.shift();
      }
      triggerRef(history);
    },
    { deep },
  );

  const reset = ({ initial = [] }: Pick<UseSimpleRefHistoryOptions<T>, "initial">) => {
    history.value = initial;
  };

  return { history, reset };
}
