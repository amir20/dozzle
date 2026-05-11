import type { CloudConfig, CloudStatus } from "@/types/notifications";

// Shared state across all component instances
const cloudConfig = ref<CloudConfig | null>(null);
const cloudStatus = ref<CloudStatus | null>(null);
const cloudStatusError = ref<"auth" | "unavailable" | false>(false);
const isLoadingCloudStatus = ref(false);

async function fetchCloudConfig() {
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

// Loaded once at module import (i.e. app boot). Every consumer reads the
// shared `cloudConfig` ref — no per-component fetch.
const initialLoad = fetchCloudConfig();

async function fetchCloudStatus() {
  if (!cloudConfig.value?.linked) return;
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
    initialLoad,
    fetchCloudConfig,
    fetchCloudStatus,
    clearCloudState,
  };
}
