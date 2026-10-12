import { allLevels } from "./logContext";

/**
 * What the Streams and Levels filters are set to, in the short form a collapsed
 * submenu or the filter chip's tooltip shows, so the reader can tell what they
 * are looking at without opening anything.
 */
export function useLogFilterSummary() {
  const { streamConfig, levels } = useLoggingContext();
  const { t } = useI18n();

  const streamSummary = computed(() => {
    if (streamConfig.value.stdout && streamConfig.value.stderr) return t("toolbar.all");
    if (streamConfig.value.stdout) return "STDOUT";
    if (streamConfig.value.stderr) return "STDERR";
    return t("toolbar.none");
  });

  const levelsFiltered = computed(() => levels.value.size !== allLevels.length);

  const levelSummary = computed(() =>
    levelsFiltered.value ? `${levels.value.size}/${allLevels.length}` : t("toolbar.all"),
  );

  return { streamSummary, levelSummary, levelsFiltered };
}
