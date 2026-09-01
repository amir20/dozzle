import { describe, expect, test } from "vitest";
import { hasIcon, iconSlugForImage, iconUrl } from "./appIcons";

describe("iconSlugForImage", () => {
  test.each([
    ["sonarr", "sonarr"],
    ["sonarr:latest", "sonarr"],
    ["linuxserver/radarr", "radarr"],
    ["lscr.io/linuxserver/prowlarr:latest", "prowlarr"],
    ["ghcr.io/hotio/bazarr", "bazarr"],
    ["ghcr.io/hotio/qbittorrent:release-4.6.5", "qbittorrent"],
    ["jc21/nginx-proxy-manager:2.11.3", "nginx-proxy-manager"],
    ["ghcr.io/gethomepage/homepage", "homepage"],
    ["amir20/dozzle:v8", "dozzle"],
    ["homebridge/homebridge:latest", "homebridge"],
    ["santiagosayshey/profilarr:latest", "profilarr"],
    ["ghcr.io/seerr/seerr", "seerr"],
  ])("resolves %s", (image, slug) => {
    expect(iconSlugForImage(image)).toBe(slug);
  });

  test("prefers the image name over the distributor namespace", () => {
    // linuxserver has an icon of its own upstream; the app inside still wins.
    expect(iconSlugForImage("linuxserver/plex")).toBe("plex");
  });

  test("falls back to the namespace when the name is generic", () => {
    expect(iconSlugForImage("ghcr.io/goauthentik/server:2024.6")).toBe("authentik");
  });

  test("does not match a generic name against an unrelated icon", () => {
    expect(iconSlugForImage("someunknownvendor/server")).toBeUndefined();
  });

  test("maps names that differ from the icon slug", () => {
    expect(iconSlugForImage("postgres:16-alpine")).toBe("postgresql");
    expect(iconSlugForImage("mongo")).toBe("mongodb");
    expect(iconSlugForImage("eclipse-mosquitto")).toBe("mosquitto");
    expect(iconSlugForImage("pihole/pihole")).toBe("pi-hole");
    expect(iconSlugForImage("ghcr.io/immich-app/immich-server:v1.106.4")).toBe("immich");
  });

  test("strips deployment suffixes", () => {
    expect(iconSlugForImage("ghcr.io/mealie-recipes/mealie:v1.2.0")).toBe("mealie");
    expect(iconSlugForImage("grafana/grafana-oss")).toBe("grafana");
  });

  test("keeps a registry port from being read as a tag", () => {
    expect(iconSlugForImage("registry.local:5000/traefik")).toBe("traefik");
  });

  test("ignores the digest", () => {
    expect(iconSlugForImage("nginx@sha256:abc123")).toBe("nginx");
  });

  test("returns undefined for unknown or empty images", () => {
    expect(iconSlugForImage("my-company/some-internal-thing")).toBeUndefined();
    expect(iconSlugForImage("")).toBeUndefined();
    expect(iconSlugForImage(undefined)).toBeUndefined();
  });
});

describe("iconUrl", () => {
  test("returns a url for a bundled slug", () => {
    expect(iconUrl("sonarr", true)).toBeTruthy();
  });

  test("prefers the light variant on a dark theme", () => {
    // plex-light.svg is bundled alongside plex.svg.
    expect(iconUrl("plex", true)).not.toBe(iconUrl("plex", false));
  });

  test("falls back to the base artwork when no variant exists", () => {
    expect(iconUrl("traefik", true)).toBe(iconUrl("traefik", false));
  });

  test("returns undefined without a slug", () => {
    expect(iconUrl(undefined, true)).toBeUndefined();
    expect(iconUrl("definitely-not-bundled", true)).toBeUndefined();
  });
});

describe("hasIcon", () => {
  test("reports what is bundled", () => {
    expect(hasIcon("radarr")).toBe(true);
    expect(hasIcon("definitely-not-bundled")).toBe(false);
  });
});
