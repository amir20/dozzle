// State and API for the setup wizard (SetupWizard.vue).
//
// The step list is a pure function of GET /api/setup plus two cloud facts, so the
// rules for "which steps does this install see" are testable without a browser.

export type SetupStepId = "login" | "actions" | "cloud" | "restart";
export type SetupStepState = "done" | "current" | "todo" | "skipped";

export interface SetupStatus {
  mode: string;
  dataPersisted: boolean;
  authProvider: string;
  usersFileExists: boolean;
  enableActions: boolean;
  enableShell: boolean;
  locked: { authProvider: boolean; enableActions: boolean; enableShell: boolean };
  pending: { authProvider?: string; enableActions?: boolean; enableShell?: boolean };
  canRestart: boolean;
  windowOpen: boolean;
  canWrite: boolean;
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
  if (!cloud.linked && cloud.canLink) steps.push("cloud");
  steps.push("restart");
  return steps;
}

// Opens by itself only on a fresh install (an empty profile, so nobody upgrading
// gets ambushed) or when a restart or the cloud round trip left a resume marker.
export function setupShouldAutoOpen(input: {
  mode: string;
  authProvider: string;
  setupSeen: boolean;
  profile: object | undefined;
  resume: SetupStepId | undefined;
  hideMenu: boolean;
}): boolean {
  if (input.hideMenu || input.mode !== "server") return false;
  if (input.resume) return true;
  return input.authProvider === "none" && !input.setupSeen && Object.keys(input.profile ?? {}).length === 0;
}

// Login counts as done once a provider is running or saved and waiting for a restart.
export function setupLoginConfigured(status: SetupStatus): boolean {
  return status.authProvider !== "none" || !!status.pending.authProvider;
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

export class SetupError extends Error {
  constructor(
    public status: number,
    message: string,
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
    throw new SetupError(res.status, text.trim() || res.statusText);
  }
  return res;
}

// Shared across the layout's wizard and the settings entry that reopens it.
const status = ref<SetupStatus | null>(null);
const loading = ref(false);
const loadError = ref(false);
const wizardOpen = ref(false);

async function fetchStatus() {
  loading.value = true;
  try {
    const res = await request("/api/setup");
    status.value = await res.json();
    loadError.value = false;
  } catch {
    loadError.value = true;
  } finally {
    loading.value = false;
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

  async function saveConfig(patch: { enableActions?: boolean; enableShell?: boolean }) {
    if (Object.keys(patch).length === 0) return;
    await request("/api/setup/config", { method: "PATCH", body: JSON.stringify(patch) });
    await fetchStatus();
  }

  async function restart() {
    await request("/api/setup/restart", { method: "POST" });
  }

  // The restart lands ~500ms after the 202, so a response right away is still the
  // old process. Back means it answered after failing once, or after 2s.
  async function waitForRestart({ timeout = 60_000, interval = 500 } = {}) {
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
      else if (failed || Date.now() - started >= 2000) {
        window.location.assign(withBase("/"));
        return true;
      }
    }
    return false;
  }

  function openWizard() {
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
    restart,
    waitForRestart,
    openWizard,
  };
}
