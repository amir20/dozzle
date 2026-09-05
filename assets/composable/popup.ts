const show = ref(false);
const debouncedShow = debouncedRef(show, 1000);

const delayedShow = computed({
  set(newVal: boolean) {
    show.value = newVal;
  },
  get() {
    return debouncedShow.value;
  },
});

export const globalShowPopup = () => delayedShow;

// Identifies the popup the pointer is currently on. Opening one closes any other
// immediately, so the grace period below only ever covers the gap between a trigger
// and its own content, never the move to a neighboring row.
const active = ref<symbol | null>(null);

export const activePopup = () => active;
