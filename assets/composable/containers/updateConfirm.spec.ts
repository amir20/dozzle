/**
 * @vitest-environment jsdom
 */
import { describe, expect, test } from "vitest";
import { cloudWatchable } from "./updateConfirm";

describe("cloudWatchable", () => {
  test("a container on a Docker host Dozzle talks to directly", () => {
    expect(cloudWatchable({ isSwarm: false }, "local")).toBe(true);
    expect(cloudWatchable({ isSwarm: false }, "remote")).toBe(true);
  });

  test("not on an agent, swarm or k8s host, nor before the host is known", () => {
    expect(cloudWatchable({ isSwarm: false }, "agent")).toBe(false);
    expect(cloudWatchable({ isSwarm: false }, "swarm")).toBe(false);
    expect(cloudWatchable({ isSwarm: false }, "k8s")).toBe(false);
    expect(cloudWatchable({ isSwarm: false }, undefined)).toBe(false);
  });

  test("not a swarm service, nor Dozzle itself", () => {
    expect(cloudWatchable({ isSwarm: true }, "local")).toBe(false);
    expect(cloudWatchable({ isSwarm: false }, "local", true)).toBe(false);
  });
});
