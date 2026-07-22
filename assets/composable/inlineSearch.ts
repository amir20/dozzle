// Tracks whether the current page already renders an inline container-search
// bar (the dashboard topbar does, via PageWithLinks). The sidebar reads this
// to avoid showing a duplicate "Search containers" bar on those pages.
const count = ref(0);

// Call from a component that renders an inline search bar. Registers while the
// component is mounted so `hasInlineSearch` reflects the current page.
export function useInlineSearchProvider() {
  onMounted(() => count.value++);
  onUnmounted(() => count.value--);
}

export const hasInlineSearch = computed(() => count.value > 0);
