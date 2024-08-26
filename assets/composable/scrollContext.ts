type ScrollContext = {
  loading: boolean;
  paused: boolean;
  progress: number;
  currentDate: Date;
};

// export for testing
export const scrollContextKey = Symbol("scrollContext") as InjectionKey<ScrollContext>;

export const provideScrollContext = () => {
  const context = defauleValue();
  provide(scrollContextKey, context);
  return context;
};

export const useScrollContext = () => {
  const defaultValue = defauleValue();
  const context = inject(scrollContextKey, defaultValue);
  return toRefs(context);
};

function defauleValue() {
  return reactive({
    loading: false,
    paused: false,
    progress: 1,
    currentDate: new Date(),
  });
}
