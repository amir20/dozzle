// State and API for the setup wizard (SetupWizard.vue).
//
// The step list is a pure function of GET /api/setup plus two cloud facts, so the
// rules for "which steps does this install see" are testable without a browser.

export type SetupStepId = "login" | "actions" | "hosts" | "cloud" | "update" | "restart";
export type SetupStepState = "done" | "current" | "todo" | "skipped" | "disabled";

export type AutoUpdateMode = "off" | "daily" | "weekly";
export type AutoUpdateReason = "not-server" | "no-container" | "pinned-tag" | "swarm-worker" | "actions-off";

export interface SetupAutoUpdate {
  mode: AutoUpdateMode;
  // "HH:MM", server local time.
  time: string;
  supported: boolean;
  reason?: AutoUpdateReason;
  image: string;
  currentVersion: string;
}

// An agent this hub connects to. A locked one came from DOZZLE_REMOTE_AGENT and can
// only be removed where it was set; the rest live in dozzle.yml.
export interface SetupAgent {
  endpoint: string;
  address: string;
  name?: string;
  // Empty while the agent is not connected.
  hostId?: string;
  locked: boolean;
  // Authenticates with the hub's own pair instead of the one it started with.
  private?: boolean;
}

// The hub's private pair for agents added from the UI. It holds a private key, so
// it only ever lives in the state of the component that asked for it.
export interface SetupAgentCert {
  cert: string;
  key: string;
  notAfter: string;
}

// The image the agent should run: the one this hub follows, so an agent never lacks
// what the hub's snippet asks of it (DOZZLE_CERT_PEM is newer than most agents).
// Outside a container the hub cannot see its image, so its version names the tag.
export function agentImage(hubImage: string | undefined, version: string): string {
  // An image id, a tag with no repository like dozzle:dev, or one from a registry on
  // the hub's own loopback only exists on that machine, so another cannot pull it.
  if (
    hubImage &&
    hubImage.includes("/") &&
    !/^(sha256:)?[0-9a-f]{12,64}$/.test(hubImage) &&
    !/^(localhost|127\.\d+\.\d+\.\d+|\[::1\])(:\d+)?\//i.test(hubImage)
  )
    return hubImage;
  if (/^v\d+\.\d+\.\d+(-[\w.]+)?$/.test(version)) return `amir20/dozzle:${version}`;
  // A PR build reports pr-<number>-<commit>, but only pr-<number> is published.
  const pr = version.match(/^pr-\d+/);
  if (pr) return `amir20/dozzle:${pr[0]}`;
  return "amir20/dozzle:latest";
}

// The agent compose file. With a pair, the PEMs go in as YAML literal blocks, each
// line indented under its key so the file stays valid YAML.
export function agentComposeSnippet(image: string, cert?: Pick<SetupAgentCert, "cert" | "key">): string {
  const lines = ["services:", "  dozzle-agent:", `    image: ${image}`, "    command: agent"];
  if (cert) {
    const block = (name: string, pem: string) => [
      `      ${name}: |`,
      ...pem
        .trim()
        .split(/\r?\n/)
        .map((line) => `        ${line.trim()}`),
    ];
    lines.push("    environment:", ...block("DOZZLE_CERT_PEM", cert.cert), ...block("DOZZLE_KEY_PEM", cert.key));
  }
  lines.push("    volumes:", "      - /var/run/docker.sock:/var/run/docker.sock:ro", "    ports:", "      - 7007:7007");
  return lines.join("\n");
}

export interface SetupStatus {
  mode: string;
  dataPersisted: boolean;
  authProvider: string;
  usersFileExists: boolean;
  enableActions: boolean;
  enableShell: boolean;
  locked: { authProvider: boolean; enableActions: boolean; enableShell: boolean; autoUpdate?: boolean };
  pending: { authProvider?: string; enableActions?: boolean; enableShell?: boolean };
  canRestart: boolean;
  windowOpen: boolean;
  canWrite: boolean;
  // Applies live, so it never shows up in pending. Absent on servers without self-update.
  autoUpdate?: SetupAutoUpdate;
  // Absent on servers that predate adding hosts from the UI.
  agents?: SetupAgent[];
  canAddAgents?: boolean;
  // The hub runs its own certificate pair, which every agent has to be given too.
  customCert?: boolean;
}

// What a step tells the wizard's footer. The step owns what "Next" means on it
// (create the account, save toggles, restart), the wizard owns moving between steps.
export type SetupNextResult = "advance" | "skip" | "stay" | "finish";

export interface SetupStepHandle {
  nextLabel: string;
  nextDisabled: boolean;
  // True when going forward is not the answer the step should favor, e.g. an optional step.
  nextPlain: boolean;
  // Set on steps that are fine to pass on. It sits beside Next at the same size,
  // so declining never reads as the lesser, harder-to-find choice.
  skipLabel?: string;
  // The actions step's unsaved switch, so steps that depend on it react before Next.
  actionsDraft?: boolean;
  // Unsaved changes on the step. Jumping away from the rail saves them first, the
  // same as Next, so a click on another step never quietly drops them.
  dirty?: boolean;
  busy: boolean;
  next: () => Promise<SetupNextResult>;
}

