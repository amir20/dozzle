const RELATIVE_TIME_INTERVAL = 30_000;

// One timer for every relative timestamp on the page, deliberately at module scope.
// RelativeTime renders twice per container table row, so a useIntervalFn per component
// meant N unaligned wakeups and N separate render passes every half minute. Sharing the
// tick collapses those into one that Vue batches.
const tick = ref(0);

useIntervalFn(() => tick.value++, RELATIVE_TIME_INTERVAL);

/**
 * Reactive counter that advances every 30s. Read it inside a computed to have that
 * computed re-evaluate on each tick.
 */
export const relativeTimeTick = readonly(tick);
