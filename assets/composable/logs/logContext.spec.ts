/** @vitest-environment jsdom */
import { describe, expect, test, vi } from "vitest";
import { containerFixture } from "@/testing/containerFixture";
import { sharedNamePrefix } from "./logContext";

vi.mock("@/stores/config", () => ({ default: { base: "", hosts: [] }, withBase: (path: string) => path }));

const compose = (name: string, project = "myapp") =>
  containerFixture({ name, labels: { "com.docker.compose.project": project } });

describe("sharedNamePrefix", () => {
  test("drops the compose project every container shares", () => {
    expect(sharedNamePrefix([compose("myapp-web-1"), compose("myapp-worker-1")])).toBe("myapp-");
  });

  test("handles the underscore separator swarm and compose v1 use", () => {
    expect(sharedNamePrefix([compose("myapp_web.1.abc"), compose("myapp_db.1.def")])).toBe("myapp_");
  });

  test("keeps full names when projects differ", () => {
    expect(sharedNamePrefix([compose("myapp-web-1"), compose("other-web-1", "other")])).toBe("");
  });

  test("keeps full names when a container has no project", () => {
    expect(sharedNamePrefix([compose("myapp-web-1"), containerFixture({ name: "myapp-db" })])).toBe("");
  });

  test("keeps full names when a name does not carry the project", () => {
    expect(sharedNamePrefix([compose("myapp-web-1"), compose("custom-name")])).toBe("");
  });

  test("never strips a name down to nothing", () => {
    expect(sharedNamePrefix([compose("myapp-"), compose("myapp-web")])).toBe("");
  });

  test("does nothing for a single container", () => {
    expect(sharedNamePrefix([compose("myapp-web-1")])).toBe("");
  });
});
