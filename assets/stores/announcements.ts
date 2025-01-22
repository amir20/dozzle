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

const otherAnnouncements = [
  {
    body: "I'd love to hear about your experience in this short survey to shape the future of Dozzle!",
    createdAt: new Date("2025-01-22T18:00:00Z"),
    htmlUrl: "https://tally.so/r/wLv4g2?ref=notification",
    name: "Take survey!",
    announcement: true,
    tag: "survey-2025-01",
    latest: true,
    mentionsCount: 0,
    features: 0,
    bugFixes: 0,
    breaking: 0,
  },
] as Announcement[];

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
