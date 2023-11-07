const { data: releases } = useFetch(withBase("/api/releases"))
  .get()
  .json<{ name: string; mentionsCount: number; createdAt: string; body: string; tag: string }[]>();

const hasUpdate = computed(() => {
  if (!releases.value?.length) return false;
  return releases.value[0].tag !== config.version;
});

const latest = computed(() => {
  if (!releases.value?.length) return undefined;
  return releases.value[0];
});

export function useReleases() {
  return {
    hasUpdate,
    latest,
    releases,
  };
}
