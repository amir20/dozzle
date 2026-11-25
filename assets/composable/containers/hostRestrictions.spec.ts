/**
 * @vitest-environment jsdom
 */
import { describe, expect, test, vi } from "vitest";

vi.mock("@/stores/hosts", () => ({ useHosts: () => ({ hosts: { value: {} } }) }));
vi.stubGlobal("useHosts", () => ({ hosts: { value: {} } }));

const { isLocalHostname } = await import("./hostRestrictions");

describe("isLocalHostname", () => {
  test.each([
    "localhost",
    "127.0.0.1",
    "192.168.1.10",
    "[::1]",
    "fe80::1",
    "nas",
    "dozzle.lan",
    "NAS.LOCAL",
    "dozzle.home.arpa",
    "dozzle.internal.",
  ])("%s can open a shell", (h) => {
    expect(isLocalHostname(h)).toBe(true);
  });

  test.each(["rebind.example", "logs.example.com", "1.2.3.4.nip.io", "dozzle.lan.example"])("%s cannot", (h) => {
    expect(isLocalHostname(h)).toBe(false);
  });
});
