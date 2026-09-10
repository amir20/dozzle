/**
 * One rule for every cloud-backed surface.
 *
 * A surface is `absent` when there is nothing to show and nothing the viewer
 * could do about it, `empty` when cloud is reachable but has no data yet, and
 * `ready` when it has something. The middle state is the point: an empty
 * section with one muted line saying what it would hold reads as a feature that
 * hasn't filled up, while a locked card reads as a paywall. Nothing here ever
 * renders a lock — F4 handles the gates cloud hands us.
 */
export type CloudSurfaceState = "absent" | "empty" | "ready";

type HasData = Ref<boolean> | (() => boolean) | undefined;

function resolve(hasData: HasData): boolean {
  if (hasData === undefined) return true;
  return typeof hasData === "function" ? hasData() : hasData.value;
}

export function useCloudSurface(hasData?: HasData) {
  const { cloudConfig } = useCloudConfig();

  const linked = computed(() => !!cloudConfig.value?.linked);
  const canLink = computed(() => config.enableCloud && config.canLinkCloud);

  // An unlinked instance is worth showing only to someone who can do something
  // about it. For everyone else the surface simply does not exist, rather than
  // sitting in the nav forever as an advert.
  const mounted = computed(() => config.enableCloud && (linked.value || canLink.value));

  const state = computed<CloudSurfaceState>(() => {
    if (!mounted.value) return "absent";
    if (!linked.value || !resolve(hasData)) return "empty";
    return "ready";
  });

  return { state, mounted, linked, canLink };
}
