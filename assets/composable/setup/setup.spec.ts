import { afterEach, describe, expect, test, vi } from "vitest";

vi.mock("@/stores/config", () => ({
  default: { base: "" },
  withBase: (path: string) => path,
}));

import { parse } from "yaml";
import {
  agentComposeSnippet,
  agentImage,
  setupEnvSnippet,
  setupHasPending,
  setupLoginConfigured,
  setupShouldAutoOpen,
  setupUpdateTimes,
  setupStepConfigured,
  setupSteps,
  setupToggles,
  SetupError,
  useSetup,
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

  test("hosts shows only when agents can be added live", () => {
    expect(setupSteps(status({ canAddAgents: true }), unlinked)).toEqual([
      "login",
      "actions",
      "hosts",
      "cloud",
      "restart",
    ]);
    expect(setupSteps(status({ canAddAgents: false }), unlinked)).not.toContain("hosts");
  });

  test("cloud is skipped when already linked", () => {
    expect(setupSteps(status(), { linked: true, canLink: true })).not.toContain("cloud");
  });

  test("cloud is skipped when the user cannot link", () => {
    expect(setupSteps(status(), { linked: false, canLink: false })).not.toContain("cloud");
  });

  describe("update", () => {
    const autoUpdate = {
      mode: "off" as const,
      time: "03:00",
      supported: true,
      image: "amir20/dozzle:latest",
      currentVersion: "v9.0.0",
    };

    test("shows after cloud when actions are on", () => {
      expect(setupSteps(status({ enableActions: true, autoUpdate }), unlinked)).toEqual([
        "login",
        "actions",
        "cloud",
        "update",
        "restart",
      ]);
    });

    // Listed either way so people see what actions unlock; the wizard greys it out.
    test("still listed when actions are off", () => {
      expect(setupSteps(status({ autoUpdate }), unlinked)).toContain("update");
    });

    test("still listed when actions are on but pending off", () => {
      const s = status({ enableActions: true, pending: { enableActions: false }, autoUpdate });
      expect(setupSteps(s, unlinked)).toContain("update");
    });

    test("hidden outside server mode", () => {
      expect(setupSteps(status({ mode: "swarm", enableActions: true, autoUpdate }), unlinked)).not.toContain("update");
    });

    test("hidden when the server does not report auto-update", () => {
      expect(setupSteps(status({ enableActions: true }), unlinked)).not.toContain("update");
    });
  });

  test("restart is always last", () => {
    const s = status({ locked: { authProvider: true, enableActions: true, enableShell: true } });
    expect(setupSteps(s, { linked: true, canLink: false })).toEqual(["restart"]);
  });
});

