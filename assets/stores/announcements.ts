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

const { data: releases } = useFetch(withBase("/api/releases")).get().json<Announcement[]>();

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
  };
}
