const searchQueryFilter = ref<string>("");
const debouncedSearchFilter = refDebounced(searchQueryFilter);
const showSearch = ref(false);
const inverseFilter = ref(false);

const searchParams = new URLSearchParams(window.location.search);
if (searchParams.get("search") !== null && searchParams.get("search") !== "") {
  searchQueryFilter.value = searchParams.get("search") || "";
  showSearch.value = true;
}
function resetSearch() {
  searchQueryFilter.value = "";
  showSearch.value = false;
  inverseFilter.value = false;
  appliedSearchFilter.value = "";
}

function toggleInverse() {
  inverseFilter.value = !inverseFilter.value;
}

function parses(query: string) {
  try {
    new RegExp(query);
    return true;
  } catch {
    return false;
  }
}

/**
 * The pattern the stream is actually filtered by. Half of a regex is still a keystroke on
 * the way to a whole one: `(` and `[` are what anyone types first, and the backend answers
 * a pattern it cannot compile with a 400, which tore the stream down and left the view
 * empty until the query closed. The last pattern that parsed stays applied instead, so the
 * matches on screen only ever move when the new one is something the server can run.
 */
const appliedSearchFilter = ref("");
watchEffect(() => {
  if (!showSearch.value) {
    appliedSearchFilter.value = "";
  } else if (parses(debouncedSearchFilter.value)) {
    appliedSearchFilter.value = debouncedSearchFilter.value;
  }
});

const isSearching = computed(() => showSearch.value && appliedSearchFilter.value !== "");

// The warning on the box tracks what is in the box, not what the stream is running, so it
// appears on the keystroke that breaks the pattern rather than a debounce later.
const isValidQuery = computed(() => parses(searchQueryFilter.value));

/**
 * Room the floating find box needs at the top of a log view so it does not open on the
 * first rows, which are the ones a search just put there. Zero once the box has been
 * dragged somewhere of the user's choosing, and zero while it is closed.
 */
const searchOverlayHeight = ref(0);

export function useSearchFilter() {
  return {
    searchQueryFilter,
    isValidQuery,
    debouncedSearchFilter,
    appliedSearchFilter,
    searchOverlayHeight,
    showSearch,
    resetSearch,
    isSearching,
    inverseFilter,
    toggleInverse,
  };
}
