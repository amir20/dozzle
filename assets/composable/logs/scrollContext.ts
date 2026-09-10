type ScrollContext = {
  paused: boolean;
  progress: number;
  /** Whether anything actually computes `progress`. Only a single-container
   * view can place a log on its container's lifetime, so merged, stack, host
   * and group views leave this false and the readout hides the position. */
  available: boolean;
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
    available: false,
    currentDate: new Date(),
  });
}
