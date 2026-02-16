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
