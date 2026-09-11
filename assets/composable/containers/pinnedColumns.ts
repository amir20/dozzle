import type { RouteLocationNormalizedLoaded } from "vue-router";

// A pipe reads better than a comma in a URL and does not collide with the comma that
// /merged/[ids] already uses for the containers merged into a single pane.
const SEPARATOR = "|";
const QUERY_KEY = "columns";

// Deduped: pinContainer() already refuses to pin the same container twice, and a link
// carrying `a|a` should not be able to say otherwise.
const parseColumns = (value: unknown) =>
  typeof value === "string" ? [...new Set(value.split(SEPARATOR).filter(Boolean))] : [];

/**
 * Keeps the pinned columns and the `?columns=a|b|c` query in step, in both directions.
 *
 * Call it once, from the layout that renders the columns, and before anything reads the store:
 * the first thing it does is seed the store from the URL. It cannot live in the store itself,
 * since reaching the router singleton from there closes an import cycle through the generated
 * layouts, and a store has no component context to inject the router from.
 */
export const usePinnedColumnsInUrl = () => {
  const router = useRouter();
  const { pinnedContainerIds } = storeToRefs(usePinnedLogsStore());

  const columnsInUrl = (route: RouteLocationNormalizedLoaded = router.currentRoute.value) =>
    parseColumns(route.query[QUERY_KEY]);
  // What the query literally says, which a link can spell differently from what it means
  // (`a|a`, a trailing separator, `|||`). Comparing against this rather than the parsed
  // set is what lets the URL be rewritten to the columns actually open.
  const rawColumns = (route: RouteLocationNormalizedLoaded = router.currentRoute.value) =>
    typeof route.query[QUERY_KEY] === "string" ? (route.query[QUERY_KEY] as string) : "";
  const pinned = () => pinnedContainerIds.value.join(SEPARATOR);

  const writeToUrl = (route: RouteLocationNormalizedLoaded = router.currentRoute.value) => {
    const { [QUERY_KEY]: _, ...query } = route.query;
    const columns = pinned();
    router.replace({
      path: route.path,
      hash: route.hash,
      query: columns ? { ...query, [QUERY_KEY]: columns } : query,
    });
  };

  pinnedContainerIds.value = columnsInUrl();
  if (rawColumns() !== pinned()) writeToUrl();

  // Deep, because pinning and unpinning mutate the array in place.
  watch(
    pinnedContainerIds,
    () => {
      if (rawColumns() !== pinned()) writeToUrl();
    },
    { deep: true },
  );

  // Columns outlive a navigation, so a route that lands without them gets them written back.
  // A route carrying its own set (a shared link, back/forward) wins instead, and is written
  // back only when the URL spells it differently from the columns it opened.
  const stopCarryingColumns = router.afterEach((to: RouteLocationNormalizedLoaded) => {
    const columns = columnsInUrl(to);
    if (columns.length && columns.join(SEPARATOR) !== pinned()) {
      pinnedContainerIds.value = columns;
    }
    if (rawColumns(to) !== pinned()) writeToUrl(to);
  });

  onScopeDispose(stopCarryingColumns);
};
