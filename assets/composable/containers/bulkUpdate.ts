import { Container } from "@/models/Container";

export type BulkUpdateStatus =
  "queued" | "pulling" | "recreating" | "verifying" | "done" | "up-to-date" | "rolled-back" | "error";

// Mirrors bulkUpdateItem in internal/web/bulk_update.go.
export interface BulkUpdateItem {
  host: string;
  id: string;
  name: string;
  image: string;
  self?: boolean;
  status: BulkUpdateStatus;
  current?: number;
  total?: number;
  error?: string;
}

export interface BulkUpdateJob {
  trigger: "manual" | "schedule";
  startedAt: string;
  finishedAt?: string;
  items: BulkUpdateItem[];
  running: boolean;
}

export function isFinished(status: BulkUpdateStatus) {
  return status === "done" || status === "up-to-date" || status === "rolled-back" || status === "error";
}

// One job and one stream per tab, however many surfaces show it. The stream is
// only open while something is looking: the drawer, or a job this tab started
// and still has to report on.
const job = ref<BulkUpdateJob | null>(null);
let source: EventSource | null = null;
let holds = 0;

function hold() {
  holds++;
  if (!source) {
    source = new EventSource(withBase("/api/updates/stream"));
    source.addEventListener("bulk-update", (e) => {
      const next = JSON.parse((e as MessageEvent).data) as BulkUpdateJob;
      job.value = next.items ? next : null;
    });
  }

  let released = false;
  return () => {
    if (released) return;
    released = true;
    holds--;
    if (holds === 0) {
      source?.close();
      source = null;
    }
  };
}

export function useBulkUpdate() {
  const { t } = useI18n();
  const { showToast } = useToast();

  const running = computed(() => job.value?.running ?? false);

  async function start(containers: Container[]) {
    // Held from before the POST, so the first snapshot of the new job is not missed.
    const release = hold();
    let startedAt: string;
    try {
      const response = await fetch(withBase("/api/updates"), {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ containers: containers.map(({ host, id }) => ({ host, id })) }),
      });
      if (!response.ok) {
        release();
        showToast({
          type: "error",
          title: t("error.update-failed"),
          message: response.status === 409 ? t("updates.busy") : t("error.unable-to-update"),
        });
        return;
      }
      ({ startedAt } = (await response.json()) as { startedAt: string });
    } catch {
      release();
      showToast({ type: "error", title: t("error.update-failed"), message: t("error.something-went-wrong") });
      return;
    }

    // The stream may still be holding the previous, finished job.
    await until(() => job.value?.startedAt === startedAt && !!job.value.finishedAt).toBe(true, {
      // A Dozzle that went away mid-job comes back with no job at all.
      timeout: 30 * 60_000,
    });
    const items = job.value?.startedAt === startedAt ? job.value.items : [];
    release();
    if (items.length === 0) return;

    // A rolled back container is running its old version again: not updated.
    const failed = items.filter((i) => i.status === "error" || i.status === "rolled-back").length;
    const updated = items.filter((i) => i.status === "done").length;
    showToast(
      {
        type: failed ? "error" : "info",
        title: t("updates.title"),
        message: failed ? t("updates.summary-failed", { updated, failed }) : t("updates.summary", { updated }),
      },
      failed ? {} : { expire: 5000 },
    );

    // Dozzle went last and is swapping itself out now. The old process keeps
    // answering for a while, so only a reply after it went away counts.
    if (items.some((i) => i.self && i.status === "done")) {
      showToast({ type: "info", title: t("updates.title"), message: t("setup.update.restarting") }, { expire: 5000 });
      await useSetup().waitForRestart({ timeout: 180_000, mustGoDown: true });
    }
  }

  return { job: readonly(job), running, start, hold };
}
