import type { CloudConfig, CloudStatus } from "@/types/notifications";

// Shared state across all component instances.
//
// Seeded from the shell rather than fetched: the server reads this out of memory
// when it renders the page, and everything else cloud-shaped keys off `linked`,
// so asking for it over the wire put a round trip in front of the cloud status
// and the recent-alerts history. Reads are open to any signed-in user; only
// linking needs the role. The key is absent on the login page's cut-down config
// and in unit tests that inject none.
const cloudConfig = ref<CloudConfig | null>(config.enableCloud ? (config.cloudConfig ?? null) : null);
const cloudStatus = ref<CloudStatus | null>(null);
const cloudStatusError = ref<"auth" | "unavailable" | false>(false);
const isLoadingCloudStatus = ref(false);

// Re-reads what the server actually stored, for the forms that change it. Boot
// does not call this — see the seed above.
async function fetchCloudConfig() {
  if (!config.enableCloud) {
    cloudConfig.value = null;
    return;
  }
  try {
    const res = await fetch(withBase("/api/cloud/config"));
    if (!res.ok) {
      cloudConfig.value = null;
      return;
    }
    cloudConfig.value = await res.json();
  } catch {
    cloudConfig.value = null;
  }
}

async function loadCloudStatus() {
  isLoadingCloudStatus.value = true;
  cloudStatusError.value = false;
  try {
    const res = await fetch(withBase("/api/cloud/status"));
    if (!res.ok) {
      cloudStatusError.value = res.status === 401 || res.status === 403 ? "auth" : "unavailable";
      return;
    }
    cloudStatus.value = await res.json();
  } catch {
    cloudStatusError.value = "unavailable";
  } finally {
    isLoadingCloudStatus.value = false;
  }
}

// Several components (nav popover, pro badge, settings card) can ask for the
// status at once on the same page, so callers that overlap share one request.
let pendingStatus: Promise<void> | null = null;
async function fetchCloudStatus() {
  if (!config.enableCloud || !cloudConfig.value?.linked) return;
  pendingStatus ??= loadCloudStatus().finally(() => (pendingStatus = null));
  return pendingStatus;
}

// For consumers that only need a status, not a fresh one.
async function ensureCloudStatus() {
  if (cloudStatus.value) return;
  return fetchCloudStatus();
}

// Cloud decides what a plan buys. The name checks only cover a Cloud older than
// `paid` and `metrics_lookback_hours`; they used to be the whole answer, and
// classed "team" as free.
const isPro = computed(() => {
  const plan = cloudStatus.value?.plan;
  if (!plan) return false;
  return plan.paid ?? ["pro", "team"].includes(plan.name.toLowerCase());
});
const metricsLookbackHours = computed(() => cloudStatus.value?.plan.metrics_lookback_hours ?? (isPro.value ? 720 : 24));

function clearCloudState() {
  cloudConfig.value = null;
  cloudStatus.value = null;
  cloudStatusError.value = false;
}

export function useCloudConfig() {
  return {
    cloudConfig,
    cloudStatus,
    cloudStatusError,
    isLoadingCloudStatus,
    isPro,
    metricsLookbackHours,
    fetchCloudConfig,
    fetchCloudStatus,
    ensureCloudStatus,
    clearCloudState,
  };
}
