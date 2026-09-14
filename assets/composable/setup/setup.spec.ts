import { describe, expect, test, vi } from "vitest";

vi.mock("@/stores/config", () => ({
  default: { base: "" },
  withBase: (path: string) => path,
}));

import {
  setupEnvSnippet,
  setupHasPending,
  setupLoginConfigured,
  setupShouldAutoOpen,
  setupSteps,
  setupToggles,
  type SetupStatus,
} from "./setup";

function status(overrides: Partial<SetupStatus> = {}): SetupStatus {
  return {
    mode: "server",
    dataPersisted: true,
    authProvider: "none",
    usersFileExists: false,
    enableActions: false,
    enableShell: false,
    locked: { authProvider: false, enableActions: false, enableShell: false },
    pending: {},
    canRestart: true,
    windowOpen: true,
    canWrite: true,
    ...overrides,
  };
}

const unlinked = { linked: false, canLink: true };

describe("setupSteps", () => {
  test("fresh install sees every step", () => {
    expect(setupSteps(status(), unlinked)).toEqual(["login", "actions", "cloud", "restart"]);
  });

  test("locked auth provider skips login", () => {
    const s = status({ locked: { authProvider: true, enableActions: false, enableShell: false } });
    expect(setupSteps(s, unlinked)).toEqual(["actions", "cloud", "restart"]);
  });

  test("one locked toggle keeps the actions step", () => {
    const s = status({ locked: { authProvider: false, enableActions: true, enableShell: false } });
    expect(setupSteps(s, unlinked)).toContain("actions");
  });

  test("both toggles locked drops the actions step", () => {
    const s = status({ locked: { authProvider: false, enableActions: true, enableShell: true } });
    expect(setupSteps(s, unlinked)).toEqual(["login", "cloud", "restart"]);
  });

  test("cloud is skipped when already linked", () => {
    expect(setupSteps(status(), { linked: true, canLink: true })).not.toContain("cloud");
  });

  test("cloud is skipped when the user cannot link", () => {
    expect(setupSteps(status(), { linked: false, canLink: false })).not.toContain("cloud");
  });

  test("restart is always last", () => {
    const s = status({ locked: { authProvider: true, enableActions: true, enableShell: true } });
    expect(setupSteps(s, { linked: true, canLink: false })).toEqual(["restart"]);
  });
});

describe("setupLoginConfigured", () => {
  test("none with nothing pending is not configured", () => {
    expect(setupLoginConfigured(status())).toBe(false);
  });
  test("a running provider is configured", () => {
    expect(setupLoginConfigured(status({ authProvider: "simple" }))).toBe(true);
  });
  test("a saved provider waiting for restart is configured", () => {
    expect(setupLoginConfigured(status({ pending: { authProvider: "simple" } }))).toBe(true);
  });
});

describe("pending", () => {
  test("toggles prefer pending values", () => {
    expect(setupToggles(status({ enableActions: false, pending: { enableActions: true } }))).toEqual({
      enableActions: true,
      enableShell: false,
    });
  });

  test("detects pending changes", () => {
    expect(setupHasPending(status())).toBe(false);
    expect(setupHasPending(status({ pending: { enableShell: false } }))).toBe(true);
  });

  test("env snippet lists only pending keys", () => {
    const snippet = setupEnvSnippet(status({ pending: { authProvider: "simple", enableActions: true } }));
    expect(snippet).toContain("DOZZLE_AUTH_PROVIDER: simple");
    expect(snippet).toContain('DOZZLE_ENABLE_ACTIONS: "true"');
    expect(snippet).not.toContain("DOZZLE_ENABLE_SHELL");
  });
});

describe("setupShouldAutoOpen", () => {
  const base = {
    mode: "server",
    authProvider: "none",
    setupSeen: false,
    profile: undefined,
    resume: undefined,
    hideMenu: false,
  };

  test("opens on a fresh server install", () => {
    expect(setupShouldAutoOpen(base)).toBe(true);
  });
  test("does not ambush an existing profile", () => {
    expect(setupShouldAutoOpen({ ...base, profile: { releaseSeen: "v8" } })).toBe(false);
  });
  test("stays closed once seen", () => {
    expect(setupShouldAutoOpen({ ...base, setupSeen: true })).toBe(false);
  });
  test("stays closed with auth on unless resuming", () => {
    expect(setupShouldAutoOpen({ ...base, authProvider: "simple" })).toBe(false);
    expect(setupShouldAutoOpen({ ...base, authProvider: "simple", resume: "actions" })).toBe(true);
  });
  test("never opens in swarm, k8s or hideMenu", () => {
    expect(setupShouldAutoOpen({ ...base, mode: "swarm", resume: "actions" })).toBe(false);
    expect(setupShouldAutoOpen({ ...base, mode: "k8s" })).toBe(false);
    expect(setupShouldAutoOpen({ ...base, hideMenu: true, resume: "actions" })).toBe(false);
  });
});
