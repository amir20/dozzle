type Announcement = {
  name: string;
  announcement: boolean;
  createdAt: Date;
  body: string;
  tag: string;
  htmlUrl: string;
  latest: boolean;
  mentionsCount: number;
  features: number;
  bugFixes: number;
  breaking: number;
};

const releases = ref<Announcement[]>([]);
let fetched = false;

async function fetchReleases() {
  if (fetched) return;
  fetched = true;

  try {
    const { data } = await useFetch(withBase("/api/releases")).get().json<Announcement[]>();
    releases.value = data.value || [];
  } catch (error) {
    console.error("Error while fetching releases:\n", error);
    fetched = false;
  }
}

if (config.releaseCheckMode === "automatic") {
  fetchReleases();
}

const otherAnnouncements = [] as Announcement[];

const announcements = computed(() => {
  const newReleases =
    releases.value?.map((release) => ({ ...release, createdAt: new Date(release.createdAt), announcement: false })) ??
    [];
  return [...newReleases, ...otherAnnouncements].sort((a, b) => b.createdAt.getTime() - a.createdAt.getTime());
});

const mostRecent = computed(() => announcements.value?.[0]);
const latestRelease = computed(() => announcements.value?.find((release) => release.latest && !release.announcement));
const hasRelease = computed(() => latestRelease.value !== undefined);

export function useAnnouncements() {
  return {
    mostRecent,
    announcements,
    latestRelease,
    hasRelease,
    fetchReleases,
  };
}