export interface SetupCloudFacts {
  linked: boolean;
  canLink: boolean;
}

// Written right before anything that leaves the page (a restart, the cloud link
// round trip) so the wizard reopens at the step after it.
export const SETUP_RESUME_KEY = "DOZZLE_SETUP_RESUME";

export function setupSteps(status: SetupStatus, cloud: SetupCloudFacts): SetupStepId[] {
  const steps: SetupStepId[] = [];
  // An operator who pinned the provider with a flag has already answered this.
  if (!status.locked.authProvider) steps.push("login");
  // Nothing to toggle when both are pinned.
  if (!(status.locked.enableActions && status.locked.enableShell)) steps.push("actions");
  // Adding a host applies live, so the step never feeds the restart at the end.
  if (status.canAddAgents) steps.push("hosts");
  if (!cloud.linked && cloud.canLink) steps.push("cloud");
  // Always listed so people see what actions would unlock. The wizard greys it out
  // while actions are off, since self-update is an action.
  if (status.mode === "server" && status.autoUpdate) steps.push("update");
  steps.push("restart");
  return steps;
}

// Every hour on the hour, plus whatever the file holds if someone wrote 03:30 by hand.
export function setupUpdateTimes(current: string): string[] {
  const times = Array.from({ length: 24 }, (_, h) => `${String(h).padStart(2, "0")}:00`);
  if (/^([01]\d|2[0-3]):[0-5]\d$/.test(current) && !times.includes(current)) {
    times.push(current);
    times.sort();
  }
  return times;
}

// Opens by itself only on a fresh install (an empty profile, so nobody upgrading
// gets ambushed) or when a restart or the cloud round trip left a resume marker.
// DOZZLE_DISABLE_SETUP_WIZARD only drops the fresh-install case, so a wizard opened
// from Settings still resumes after the restart it asked for.
export function setupShouldAutoOpen(input: {
  mode: string;
  authProvider: string;
  setupSeen: boolean;
  profile: object | undefined;
  resume: SetupStepId | undefined;
  hideMenu: boolean;
  disabled?: boolean;
}): boolean {
  if (input.hideMenu || input.mode !== "server") return false;
  if (input.resume) return true;
  if (input.disabled) return false;
  return input.authProvider === "none" && !input.setupSeen && Object.keys(input.profile ?? {}).length === 0;
}

// Login counts as done once a provider is running or saved and waiting for a restart.
export function setupLoginConfigured(status: SetupStatus): boolean {
  return status.authProvider !== "none" || !!status.pending.authProvider;
}

// Whether a step already holds a choice, so reopening the wizard later shows it as
// done instead of asking again from scratch.
export function setupStepConfigured(id: SetupStepId, status: SetupStatus): boolean {
  if (id === "login") return setupLoginConfigured(status);
  if (id === "actions") {
    const toggles = setupToggles(status);
    return toggles.enableActions || toggles.enableShell;
  }
  if (id === "hosts") return (status.agents?.length ?? 0) > 0;
  if (id === "update") return !!status.autoUpdate && status.autoUpdate.mode !== "off";
  return false;
}

// What each toggle will be after the next restart.
export function setupToggles(status: SetupStatus) {
  return {
    enableActions: status.pending.enableActions ?? status.enableActions,
    enableShell: status.pending.enableShell ?? status.enableShell,
  };
}

export function setupHasPending(status: SetupStatus): boolean {
  return Object.values(status.pending).some((v) => v !== undefined && v !== null);
}

// The compose lines that do the same as the pending changes, for installs that
// cannot restart themselves.
export function setupEnvSnippet(status: SetupStatus): string {
  const lines: string[] = [];
  const { authProvider, enableActions, enableShell } = status.pending;
  if (authProvider != null) lines.push(`      DOZZLE_AUTH_PROVIDER: ${authProvider}`);
  if (enableActions != null) lines.push(`      DOZZLE_ENABLE_ACTIONS: "${enableActions}"`);
  if (enableShell != null) lines.push(`      DOZZLE_ENABLE_SHELL: "${enableShell}"`);
  return ["services:", "  dozzle:", "    environment:", ...lines].join("\n");
}

export function readSetupResume(): SetupStepId | undefined {
  try {
    const value = localStorage.getItem(SETUP_RESUME_KEY);
    return value ? (value as SetupStepId) : undefined;
  } catch {
    return undefined;
  }
}

export function writeSetupResume(step: SetupStepId) {
  try {
    localStorage.setItem(SETUP_RESUME_KEY, step);
  } catch {
    // Private windows and blocked storage: the wizard just does not resume.
  }
}

export function clearSetupResume() {
  try {
    localStorage.removeItem(SETUP_RESUME_KEY);
  } catch {
    // Nothing to clear.
  }
}

