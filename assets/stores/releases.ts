const { data: releases } = useFetch(withBase("/api/releases")).get().json<
  {
    name: string;
    mentionsCount: number;
    createdAt: string;
    body: string;
    tag: string;
    htmlUrl: string;
    current: boolean;
    latest: boolean;
  }[]
>();

const hasUpdate = computed(() => {
  if (!releases.value?.length) return false;
  return releases.value[0].tag !== config.version;
});

const latest = computed(() => releases.value?.find((release) => release.latest));

const current = computed(() => releases.value?.find((release) => release.current));

export function useReleases() {
  return {
    hasUpdate,
    current,
    latest,
    releases,
  };
}
