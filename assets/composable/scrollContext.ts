type ScrollContext = {
  paused: boolean;
  progress: number;
  currentDate: Date;
};

// export for testing
export const scrollContextKey = Symbol("scrollContext") as InjectionKey<ScrollContext>;

export const provideScrollContext = () => {
  const context = defaultValue();
  provide(scrollContextKey, context);
  return context;
};

export const useScrollContext = () => {
  const context = inject(scrollContextKey, defaultValue());
  return toRefs(context);
};

function defaultValue() {
  return reactive({
    paused: false,
    progress: 1,
    currentDate: new Date(),
  });
}
