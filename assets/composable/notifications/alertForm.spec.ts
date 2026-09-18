import { describe, expect, test, vi } from "vitest";

import { alertTargetFor } from "./alertForm";

vi.mock("@/stores/config", () => ({
  __esModule: true,
  default: { base: "", hosts: [{ name: "localhost", id: "localhost" }] },
  withBase: (path: string) => path,
}));

describe("alertTargetFor", () => {
  test("keys a k8s pod on its workload so the next CronJob run matches", () => {
    const target = alertTargetFor({
      name: "hello-29824580-tznrr/hello",
      labels: {
        "@k8s.namespace": "default",
        "@k8s.workload.kind": "CronJob",
        "@k8s.workload.name": "hello",
      },
    });

    expect(target).toEqual({
      name: "CronJob/hello",
      expression:
        'labels["@k8s.namespace"] == "default" && labels["@k8s.workload.kind"] == "CronJob" && labels["@k8s.workload.name"] == "hello" && name endsWith "/hello"',
    });
  });

  test("keys a Swarm task on its service", () => {
    const target = alertTargetFor({
      name: "web_api.1.x7f3k2",
      labels: { "com.docker.swarm.service.name": "web_api" },
    });

    expect(target).toEqual({ name: "web_api", expression: 'labels["com.docker.swarm.service.name"] == "web_api"' });
  });

  test("keeps the container name for everything else", () => {
    expect(alertTargetFor({ name: "nginx", labels: {} })).toEqual({
      name: "nginx",
      expression: 'name contains "nginx"',
    });
  });
});
