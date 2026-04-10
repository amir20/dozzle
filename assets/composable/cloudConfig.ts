import type { CloudConfig, CloudStatus } from "@/types/notifications";

export function useCloudConfig() {
  const cloudConfig = ref<CloudConfig | null>(null);
  const cloudStatus = ref<CloudStatus | null>(null);
  const cloudStatusError = ref(false);
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

  async function fetchCloudStatus() {
    if (!cloudConfig.value?.linked) return;
    isLoadingCloudStatus.value = true;
    cloudStatusError.value = false;
    try {
      const res = await fetch(withBase("/api/cloud/status"));
      if (!res.ok) {
        cloudStatusError.value = true;
        return;
      }
      cloudStatus.value = await res.json();
    } catch {
      cloudStatusError.value = true;
    } finally {
      isLoadingCloudStatus.value = false;
    }
  }

  return {
    cloudConfig,
    cloudStatus,
    cloudStatusError,
    isLoadingCloudStatus,
    fetchCloudConfig,
    fetchCloudStatus,
  };
}
