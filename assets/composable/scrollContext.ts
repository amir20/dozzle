type ScrollContext = {
  loading: boolean;
  paused: boolean;
};

// export for testing
export const scrollContextKey = Symbol("scrollContext") as InjectionKey<ScrollContext>;

export const provideScrollContext = () => {
  const context = reactive({
    loading: false,
    paused: false,
  });
  provide(scrollContextKey, context);
  return context;
};

export const useScrollContext = () => {
  const context = inject(scrollContextKey);
  if (!context) {
    throw new Error("No scroll context provided");
  }
  return toRefs(context);
};
