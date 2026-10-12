import type { Container } from "@/models/Container";
import { allLevels } from "./logContext";
import { appendRangeParams } from "./timeRange";

// A merged view can hold a whole stack; this keeps a copy from opening one
// connection per container at once.
export const COPY_CONCURRENCY = 4;

export type CopyBlock = { name: string; text: string };

/**
 * One block per container, each under a `── name ──` header, separated by a blank line.
 * A single container is copied as the endpoint returned it, with no header, so the
 * one-container copy stays exactly what it always was.
 */
export function formatCopyBlocks(blocks: CopyBlock[]): string {
  if (blocks.length === 1) return blocks[0].text;
  return blocks
    .map(({ name, text }) => {
      const body = text.replace(/\n+$/, "");
      return `── ${name} ──\n${body ? body + "\n" : ""}`;
    })
    .join("\n");
}

/** Runs `fn` over `items` with at most `limit` in flight, keeping results in input order. */
export async function mapLimit<T, R>(items: T[], limit: number, fn: (item: T) => Promise<R>): Promise<R[]> {
  const results: R[] = new Array(items.length);
  let next = 0;
  let failed = false;
  async function worker() {
    while (!failed && next < items.length) {
      const i = next++;
      try {
        results[i] = await fn(items[i]);
      } catch (err) {
        failed = true;
        throw err;
      }
    }
  }
  await Promise.all(Array.from({ length: Math.min(limit, items.length) }, worker));
  return results;
}

/**
 * Copies the logs of every container in `containers`, in that order, with the view's
 * streams, levels, search and time range. `fallback` is offered on the notice when the
 * clipboard cannot be reached, e.g. a download of the same logs.
 */
export function useCopyLogs(containers: MaybeRefOrGetter<Container[]>, fallback?: () => void) {
  const { streamConfig, levels, timeRange } = useLoggingContext();
  const { appliedSearchFilter } = useSearchFilter();
  const { copyLazy } = useCopy();
  const { t } = useI18n();
  const { showToast, removeToast } = useToast();

  function logsUrl(container: Container) {
    const params = new URLSearchParams();
    if (streamConfig.value.stdout) params.append("stdout", "1");
    if (streamConfig.value.stderr) params.append("stderr", "1");
    params.append("everything", "1");
    appendRangeParams(params, timeRange.value);

    if (appliedSearchFilter.value) {
      params.append("filter", appliedSearchFilter.value);
    }

    const selectedLevels = Array.from(levels.value);
    if (selectedLevels.length > 0 && selectedLevels.length < allLevels.length) {
      selectedLevels.forEach((level) => params.append("levels", level));
    }

    return withBase(`/api/hosts/${container.host}/containers/${container.id}/logs?${params.toString()}`);
  }

  async function fetchText(container: Container) {
    const response = await fetch(logsUrl(container), { headers: { Accept: "text/plain" } });
    if (!response.ok) throw new Error(response.statusText);
    return response.text();
  }

  async function fetchAll(list: Container[]) {
    const blocks = await mapLimit(list, COPY_CONCURRENCY, async (container) => {
      try {
        return { name: container.name, text: await fetchText(container) };
      } catch (err) {
        // In a merged view, say which container it was.
        if (list.length > 1 && err instanceof Error) err.message = `${container.name}: ${err.message}`;
        throw err;
      }
    });
    return formatCopyBlocks(blocks);
  }

  async function copyLogs() {
    const list = [...toValue(containers)];
    const toastId = "copy-logs";
    showToast(
      {
        id: toastId,
        title: t("toolbar.copying-logs"),
        message: "",
        type: "info",
      },
      { once: true },
    );

    // The success toast belongs to the copy, not to the download: on http the
    // clipboard is out of reach, and the logs arriving is not the same as them
    // landing somewhere the user can paste. A failed download has already been
    // reported by then, so it is swallowed here.
    await copyLazy(
      () =>
        fetchAll(list)
          .then((text) => {
            removeToast(toastId);
            return text;
          })
          .catch((err) => {
            removeToast(toastId);
            showToast(
              {
                title: "Error",
                message: escapeHtml(err.message),
                type: "error",
              },
              { expire: 5000 },
            );
            throw err;
          }),
      // A browser only allows the fallback copy for a few seconds after the click,
      // which a long download can outlast, so the notice offers the same content as
      // a file rather than dead-ending.
      { action: fallback ? { label: t("toolbar.download"), handler: fallback } : undefined },
    ).catch(() => {});
  }

  return { copyLogs };
}

/** Starts a browser download of `url`, as the toolbar's download row does. */
export function downloadFrom(url: string) {
  const link = document.createElement("a");
  link.href = url;
  link.download = "";
  link.click();
}
