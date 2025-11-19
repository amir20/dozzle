import { Container } from "@/models/Container";
import { allLevels } from "@/composable/logContext";

export function useDownloadUrl(
  containers: Ref<Container[]> | ComputedRef<Container[]>,
  streamConfig: { stdout: boolean; stderr: boolean } | Ref<{ stdout: boolean; stderr: boolean }>,
  levels: Ref<Set<string>>,
) {
  const { debouncedSearchFilter } = useSearchFilter();

  const downloadUrl = computed(() => {
    const params = new URLSearchParams();
    const config = toValue(streamConfig);

    // Add stdout/stderr
    if (config.stdout) params.append("stdout", "1");
    if (config.stderr) params.append("stderr", "1");

    // Add filter if search is active
    if (debouncedSearchFilter.value) {
      params.append("filter", debouncedSearchFilter.value);
    }

    // Add levels (multiple values) only if filtered
    const selectedLevels = Array.from(levels.value);
    if (selectedLevels.length > 0 && selectedLevels.length < allLevels.length) {
      selectedLevels.forEach((level) => params.append("levels", level));
    }

    const containerIds = toValue(containers)
      .map((c) => c.host + "~" + c.id)
      .join(",");

    return withBase(`/api/containers/${containerIds}/download?${params.toString()}`);
  });

  const isFiltered = computed(
    () => debouncedSearchFilter.value || (levels.value.size > 0 && levels.value.size < allLevels.length),
  );

  return {
    downloadUrl,
    isFiltered,
  };
}
