import { client } from "@/modules/urql";
import { GetReleasesDocument, type Release } from "@/types/graphql";

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
    const { data } = await client.query(GetReleasesDocument, {});
    releases.value =
      data?.releases?.map((r) => ({
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
