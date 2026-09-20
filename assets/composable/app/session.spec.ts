/**
 * @vitest-environment jsdom
 */
import { describe, expect, test, vi, beforeEach, afterEach } from "vitest";

const holder = vi.hoisted(() => ({
  config: { authProvider: "simple" } as Record<string, unknown>,
}));

// config is the default export of this module, and withBase lives alongside it.
vi.mock("@/stores/config", () => ({
  default: holder.config,
  withBase: (path: string) => path,
}));

const reload = vi.fn();

// jsdom's location is not writable, and reloading a page under test is not a
// thing that can happen anyway.
Object.defineProperty(window, "location", {
  configurable: true,
  value: { reload },
});

// session.ts keeps its "already answered" state at module scope on purpose, so
// each test needs its own copy of the module.
async function loadSession() {
  vi.resetModules();
  return (await import("./session")).checkSession;
}

const respondWith = (status: number) => {
  const fetchMock = vi.fn().mockResolvedValue({ status } as Response);
  vi.stubGlobal("fetch", fetchMock);
  return fetchMock;
};

describe("checkSession", () => {
  beforeEach(() => {
    holder.config.authProvider = "simple";
    reload.mockClear();
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  test("reloads once the session is gone", async () => {
    respondWith(401);
    const checkSession = await loadSession();

    await checkSession();

    expect(reload).toHaveBeenCalledTimes(1);
  });

  test("leaves a live session alone", async () => {
    respondWith(200);
    const checkSession = await loadSession();

    await checkSession();

    expect(reload).not.toHaveBeenCalled();
  });

  // A role the user does not have is not something signing in again can fix, and
  // reloading would bounce them between the app and the login page.
  test("ignores a forbidden response", async () => {
    respondWith(403);
    const checkSession = await loadSession();

    await checkSession();

    expect(reload).not.toHaveBeenCalled();
  });

  // The case this has to stay out of: the server is down or the phone is offline,
  // which is what the reconnect backoff is for.
  test("ignores a failed request", async () => {
    vi.stubGlobal("fetch", vi.fn().mockRejectedValue(new TypeError("Failed to fetch")));
    const checkSession = await loadSession();

    await checkSession();

    expect(reload).not.toHaveBeenCalled();
  });

  test("asks nothing when the instance runs without auth", async () => {
    holder.config.authProvider = "none";
    const fetchMock = respondWith(401);
    const checkSession = await loadSession();

    await checkSession();

    expect(fetchMock).not.toHaveBeenCalled();
    expect(reload).not.toHaveBeenCalled();
  });

  // Every open stream fails at once when a session expires, and between them they
  // should cost one request and one reload, not one each.
  test("collapses concurrent checks into a single request", async () => {
    const fetchMock = respondWith(401);
    const checkSession = await loadSession();

    await Promise.all([checkSession(), checkSession(), checkSession()]);

    expect(fetchMock).toHaveBeenCalledTimes(1);
    expect(reload).toHaveBeenCalledTimes(1);
  });

  test("stops asking after the session is known to be gone", async () => {
    const fetchMock = respondWith(401);
    const checkSession = await loadSession();

    await checkSession();
    await checkSession();

    expect(fetchMock).toHaveBeenCalledTimes(1);
    expect(reload).toHaveBeenCalledTimes(1);
  });
});
