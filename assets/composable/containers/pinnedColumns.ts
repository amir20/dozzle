import type { RouteLocationNormalizedLoaded } from "vue-router";

// A pipe reads better than a comma in a URL and does not collide with the comma that
// /merged/[ids] already uses for the containers merged into a single pane.
const SEPARATOR = "|";
const QUERY_KEY = "columns";

const parseColumns = (value: unknown) => (typeof value === "string" ? value.split(SEPARATOR).filter(Boolean) : []);

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

  // Deep, because pinning and unpinning mutate the array in place.
  watch(
    pinnedContainerIds,
    () => {
      if (columnsInUrl().join(SEPARATOR) !== pinned()) writeToUrl();
    },
    { deep: true },
  );

  // Columns outlive a navigation, so a route that lands without them gets them written back.
  // A route carrying its own set (a shared link, back/forward) wins instead, and replaying it
  // into the store leaves the URL already correct, so the watcher above stops there.
  const stopCarryingColumns = router.afterEach((to: RouteLocationNormalizedLoaded) => {
    const columns = columnsInUrl(to).join(SEPARATOR);
    if (columns === pinned()) return;
    if (columns) {
      pinnedContainerIds.value = columnsInUrl(to);
    } else {
      writeToUrl(to);
    }
  });

  onScopeDispose(stopCarryingColumns);
};
