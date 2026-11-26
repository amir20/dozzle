// App icons for well-known container images, sourced from homarr-labs/dashboard-icons
// and vendored under assets/icons/apps. They are bundled, never fetched from a CDN:
// asking a third party for "sonarr.svg" would leak what a user is running, and would
// break air-gapped installs.
//
// Vite emits each file as its own hashed asset (see assetsInlineLimit in vite.config.ts),
// so this map costs a few KB of strings and the browser only downloads the handful of
// icons actually rendered.
import matching from "../icons/apps/matching.json";

const bundled = import.meta.glob<string>("../icons/apps/*.{svg,webp}", {
  eager: true,
  query: "?url",
  import: "default",
});

// In dev the page is served by the Go server while modules come from the Vite server on
// another port, and Vite hands back root-relative asset paths. Resolving those against the
// page origin lands on Go's catch-all, which answers with index.html. Anchoring them to the
// module's own origin puts them back on the Vite server. Production URLs are already
// absolute, so this is a no-op there.
const origin = import.meta.env.DEV ? new URL(import.meta.url).origin : "";

const icons = new Map<string, string>();
for (const [path, url] of Object.entries(bundled)) {
  icons.set(path.slice(path.lastIndexOf("/") + 1).replace(/\.(svg|webp)$/, ""), origin + url);
}

// The matching tables live in JSON beside the logos so Dozzle Cloud reads the same
// ones (see the file's own comment). Distributors repackage other people's software,
// so their own logo is never the answer; generic names (`goauthentik/server`) must
// not match an icon called "server"; the suffix decorates a name without changing it;
// aliases map image names that don't line up with their slug.
const DISTRIBUTORS = new Set(matching.distributors);
const GENERIC = new Set(matching.generic);
const SUFFIX = new RegExp(matching.suffix);
const ALIASES: Record<string, string> = matching.aliases;

const stripSuffix = (name: string) => {
  const stripped = name.replace(SUFFIX, "");
  return stripped.length > 1 ? stripped : name;
};

/**
 * Resolves a Docker image reference to a bundled icon slug.
 *
 * Matching prefers the image name itself and falls back to the owning namespace,
 * which is what rescues images published under a generic name such as
 * `ghcr.io/goauthentik/server`.
 */
export function iconSlugForImage(image: string | undefined): string | undefined {
  if (!image) return undefined;

  // Drop the digest, then the tag. The tag test has to be anchored past the last
  // slash so a registry port ("registry:5000/app") is not mistaken for one.
  let ref = image.split("@")[0];
  const lastSlash = ref.lastIndexOf("/");
  const lastColon = ref.lastIndexOf(":");
  if (lastColon > lastSlash) ref = ref.slice(0, lastColon);

  const segments = ref.split("/").filter(Boolean);
  if (segments.length === 0) return undefined;

  // Registry hosts carry a dot or a port, except for the localhost special case.
  if (segments.length > 1 && (/[.:]/.test(segments[0]) || segments[0] === "localhost")) {
    segments.shift();
  }

  // Nearest segment wins: `linuxserver/sonarr` is Sonarr, not LinuxServer.
  const candidates = segments.slice(-2).reverse();

  for (const segment of candidates) {
    const name = segment.toLowerCase().replace(/_/g, "-");
    if (!name || GENERIC.has(name) || DISTRIBUTORS.has(name)) continue;

    for (const slug of [ALIASES[name], name, ALIASES[stripSuffix(name)], stripSuffix(name)]) {
      if (slug && icons.has(slug)) return slug;
    }
  }

  return undefined;
}

/**
 * Picks the themed variant of an icon. dashboard-icons ships `-light` artwork meant
 * for dark backgrounds and `-dark` artwork for light ones; most icons have neither
 * and read fine either way.
 */
export function iconUrl(slug: string | undefined, dark: boolean): string | undefined {
  if (!slug) return undefined;
  if (isDataIcon(slug)) return slug;
  return icons.get(dark ? `${slug}-light` : `${slug}-dark`) ?? icons.get(slug);
}

export const hasIcon = (slug: string) => icons.has(slug);

// Labels ride along with every container the browser is told about, so an inline icon
// has to stay small. Bundled icons land around 1-3KB; this leaves room for a real SVG.
export const MAX_DATA_ICON_LENGTH = 16 * 1024;

/**
 * An icon carried inline in `dev.dozzle.icon` as a data URI, for images that will never
 * be bundled. Only raster and SVG images are accepted, and only through `<img>`, where an
 * SVG cannot run script. The CSP already allows `data:` for images and nothing else.
 */
export const isDataIcon = (value: string) =>
  value.length <= MAX_DATA_ICON_LENGTH && /^data:image\/(svg\+xml|png|webp)[;,]/i.test(value);
