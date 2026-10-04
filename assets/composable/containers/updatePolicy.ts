// Which containers the auto-update schedule updates. Every container is one of:
//
//   auto    updated on the schedule
//   manual  checked and shown as "update available", updated by hand
//   off     never checked, never updated
//
// A dev.dozzle.update label decides first, then the choice saved from the UI, then
// the instance-wide mode for a container nobody chose for. The server resolves all
// of that (internal/updatepolicy); this file only reads the answer and saves choices.
//
// Server mode only: swarm has no Updates page and Kubernetes cannot update at all.

import type { Container } from "@/models/Container";
import type { ImageUpdateResult } from "@/composable/containers/imageUpdate";

export type UpdatePolicy = "auto" | "manual" | "off";
export type UpdatePolicySource = "label" | "choice" | "default";
// How a container nobody chose for is treated. Unset is "picked", which is how
// Dozzle behaved before the choice existed.
export type UpdateContainersMode = "dozzle" | "picked" | "all";

export const UPDATE_CONTAINERS_MODES: UpdateContainersMode[] = ["dozzle", "picked", "all"];
export const UPDATE_POLICIES: UpdatePolicy[] = ["auto", "manual", "off"];
export const UPDATE_LABEL = "dev.dozzle.update";

// Mirrors containerUpdatePolicy in internal/web/update_policy.go.
export interface ContainerUpdatePolicy {
  host: string;
  id: string;
  name: string;
  image: string;
  state: string;
  health?: string;
  self?: boolean;
  policy: UpdatePolicy;
  source: UpdatePolicySource;
  // What the UI saved, even when a label or "Dozzle only" overrides it.
  choice?: UpdatePolicy;
  volumes?: string[];
  last?: { status: string; error?: string; at: string };
}

export interface UpdatePolicies {
  mode: UpdateContainersMode;
  // False when /data is not on a volume: nothing chosen here would survive a recreate.
  persisted: boolean;
  canChoose: boolean;
  containers: ContainerUpdatePolicy[];
}

export const updatePoliciesAvailable = () => config.mode === "server";

/** A container nobody chose for that keeps data in named volumes, which "Everything" would update. */
export function riskyContainers(list: ContainerUpdatePolicy[]): ContainerUpdatePolicy[] {
  return list.filter((c) => !c.self && c.source === "default" && (c.volumes?.length ?? 0) > 0);
}

/** Whether the choice can be changed from the UI at all, and if not, why. */
export function policyLock(
  entry: Pick<ContainerUpdatePolicy, "source" | "self"> | undefined,
  policies: Pick<UpdatePolicies, "persisted" | "canChoose"> | null,
): "label" | "self" | "volume" | "access" | undefined {
  if (!entry || !policies) return "access";
  if (entry.self) return "self";
  if (entry.source === "label") return "label";
  if (!policies.persisted) return "volume";
  if (!policies.canChoose) return "access";
  return undefined;
}

/** An auto choice that "Dozzle only" holds back. */
export function heldBack(entry: Pick<ContainerUpdatePolicy, "choice" | "policy">) {
  return entry.choice === "auto" && entry.policy !== "auto";
}

export type ImageStatusKind = "current" | "available" | "off" | "pinned" | "local" | "private" | "unknown" | "pending";

/** What the containers list says about an image check, in a word. */
export function imageStatusKind(policy: UpdatePolicy, result: ImageUpdateResult | undefined): ImageStatusKind {
  if (policy === "off") return "off";
  switch (result?.status) {
    case undefined:
      return "pending";
    case "up-to-date":
      return "current";
    case "update-available":
      return "available";
    case "pinned":
      return "pinned";
    case "not-checkable":
      return "local";
    case "auth-required":
      return "private";
    case "skipped":
      return "off";
    default:
      return "unknown";
  }
}

const state = ref<UpdatePolicies | null>(null);
const loading = ref(false);
let fetchedAt = 0;
let seq = 0;

// Policies change rarely and only through this tab or dozzle.yml, so one answer is
// shared by every surface for a minute.
const FRESH_FOR = 60_000;

async function fetchPolicies(force = false) {
  if (!updatePoliciesAvailable()) return state.value;
  if (!force && state.value && Date.now() - fetchedAt < FRESH_FOR) return state.value;
  const mine = ++seq;
  loading.value = true;
  try {
    const res = await fetch(withBase("/api/updates/policy"));
    if (res.ok && mine === seq) {
      state.value = (await res.json()) as UpdatePolicies;
      fetchedAt = Date.now();
    }
  } catch {
    // The surfaces that need it stay hidden.
  } finally {
    if (mine === seq) loading.value = false;
  }
  return state.value;
}

export class UpdatePolicyError extends Error {
  constructor(
    public status: number,
    public code?: string,
  ) {
    super(code ?? String(status));
  }
}

export function useUpdatePolicies() {
  // Containers are recreated under new ids by an update, so lookups go by the id
  // the store has now, which is the one the server listed on its last answer.
  const byKey = computed(() => {
    const map = new Map<string, ContainerUpdatePolicy>();
    for (const c of state.value?.containers ?? []) map.set(`${c.host}/${c.id}`, c);
    return map;
  });

  function policyFor(container: Pick<Container, "host" | "id">) {
    return byKey.value.get(`${container.host}/${container.id}`);
  }

  // "" forgets the choice, so the container follows the mode again.
  async function setPolicy(containers: Pick<Container, "host" | "id">[], policy: UpdatePolicy | "") {
    const res = await fetch(withBase("/api/updates/policy"), {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ containers: containers.map(({ host, id }) => ({ host, id })), policy }),
    });
    if (!res.ok) throw new UpdatePolicyError(res.status, res.headers.get("X-Dozzle-Error") ?? undefined);
    await fetchPolicies(true);
  }

  return {
    policies: readonly(state),
    loading: readonly(loading),
    fetchPolicies,
    policyFor,
    setPolicy,
  };
}
