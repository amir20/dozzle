import type { Container } from "@/models/Container";
import { allLevels } from "./logContext";
import { appendRangeParams } from "./timeRange";

/**
 * The whole log of the view's containers, as the view filters it: streams, levels,
 * search and time range. One request to `/api/containers/logs` serves both the copy
 * button (plain text, a header per container when there are several) and the
 * download (`format=zip`, a file per container). `name` names the zip.
 */
export function useContainerLogs(
  containers: MaybeRefOrGetter<Container[]>,
  name?: MaybeRefOrGetter<string | undefined>,
) {
  const { streamConfig, levels, timeRange } = useLoggingContext();
  const { appliedSearchFilter, inverseFilter } = useSearchFilter();
  const { copyLazy } = useCopy();
  const { t } = useI18n();
  const { showToast, removeToast } = useToast();
  const { enableDownload } = config;

  const params = computed(() => {
    const params = new URLSearchParams();
    for (const c of toValue(containers)) params.append("container", `${c.host}/${c.id}`);
    if (streamConfig.value.stdout) params.append("stdout", "1");
    if (streamConfig.value.stderr) params.append("stderr", "1");
    // What the view holds: a range, or the lines since its start.
    appendRangeParams(params, timeRange.value);
    if (appliedSearchFilter.value) {
      params.append("filter", appliedSearchFilter.value);
      if (inverseFilter.value) params.append("inverse", "true");
    }
    const selected = Array.from(levels.value);
    if (selected.length > 0 && selected.length < allLevels.length) {
      selected.forEach((level) => params.append("levels", level));
    }
    return params;
  });

  const downloadUrl = computed(() => {
    const zip = new URLSearchParams(params.value);
    zip.append("format", "zip");
    const nameValue = toValue(name);
    if (nameValue) zip.append("name", nameValue);
    return withBase(`/api/containers/logs?${zip}`);
  });

  const isFiltered = computed(
    () =>
      !!appliedSearchFilter.value ||
      (levels.value.size > 0 && levels.value.size < allLevels.length) ||
      timeRange.value.kind !== "live",
  );

  function downloadLogs() {
    const link = document.createElement("a");
    link.href = downloadUrl.value;
    link.download = "";
    link.click();
  }

  async function fetchText(url: string) {
    const response = await fetch(url, { headers: { Accept: "text/plain" } });
    if (!response.ok) throw new Error((await response.text()).trim() || response.statusText);
    return response.text();
  }

  async function copyLogs() {
    if (toValue(containers).length === 0) return;
    const url = withBase(`/api/containers/logs?${params.value}`);
    const toastId = "copy-logs";
    showToast({ id: toastId, title: t("toolbar.copying-logs"), message: "", type: "info" }, { once: true });

    // The success toast belongs to the copy, not to the download: on http the
    // clipboard is out of reach, and the logs arriving is not the same as them
    // landing somewhere the user can paste. A failed download has already been
    // reported by then, so it is swallowed here.
    await copyLazy(
      () =>
        fetchText(url)
          .then((text) => {
            removeToast(toastId);
            return text;
          })
          .catch((err) => {
            removeToast(toastId);
            showToast({ title: "Error", message: escapeHtml(err.message), type: "error" }, { expire: 5000 });
            throw err;
          }),
      // A browser only allows the fallback copy for a few seconds after the click,
      // which a long download can outlast, so the notice offers the same logs as a
      // file rather than dead-ending.
      { action: enableDownload ? { label: t("toolbar.download"), handler: downloadLogs } : undefined },
    ).catch(() => {});
  }

  return { downloadUrl, isFiltered, copyLogs, downloadLogs };
}
