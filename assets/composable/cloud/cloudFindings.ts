import type { MaybeRefOrGetter } from "vue";
import type { LogMoment } from "@/composable/logs/logJump";

export type FindingSeverity = "high" | "medium" | "low";
export type FindingStatus = "open" | "ongoing" | "resolved";
/** Dozzle's three views. new = open and ongoing, not ignored. */
export type FindingsFilter = "new" | "all" | "ignored";

export interface FindingContainer {
  host: string;
  id: string;
  name: string;
}

/** One finding from /api/cloud/findings. Mirrors cloud.Finding in Go. */
export interface CloudFinding {
  /** Opaque, sqids-encoded. Never a raw row id. */
  id: string;
  severity: FindingSeverity;
  status: FindingStatus;
  /** Not yet shown to the user by their plan. headline, why and fix are empty. */
  locked: boolean;
  headline: string;
  why?: string;
  /** Absent when locked, when fixLocked, or when the review wrote none. */
  fix?: string;
  /** The on-screen containers this finding is about. */
  containers: FindingContainer[];
  daysOpen: number;
  /** Unix nanoseconds, like CloudAlert.ts. */
  firstSeenAt: number;
  lastSeenAt: number;
  resolvedAt?: number;
  /** Oldest and newest log line. Absent on findings that never recorded them. */
  logFirstAt?: number;
  logLastAt?: number;
  ignoredAt?: number;
  /** The finding's page in Dozzle Cloud. */
  url?: string;
}

export interface CloudFindingsResult {
  findings: CloudFinding[];
  /** Plan gate decided in Cloud. Never infer it from a missing fix. */
  fixLocked: boolean;
  truncated?: boolean;
}

type Scope = { host: string; id: string }[];

/** "Show me the lines" for one finding on one container: its newest line, or
 *  the last scan when the review never recorded one. */
export function findingMoment(f: CloudFinding, c: FindingContainer): LogMoment {
  return { containerId: c.id, date: new Date((f.logLastAt || f.lastSeenAt) / 1e6) };
}

export function findingsUrl(containers: Scope, filter: FindingsFilter) {
  const params = new URLSearchParams();
  for (const c of containers) params.append("container", `${c.host}/${c.id}`);
  params.set("filter", filter);
  return withBase(`/api/cloud/findings?${params}`);
}

/** A finding has no whole-instance list in Dozzle (that lives in Cloud), so an
 *  empty scope asks nothing and gets nothing. */
export async function fetchFindings(containers: Scope, filter: FindingsFilter): Promise<CloudFindingsResult> {
  if (containers.length === 0) return { findings: [], fixLocked: false };
  const res = await fetch(findingsUrl(containers, filter));
  if (!res.ok) throw new Error(`findings: ${res.status}`);
  const body = await res.json();
  return { findings: body.findings ?? [], fixLocked: !!body.fixLocked, truncated: !!body.truncated };
}

/** Ignore a finding account-wide, or take that back. */
export async function setFindingIgnored(id: string, ignored: boolean) {
  const res = await fetch(withBase(`/api/cloud/findings/${encodeURIComponent(id)}/ignore`), {
    method: ignored ? "POST" : "DELETE",
  });
  return res.ok;
}

/** The newest finding this reader has opened the panel on, by first sighting.
 *  First rather than last seen: an ongoing finding is re-seen every morning, and
 *  the dot is for something new, not for something still there. */
// Per browser rather than per profile: a profile field is a server-side setting,
// and this is only a dot.
const lastSeenFinding = useLocalStorage("DOZZLE_LAST_SEEN_FINDING", 0);

export function useCloudFindings(filter: MaybeRefOrGetter<FindingsFilter>) {
  const { containers } = useCloudScope();
  const { t } = useI18n();
  const { showToast } = useToast();

  const findings = ref<CloudFinding[]>([]);
  const fixLocked = ref(false);
  const truncated = ref(false);
  const loading = ref(false);
  const failed = ref(false);

  let request = 0;
  async function load() {
    const mine = ++request;
    loading.value = true;
    failed.value = false;
    try {
      const result = await fetchFindings(containers.value, toValue(filter));
      if (mine !== request) return;
      findings.value = result.findings;
      fixLocked.value = result.fixLocked;
      truncated.value = !!result.truncated;
    } catch {
      if (mine !== request) return;
      findings.value = [];
      failed.value = true;
    } finally {
      if (mine === request) loading.value = false;
    }
  }

  watch(() => [toValue(filter), containers.value.map((c) => `${c.host}/${c.id}`).join(",")], load, { immediate: true });

  /** Ignored leaves New and All, and un-ignored leaves Ignored, so either way
   *  the row goes; the toast puts it back if the reader changes their mind. */
  async function toggleIgnored(finding: CloudFinding, ignored: boolean) {
    const index = findings.value.indexOf(finding);
    if (index === -1) return;
    findings.value.splice(index, 1);
    if (!(await setFindingIgnored(finding.id, ignored))) {
      findings.value.splice(index, 0, finding);
      showToast({ type: "error", message: t("cloud-findings.ignore-failed") }, { expire: 5000 });
      return;
    }
    if (ignored) {
      showToast(
        {
          type: "info",
          message: t("cloud-findings.ignored"),
          action: { label: t("cloud-findings.undo"), handler: () => restore(finding, index) },
        },
        { expire: 8000 },
      );
    }
  }

  async function restore(finding: CloudFinding, index: number) {
    if (await setFindingIgnored(finding.id, false)) findings.value.splice(index, 0, finding);
  }

  function markSeen() {
    const newest = Math.max(0, ...findings.value.map((f) => f.firstSeenAt));
    if (newest > lastSeenFinding.value) lastSeenFinding.value = newest;
  }

  return {
    findings,
    fixLocked,
    truncated,
    loading,
    failed,
    reload: load,
    ignore: (f: CloudFinding) => toggleIgnored(f, true),
    unignore: (f: CloudFinding) => toggleIgnored(f, false),
    markSeen,
  };
}

/** Whether the scope on screen has a new finding first seen after the reader
 *  last opened the panel. Asked once per scope, for the tab's dot. */
export function useUnseenFindings(enabled: MaybeRefOrGetter<boolean> = true) {
  const { containers } = useCloudScope();
  const newest = ref(0);

  watch(
    () => (toValue(enabled) ? containers.value.map((c) => `${c.host}/${c.id}`).join(",") : ""),
    async (key) => {
      newest.value = 0;
      if (!key) return;
      try {
        const { findings } = await fetchFindings(containers.value, "new");
        newest.value = Math.max(0, ...findings.map((f) => f.firstSeenAt));
      } catch {
        // No dot is the honest answer when the question could not be asked.
      }
    },
    { immediate: true },
  );

  return computed(() => newest.value > lastSeenFinding.value);
}
