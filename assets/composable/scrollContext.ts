type ScrollContext = {
  loading: boolean;
  paused: boolean;
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
  });
}
