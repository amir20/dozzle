// App icons for well-known container images, sourced from homarr-labs/dashboard-icons
// and vendored under assets/icons/apps. They are bundled, never fetched from a CDN:
// asking a third party for "sonarr.svg" would leak what a user is running, and would
// break air-gapped installs.
//
// Vite emits each file as its own hashed asset (see assetsInlineLimit in vite.config.ts),
// so this map costs a few KB of strings and the browser only downloads the handful of
// icons actually rendered.
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

// Namespaces that repackage other people's software. Their own logo is never the
// right answer for the image sitting inside them.
const DISTRIBUTORS = new Set([
  "library",
  "linuxserver",
  "lscr",
  "hotio",
  "bitnami",
  "bitnamilegacy",
  "binhex",
  "hurlenko",
  "cm2network",
  "ich777",
]);

// Generic component names. `goauthentik/server` must not match an icon called "server".
const GENERIC = new Set([
  "server",
  "client",
  "app",
  "core",
  "main",
  "web",
  "www",
  "api",
  "backend",
  "frontend",
  "base",
  "image",
  "service",
  "worker",
  "master",
  "node",
  "agent",
  "daemon",
  "db",
  "database",
  "cache",
  "proxy",
  "gateway",
  "runner",
  "job",
  "cron",
  "init",
  "latest",
  "test",
  "demo",
  "bin",
]);

// Deployment suffixes that decorate an app name without changing it.
const SUFFIX = /-(server|client|app|core|ce|ee|oss|nox|docker|alpine|slim|standalone|bin|sync|nightly)$/;

// Image names that do not line up with the icon slug on their own.
const ALIASES: Record<string, string> = {
  postgres: "postgresql",
  pgvector: "postgresql",
  timescaledb: "postgresql",
  mongo: "mongodb",
  "eclipse-mosquitto": "mosquitto",
  httpd: "apache",
  homeassistant: "home-assistant",
  "home-assistant-core": "home-assistant",
  hass: "home-assistant",
  pihole: "pi-hole",
  adguardhome: "adguard-home",
  "pms-docker": "plex",
  plexinc: "plex",
  plexmediaserver: "plex",
  "qbittorrent-nox": "qbittorrent",
  "transmission-openvpn": "transmission",
  goauthentik: "authentik",
  gethomepage: "homepage",
  fireflyiii: "firefly-iii",
  actualbudget: "actual-budget",
  "actual-server": "actual-budget",
  binwiederhier: "ntfy",
  louislam: "uptime-kuma",
  containrrr: "watchtower",
  jc21: "nginx-proxy-manager",
  "immich-server": "immich",
  "immich-machine-learning": "immich",
  "immich-microservices": "immich",
  paperless: "paperless-ngx",
  tandoor: "tandoor-recipes",
  "wg-easy": "wireguard",
  wgeasy: "wireguard",
  "changedetection.io": "changedetection",
  "changedetection-io": "changedetection",
  dgtlmoon: "changedetection",
  drawio: "draw-io",
  "zwave-js-ui": "zwavejs2mqtt",
  zwavejs: "zwavejs2mqtt",
  z2m: "zigbee2mqtt",
  "rocketchat.chat": "rocket-chat",
  rocketchat: "rocket-chat",
  "matrix-synapse": "synapse",
  codeserver: "code-server",
  "openvscode-server": "code-server",
  "node-exporter": "prometheus-node-exporter",
  nodeexporter: "prometheus-node-exporter",
  prom: "prometheus",
  mylar3: "mylar",
  frooodle: "stirling-pdf",
  openwebui: "open-webui",
  anythingllm: "anything-llm",
  "unifi-controller": "unifi",
  "unifi-network-application": "unifi",
  "unifi-console": "unifi",
  requarks: "wikijs",
  nxzai: "nextexplorer",
  dmunozv04: "sponsorblock",
  timothyjmiller: "cloudflare"
};

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
  return icons.get(dark ? `${slug}-light` : `${slug}-dark`) ?? icons.get(slug);
}

export const hasIcon = (slug: string) => icons.has(slug);
