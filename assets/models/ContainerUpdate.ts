/**
 * One container update as Dozzle recorded it: the image under a name changed.
 * Only updates Dozzle made itself are recorded. Mirrors container.UpdateRecord,
 * sent on the log stream as a `container-update` event for the container the
 * update created.
 */
export interface ContainerUpdate {
  host: string;
  name: string;
  /** The container that ran before. */
  oldId: string;
  /** The container the update left running: the replacement, or the old one put back. */
  newId: string;
  /** repo:tag. An update keeps the ref and moves what it resolves to. */
  imageRef?: string;
  /** repo@sha256:... from Dozzle, bare sha256:... from Dozzle Cloud. */
  fromDigest?: string;
  toDigest?: string;
  fromImageId?: string;
  toImageId?: string;
  /** RFC 3339. When the update finished. */
  at: string;
  /** schedule | dozzle | cloud | rollback. Empty when only Dozzle Cloud knows the update. */
  source: string;
  /** Dozzle's swap put the old container back because the new one failed. */
  rolledBack?: boolean;
}

export type DeployVerdictKind = "pending" | "clean" | "regressed" | "unsure" | "rolled_back_by_dozzle";

/** What Dozzle Cloud made of an update. Mirrors cloud.DeployHit. */
export interface DeployVerdict {
  /** Opaque, sqids-encoded. */
  deployId: string;
  container?: string;
  fromRef?: string;
  toRef?: string;
  fromDigest?: string;
  toDigest?: string;
  verdict: DeployVerdictKind;
  reason?: string;
  decision?: "rolled_back" | "kept" | "";
  url?: string;
}

/** The tag of an image ref: "1.4.2" from "ghcr.io/immich-app/immich-server:1.4.2", "latest" when untagged. */
export function imageTag(ref: string | undefined): string {
  let r = (ref ?? "").trim();
  if (r === "") return "";
  const at = r.indexOf("@");
  if (at >= 0) r = r.slice(0, at);
  const slash = r.lastIndexOf("/");
  const colon = r.lastIndexOf(":");
  return colon > slash ? r.slice(colon + 1) : "latest";
}

/** The 12 hex characters of a digest or image id people compare. */
export function shortDigest(digest: string | undefined): string {
  let d = digest ?? "";
  const at = d.indexOf("@");
  if (at >= 0) d = d.slice(at + 1);
  return d.replace(/^sha256:/, "").slice(0, 12);
}

/**
 * How an update's two images read. The tag does not move (latest → latest), so
 * each side is the tag with its digest: "latest (aaaaaaaaaaaa)". The same rule
 * Dozzle Cloud's messages use, so the marker and the alert name the images alike.
 */
export function updateLabels(
  update: Pick<ContainerUpdate, "imageRef" | "fromDigest" | "toDigest" | "fromImageId" | "toImageId">,
): {
  from: string;
  to: string;
} {
  const tag = imageTag(update.imageRef);
  const label = (digest: string) => (tag && digest ? `${tag} (${digest})` : tag || digest);
  return {
    from: label(shortDigest(update.fromDigest || update.fromImageId)),
    to: label(shortDigest(update.toDigest || update.toImageId)),
  };
}

/** A regression, or a maybe, that nobody has kept or rolled back yet. */
export function undecidedRegression(verdict: DeployVerdict | undefined): boolean {
  return !!verdict && (verdict.verdict === "regressed" || verdict.verdict === "unsure") && !verdict.decision;
}

// The judge has not answered for long past this, so an open marker stops asking.
const VERDICT_WAIT_MS = 6 * 60 * 60 * 1000;

/**
 * Whether the marker still waits on Dozzle Cloud: no verdict yet, a pending one,
 * or a regression nobody has decided. Bounded by age, so a marker Dozzle Cloud
 * never judges (an account without judged updates) does not keep the poll asking.
 */
export function awaitingVerdict(at: Date, verdict: DeployVerdict | undefined, now = Date.now()): boolean {
  if (now - at.getTime() > VERDICT_WAIT_MS) return false;
  if (!verdict || verdict.verdict === "pending") return true;
  return undecidedRegression(verdict);
}
