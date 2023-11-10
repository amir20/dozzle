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
