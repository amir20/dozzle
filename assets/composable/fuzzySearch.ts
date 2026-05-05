// Shared open state for the global Cmd+K fuzzy-search modal.
// Lives outside the layout so any surface (home page hero, mobile menu,
// sidebar trigger) can open the same modal without prop-drilling.
const open = ref(false);

export function useFuzzySearch() {
  return {
    open,
    openSearch: () => {
      open.value = true;
    },
    closeSearch: () => {
      open.value = false;
    },
  };
}
