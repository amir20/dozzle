/** Drops a trailing `@sha256:...` digest, leaving `registry/name:tag`. */
export function stripImageDigest(image: string): string {
  return image.replace(/@sha.*/, "");
}

/**
 * A Docker image reference short enough for a table cell: `name:tag`, with the
 * registry, org path and digest gone. The tag is what people scan for, and with the
 * org kept a long path like `victoriametrics/victoria-metrics` truncated the tag away.
 * Callers put the full reference in a tooltip.
 *
 * A bare image id (what Docker reports once the tag is gone) shortens to the 12
 * characters `docker ps` shows.
 */
export function shortImageRef(image: string | undefined): string {
  if (!image) return "";

  const id = image.match(/^(?:sha256:)?([0-9a-f]{64})$/);
  if (id) return id[1].slice(0, 12);

  // The last path segment is `name:tag`. A registry port ("registry:5000/app")
  // only ever sits in an earlier segment, so it never reaches this one.
  return stripImageDigest(image).split("/").filter(Boolean).pop() ?? "";
}