describe("setupUpdateTimes", () => {
  test("lists every hour", () => {
    const times = setupUpdateTimes("03:00");
    expect(times).toHaveLength(24);
    expect(times[0]).toBe("00:00");
    expect(times[23]).toBe("23:00");
  });

  test("keeps an off-the-hour time from the file", () => {
    const times = setupUpdateTimes("03:30");
    expect(times).toHaveLength(25);
    expect(times.indexOf("03:30")).toBe(4);
  });

  test("ignores garbage", () => {
    expect(setupUpdateTimes("25:99")).toHaveLength(24);
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

describe("setupStepConfigured", () => {
  test("actions count once either toggle is on, running or pending", () => {
    expect(setupStepConfigured("actions", status())).toBe(false);
    expect(setupStepConfigured("actions", status({ enableActions: true }))).toBe(true);
    expect(setupStepConfigured("actions", status({ pending: { enableShell: true } }))).toBe(true);
  });
  test("turning actions off again leaves the step unconfigured", () => {
    expect(setupStepConfigured("actions", status({ enableActions: true, pending: { enableActions: false } }))).toBe(
      false,
    );
  });
  test("login follows setupLoginConfigured", () => {
    expect(setupStepConfigured("login", status({ authProvider: "simple" }))).toBe(true);
  });
  test("auto-update counts once a schedule is saved", () => {
    const autoUpdate = {
      mode: "off" as const,
      time: "03:00",
      supported: true,
      image: "x:latest",
      currentVersion: "v1",
    };
    expect(setupStepConfigured("update", status({ autoUpdate }))).toBe(false);
    expect(setupStepConfigured("update", status({ autoUpdate: { ...autoUpdate, mode: "weekly" } }))).toBe(true);
    expect(setupStepConfigured("update", status())).toBe(false);
  });
  test("hosts count once an agent is listed", () => {
    expect(setupStepConfigured("hosts", status())).toBe(false);
    expect(setupStepConfigured("hosts", status({ agents: [] }))).toBe(false);
    const agent = { endpoint: "10.0.0.5:7007", address: "10.0.0.5:7007", locked: true };
    expect(setupStepConfigured("hosts", status({ agents: [agent] }))).toBe(true);
  });
  test("an added host is never pending, so it never asks for a restart", () => {
    const agent = { endpoint: "10.0.0.5:7007", address: "10.0.0.5:7007", locked: false };
    expect(setupHasPending(status({ agents: [agent] }))).toBe(false);
  });
  test("cloud and restart are never pre-marked", () => {
    expect(setupStepConfigured("cloud", status({ enableActions: true }))).toBe(false);
    expect(setupStepConfigured("restart", status({ authProvider: "simple" }))).toBe(false);
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
  test("stays closed when disabled by the operator, but still resumes", () => {
    expect(setupShouldAutoOpen({ ...base, disabled: true })).toBe(false);
    expect(setupShouldAutoOpen({ ...base, disabled: true, resume: "restart" })).toBe(true);
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

describe("agentComposeSnippet", () => {
  const cert = "-----BEGIN CERTIFICATE-----\nMIIB\nabcd\n-----END CERTIFICATE-----\n";
  const key = "-----BEGIN PRIVATE KEY-----\r\nMIIE\r\n-----END PRIVATE KEY-----";

  test("plain snippet has no environment", () => {
    const doc = parse(agentComposeSnippet("amir20/dozzle:latest"));
    expect(doc.services["dozzle-agent"].environment).toBeUndefined();
    expect(doc.services["dozzle-agent"].ports).toEqual(["7007:7007"]);
  });

  test("private snippet is valid YAML that carries both PEMs intact", () => {
    const env = parse(agentComposeSnippet("amir20/dozzle:latest", { cert, key })).services["dozzle-agent"].environment;
    expect(env.DOZZLE_CERT_PEM).toBe("-----BEGIN CERTIFICATE-----\nMIIB\nabcd\n-----END CERTIFICATE-----\n");
    expect(env.DOZZLE_KEY_PEM).toBe("-----BEGIN PRIVATE KEY-----\nMIIE\n-----END PRIVATE KEY-----\n");
  });
});

describe("agentImage", () => {
  test("follows the image the hub runs", () => {
    expect(agentImage("ghcr.io/amir20/dozzle:v12.0.0", "v12.0.0")).toBe("ghcr.io/amir20/dozzle:v12.0.0");
    expect(agentImage("amir20/dozzle:pr-5258", "pr-5258")).toBe("amir20/dozzle:pr-5258");
  });

  test("falls back to the hub's version, then latest", () => {
    expect(agentImage(undefined, "v12.0.0")).toBe("amir20/dozzle:v12.0.0");
    expect(agentImage("", "pr-5258")).toBe("amir20/dozzle:pr-5258");
    expect(agentImage(undefined, "pr-5258-75b67f0")).toBe("amir20/dozzle:pr-5258");
    expect(agentImage(undefined, "head")).toBe("amir20/dozzle:latest");
    expect(agentImage(undefined, "v12.0.0-beta.1")).toBe("amir20/dozzle:v12.0.0-beta.1");
  });

  test("never hands out an image another machine cannot pull", () => {
    expect(agentImage("sha256:3f2a9c1d4e5b6a7f8091a2b3c4d5e6f7", "v12.0.0")).toBe("amir20/dozzle:v12.0.0");
    expect(agentImage("3f2a9c1d4e5b", "v12.0.0")).toBe("amir20/dozzle:v12.0.0");
    expect(agentImage("dozzle:dev", "head")).toBe("amir20/dozzle:latest");
  });

  test("never hands out an image from the hub's own loopback registry", () => {
    expect(agentImage("localhost:5000/dozzle:dev", "v12.0.0")).toBe("amir20/dozzle:v12.0.0");
    expect(agentImage("localhost/dozzle:dev", "head")).toBe("amir20/dozzle:latest");
    expect(agentImage("127.0.0.1:5000/amir20/dozzle:dev", "v12.0.0")).toBe("amir20/dozzle:v12.0.0");
    expect(agentImage("[::1]:5000/dozzle:dev", "v12.0.0")).toBe("amir20/dozzle:v12.0.0");
    // A registry that merely starts with the word is still a real one.
    expect(agentImage("localhost.example.com/dozzle:v12.0.0", "v12.0.0")).toBe("localhost.example.com/dozzle:v12.0.0");
  });
});

describe("requests", () => {
  afterEach(() => vi.unstubAllGlobals());

  test("an error carries the server's X-Dozzle-Error code", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(
        async () =>
          new Response("could not connect to agent: refused", {
            status: 502,
            headers: { "X-Dozzle-Error": "unreachable" },
          }),
      ),
    );
    const err = await useSetup()
      .addAgent({ address: "a:7007" })
      .catch((e) => e);
    expect(err).toBeInstanceOf(SetupError);
    expect(err.status).toBe(502);
    expect(err.code).toBe("unreachable");
    expect(err.message).toBe("could not connect to agent: refused");
  });

  test("an error without the header has no code", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => new Response("Forbidden", { status: 403 })),
    );
    const err = await useSetup()
      .removeAgent("a:7007")
      .catch((e) => e);
    expect(err.status).toBe(403);
    expect(err.code).toBeUndefined();
  });

  // Opening the hosts dialog and adding a host each fire their own fetch; the older
  // one answering last must not put back the stale list.
  test("fetchStatus keeps only the newest answer", async () => {
    const resolvers: ((r: Response) => void)[] = [];
    vi.stubGlobal(
      "fetch",
      vi.fn(() => new Promise<Response>((resolve) => resolvers.push(resolve))),
    );
    const { fetchStatus, status: current, loading } = useSetup();
    const older = fetchStatus();
    const newer = fetchStatus();
    resolvers[1](new Response(JSON.stringify(status({ agents: [] }))));
    await newer;
    expect(current.value?.agents).toEqual([]);
    expect(loading.value).toBe(false);

    resolvers[0](
      new Response(JSON.stringify(status({ agents: [{ endpoint: "old:7007", address: "old:7007", locked: false }] }))),
    );
    await older;
    expect(current.value?.agents).toEqual([]);
  });
});