// What the server's X-Dozzle-Error header says went wrong, so messages never hang
// on the wording of the English body. Absent on 403s and 500s.
export type SetupErrorCode =
  | "unsupported-mode"
  | "not-persisted"
  | "invalid"
  | "exists"
  | "duplicate-host"
  | "no-private-cert"
  | "rate-limited"
  | "cert-mismatch"
  | "unreachable"
  | "env-agent"
  | "not-found"
  | "custom-cert";

export class SetupError extends Error {
  constructor(
    public status: number,
    message: string,
    public code?: SetupErrorCode,
  ) {
    super(message);
  }
}

async function request(path: string, init?: RequestInit) {
  const res = await fetch(withBase(path), {
    ...init,
    headers: init?.body ? { "Content-Type": "application/json" } : undefined,
  });
  if (!res.ok) {
    const text = await res.text().catch(() => "");
    const code = (res.headers.get("X-Dozzle-Error") || undefined) as SetupErrorCode | undefined;
    throw new SetupError(res.status, text.trim() || res.statusText, code);
  }
  return res;
}

// Shared across the layout's wizard and the settings entry that reopens it.
const status = ref<SetupStatus | null>(null);
const loading = ref(false);
const loadError = ref(false);
const wizardOpen = ref(false);
// Several callers fire fetchStatus without waiting (opening the hosts dialog, then
// adding one), so only the newest request may write: an older, slower answer would
// otherwise put back a list that is already stale.
let fetchSeq = 0;

async function fetchStatus() {
  const seq = ++fetchSeq;
  loading.value = true;
  try {
    const res = await request("/api/setup");
    const next: SetupStatus = await res.json();
    if (seq === fetchSeq) {
      status.value = next;
      loadError.value = false;
    }
  } catch {
    if (seq === fetchSeq) loadError.value = true;
  } finally {
    if (seq === fetchSeq) loading.value = false;
  }
  return status.value;
}

export function useSetup() {
  async function createAccount(body: { username: string; password: string; email?: string }) {
    await request("/api/setup/account", { method: "POST", body: JSON.stringify(body) });
    await fetchStatus();
  }

  async function useProxy() {
    await request("/api/setup/auth", { method: "POST", body: JSON.stringify({ provider: "forward-proxy" }) });
    await fetchStatus();
  }

  async function saveConfig(patch: {
    enableActions?: boolean;
    enableShell?: boolean;
    autoUpdate?: { mode: AutoUpdateMode; time: string };
  }) {
    if (Object.keys(patch).length === 0) return;
    await request("/api/setup/config", { method: "PATCH", body: JSON.stringify(patch) });
    await fetchStatus();
  }

  // Dozzle dials the agent before saving, so a resolved promise means the host is
  // connected and already on its way to the sidebar over the events stream.
  async function addAgent(body: { address: string; name?: string; private?: boolean }) {
    const res = await request("/api/setup/agents", { method: "POST", body: JSON.stringify(body) });
    const host: { id: string; name: string; endpoint: string } = await res.json();
    await fetchStatus();
    return host;
  }

  // Not cached anywhere on purpose: the response carries a private key.
  async function agentCert(): Promise<SetupAgentCert> {
    const res = await request("/api/setup/agent-cert", { method: "POST" });
    return res.json();
  }

  async function removeAgent(endpoint: string) {
    await request("/api/setup/agents", { method: "DELETE", body: JSON.stringify({ endpoint }) });
    await fetchStatus();
  }

  async function restart() {
    await request("/api/setup/restart", { method: "POST" });
  }

  // POST /api/update/self. The body is the same update-progress stream the
  // container Update action reads.
  async function updateSelf() {
    return request("/api/update/self", { method: "POST" });
  }

  // The restart lands ~500ms after the 202, so a response right away is still the
  // old process. Back means it answered after failing once, or after 2s. A self-update
  // keeps the old process up for much longer, so it passes mustGoDown.
  async function waitForRestart({ timeout = 60_000, interval = 500, mustGoDown = false } = {}) {
    const started = Date.now();
    let failed = false;
    while (Date.now() - started < timeout) {
      await new Promise((resolve) => setTimeout(resolve, interval));
      let ok = false;
      try {
        const res = await fetch(withBase("/healthcheck"), { cache: "no-store" });
        ok = res.ok;
      } catch {
        ok = false;
      }
      if (!ok) failed = true;
      else if (failed || (!mustGoDown && Date.now() - started >= 2000)) {
        window.location.assign(withBase("/"));
        return true;
      }
    }
    return false;
  }

  // Opening rides on a change to wizardOpen. If it is ever left true while the dialog
  // is closed, setting it true again changes nothing and the click does nothing, so
  // reset it first and let the watcher see a real change.
  async function openWizard() {
    if (wizardOpen.value) {
      wizardOpen.value = false;
      await nextTick();
    }
    wizardOpen.value = true;
  }

  return {
    status,
    loading,
    loadError,
    wizardOpen,
    fetchStatus,
    createAccount,
    useProxy,
    saveConfig,
    addAgent,
    agentCert,
    removeAgent,
    restart,
    updateSelf,
    waitForRestart,
    openWizard,
  };
}
