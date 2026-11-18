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
  /** The list stops short: Cloud hit its limit, or the scope named more
   *  containers than are looked up. */
  truncated?: boolean;
  /** Whether this reader may ignore a finding. Decided by the server with the
   *  same checks the ignore route enforces; never inferred here. */
  canIgnore?: boolean;
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
  if (containers.length === 0) return { findings: [], fixLocked: false, canIgnore: false };
  const res = await fetch(findingsUrl(containers, filter));
  if (!res.ok) throw new Error(`findings: ${res.status}`);
  const body = await res.json();
  return {
    findings: body.findings ?? [],
    fixLocked: !!body.fixLocked,
    truncated: !!body.truncated,
    canIgnore: !!body.canIgnore,
  };
}

/** Ignore a finding account-wide, or take that back. */
export async function setFindingIgnored(id: string, ignored: boolean) {
  const res = await fetch(withBase(`/api/cloud/findings/${encodeURIComponent(id)}/ignore`), {
    method: ignored ? "POST" : "DELETE",
  });
  return res.ok;
}

/**
 * The newest finding this reader has opened the panel on, per container name,
 * by first sighting. First rather than last seen: an ongoing finding is re-seen
 * every morning, and the dot is for something new, not for something still
 * there. Per name because a finding records names, and one mark for every
 * container let reading container A put out the dot for a finding on B.
 */
// Per browser rather than per profile: a profile field is a server-side setting,
// and this is only a dot. The key once held a single number; that reads as
// nothing seen, so the dots come back once rather than hiding something new.
const SEEN_KEY = "DOZZLE_LAST_SEEN_FINDING";
/** Names kept, newest marks first. Enough for any screen, small enough to never matter. */
export const SEEN_LIMIT = 200;

function readSeen(): Record<string, number> {
  try {
    const parsed = JSON.parse(localStorage.getItem(SEEN_KEY) ?? "{}");
    return parsed && typeof parsed === "object" && !Array.isArray(parsed) ? parsed : {};
  } catch {
    // Storage blocked or garbled: no marks, so a dot shows rather than hides.
    return {};
  }
}

const seenFindings = ref<Record<string, number>>(readSeen());

/** Moves each container's mark up to the newest finding shown on it. Monotonic. */
export function markFindingsSeen(findings: CloudFinding[]) {
  const next = { ...seenFindings.value };
  let changed = false;
  for (const f of findings) {
    for (const c of f.containers) {
      if (f.firstSeenAt > (next[c.name] ?? 0)) {
        next[c.name] = f.firstSeenAt;
        changed = true;
      }
    }
  }
  if (!changed) return;
  const kept = Object.fromEntries(
    Object.entries(next)
      .sort((a, b) => b[1] - a[1])
      .slice(0, SEEN_LIMIT),
  );
  seenFindings.value = kept;
  try {
    localStorage.setItem(SEEN_KEY, JSON.stringify(kept));
  } catch {
    // Not persisted, but this page still remembers.
  }
}

/** Test seam: re-reads the marks from storage. */
export function resetSeenFindings() {
  seenFindings.value = readSeen();
}

/** Whether any of these findings is newer, on a container in `names`, than
 *  the reader has seen on that container. */
function hasUnseen(findings: CloudFinding[], names: Set<string>) {
  return findings.some((f) =>
    f.containers.some((c) => names.has(c.name) && f.firstSeenAt > (seenFindings.value[c.name] ?? 0)),
  );
}

export function useCloudFindings(filter: MaybeRefOrGetter<FindingsFilter>) {
  const { containers } = useCloudScope();
  const { t } = useI18n();
  const { showToast } = useToast();

  const findings = ref<CloudFinding[]>([]);
  const fixLocked = ref(false);
  const truncated = ref(false);
  const canIgnore = ref(false);
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
      canIgnore.value = !!result.canIgnore;
    } catch {
      if (mine !== request) return;
      findings.value = [];
      truncated.value = false;
      canIgnore.value = false;
      failed.value = true;
    } finally {
      if (mine === request) loading.value = false;
    }
  }

  /** Which list is on show: a filter and a scope. */
  const listKey = () => `${toValue(filter)}|${containers.value.map((c) => `${c.host}/${c.id}`).join(",")}`;
  watch(listKey, load, { immediate: true });

  /** Puts a row back where it was, unless the list moved on while the request
   *  was out: then it is fetched again rather than patched with a stale row. */
  function putBack(finding: CloudFinding, index: number, key: string) {
    if (key !== listKey()) return load();
    if (findings.value.some((f) => f.id === finding.id)) return;
    findings.value.splice(Math.min(index, findings.value.length), 0, finding);
  }

  /** Ignored leaves New and All, and un-ignored leaves Ignored, so either way
   *  the row goes; the toast puts it back if the reader changes their mind. */
  async function toggleIgnored(finding: CloudFinding, ignored: boolean) {
    const index = findings.value.indexOf(finding);
    if (index === -1) return;
    const key = listKey();
    findings.value.splice(index, 1);
    if (!(await setFindingIgnored(finding.id, ignored))) {
      putBack(finding, index, key);
      showToast({ type: "error", message: t("cloud-findings.ignore-failed") }, { expire: 5000 });
      return;
    }
    if (ignored) {
      showToast(
        {
          type: "info",
          message: t("cloud-findings.ignored"),
          action: { label: t("cloud-findings.undo"), handler: () => restore(finding, index, key) },
        },
        { expire: 8000 },
      );
    }
  }

  async function restore(finding: CloudFinding, index: number, key: string) {
    if (await setFindingIgnored(finding.id, false)) putBack(finding, index, key);
  }

  function markSeen() {
    markFindingsSeen(findings.value);
  }

  return {
    findings,
    fixLocked,
    truncated,
    canIgnore,
    loading,
    failed,
    reload: load,
    ignore: (f: CloudFinding) => toggleIgnored(f, true),
    unignore: (f: CloudFinding) => toggleIgnored(f, false),
    markSeen,
  };
}

/** Whether the scope on screen has a new finding first seen after the reader
 *  last opened the panel on that container. Asked once per scope, for the
 *  tab's dot. */
export function useUnseenFindings(enabled: MaybeRefOrGetter<boolean> = true) {
  const { containers } = useCloudScope();
  const latest = ref<CloudFinding[]>([]);
  const names = computed(() => new Set(containers.value.map((c) => c.name)));

  let request = 0;
  watch(
    () => (toValue(enabled) ? containers.value.map((c) => `${c.host}/${c.id}`).join(",") : ""),
    async (key) => {
      // A scope left behind must not answer for the one on screen now.
      const mine = ++request;
      latest.value = [];
      if (!key) return;
      try {
        const { findings } = await fetchFindings(containers.value, "new");
        if (mine === request) latest.value = findings;
      } catch {
        // No dot is the honest answer when the question could not be asked.
      }
    },
    { immediate: true },
  );

  return computed(() => hasUnseen(latest.value, names.value));
}
