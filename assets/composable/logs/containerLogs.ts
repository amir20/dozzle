import type { Container } from "@/models/Container";
import { allLevels } from "./logContext";
import { appendRangeParams } from "./timeRange";

const containerLogsPath = "/api/containers/logs";

/**
 * `/api/containers/logs` takes its params as a form body: a view of hundreds of
 * containers would pass a proxy's URL length limit as a query string.
 */
export function postContainerLogs(params: URLSearchParams, init: RequestInit = {}) {
  return fetch(withBase(containerLogsPath), { ...init, method: "POST", body: params });
}

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

  const isFiltered = computed(
    () =>
      !!appliedSearchFilter.value ||
      (levels.value.size > 0 && levels.value.size < allLevels.length) ||
      timeRange.value.kind !== "live",
  );

  // A form POST in this window: the zip's Content-Disposition makes it a download,
  // and the page stays where it is.
  function downloadLogs() {
    const form = document.createElement("form");
    form.method = "POST";
    form.action = withBase(containerLogsPath);
    form.hidden = true;
    const fields = new URLSearchParams(params.value);
    fields.append("format", "zip");
    const nameValue = toValue(name);
    if (nameValue) fields.append("name", nameValue);
    for (const [key, value] of fields) {
      const input = document.createElement("input");
      input.type = "hidden";
      input.name = key;
      input.value = value;
      form.append(input);
    }
    document.body.append(form);
    form.submit();
    form.remove();
  }

  async function fetchText(params: URLSearchParams) {
    const response = await postContainerLogs(params, { headers: { Accept: "text/plain" } });
    if (!response.ok) throw new Error((await response.text()).trim() || response.statusText);
    return response.text();
  }

  async function copyLogs() {
    if (toValue(containers).length === 0) return;
    const body = params.value;
    const toastId = "copy-logs";
    showToast({ id: toastId, title: t("toolbar.copying-logs"), message: "", type: "info" }, { once: true });

    // The success toast belongs to the copy, not to the download: on http the
    // clipboard is out of reach, and the logs arriving is not the same as them
    // landing somewhere the user can paste. A failed download has already been
    // reported by then, so it is swallowed here.
    await copyLazy(
      () =>
        fetchText(body)
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

  return { isFiltered, copyLogs, downloadLogs };
}
