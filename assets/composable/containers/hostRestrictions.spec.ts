/**
 * @vitest-environment jsdom
 */
import { describe, expect, test, vi } from "vitest";

vi.mock("@/stores/hosts", () => ({ useHosts: () => ({ hosts: { value: {} } }) }));
vi.stubGlobal("useHosts", () => ({ hosts: { value: {} } }));

const { isIpOrLocalhost } = await import("./hostRestrictions");

describe("isIpOrLocalhost", () => {
  test.each(["localhost", "127.0.0.1", "192.168.1.10", "[::1]", "::1", "fe80::1"])("%s can open a shell", (h) => {
    expect(isIpOrLocalhost(h)).toBe(true);
  });

  test.each(["dozzle.lan", "rebind.example", "logs.example.com", "1.2.3.4.nip.io"])("%s cannot", (h) => {
    expect(isIpOrLocalhost(h)).toBe(false);
  });
});
