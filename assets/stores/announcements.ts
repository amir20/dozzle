interface Release {
  name: string;
  mentionsCount: number;
  tag: string;
  body: string;
  createdAt: string;
  htmlUrl: string;
  latest: boolean;
  features: number;
  bugFixes: number;
  breaking: number;
}

type Announcement = Omit<Release, "createdAt"> & {
  announcement: boolean;
  createdAt: Date;
};

const releases = ref<Announcement[]>([]);
let fetched = false;

async function fetchReleases() {
  if (fetched) return;
  fetched = true;

  try {
    const res = await fetch(withBase("/api/releases"));
    const data: Release[] = await res.json();
    releases.value =
      data?.map((r) => ({
        ...r,
        createdAt: new Date(r.createdAt),
        announcement: false,
      })) || [];
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
  return [...releases.value, ...otherAnnouncements].sort((a, b) => b.createdAt.getTime() - a.createdAt.getTime());
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
