// Linking is a round trip through Dozzle Cloud that ends on /api/cloud/callback.
// The callback is a plain GET a cross-site page could also navigate to, so it
// only links when it carries back a one-time state this instance minted for the
// same person. Every link button goes through here to fetch one first.
export async function startCloudLink(from: "cloud" | "notifications" | "setup") {
  trackUsage("cloud.connect");
  const res = await fetch(withBase("/api/cloud/link"), { method: "POST" });
  if (!res.ok) throw new Error(`failed to start cloud link: ${res.status}`);
  const { state } = (await res.json()) as { state: string };

  const url = new URL(`${config.cloudUrl}/link`);
  url.searchParams.set("appUrl", `${window.location.origin}${withBase("/")}`);
  url.searchParams.set("from", from);
  url.searchParams.set("state", state);
  window.location.assign(url.toString());
}
