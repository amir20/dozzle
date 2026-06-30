import { describe, expect, test, vi } from "vitest";

import { Container } from "@/models/Container";
import { getK8sOwnerRefs, groupK8sOwners, ownerMembershipLabel } from "./k8s";

vi.mock("@/stores/config", () => ({
  __esModule: true,
  default: { base: "", hosts: [{ name: "localhost", id: "localhost" }] },
  withBase: (path: string) => path,
}));

function makeContainer(id: string, labels: Record<string, string>) {
  return new Container(
    id,
    new Date(),
    new Date(),
    new Date(),
    "image",
    id,
    "command",
    "localhost",
    labels,
    "running",
    0,
    0,
    [],
  );
}

describe("getK8sOwnerRefs", () => {
  test("parses indexed owner-chain labels", () => {
    const container = makeContainer("api", {
      namespace: "default",
      "@k8s.owner.count": "2",
      "@k8s.owner.0.type": "ReplicaSet",
      "@k8s.owner.0.kind": "ReplicaSet",
      "@k8s.owner.0.namespace": "default",
      "@k8s.owner.0.name": "api-6f88b977f4",
      "@k8s.owner.0.key": "ReplicaSet~default~api-6f88b977f4",
      "@k8s.owner.1.type": "Deployment",
      "@k8s.owner.1.kind": "Deployment",
      "@k8s.owner.1.namespace": "default",
      "@k8s.owner.1.name": "api",
      "@k8s.owner.1.key": "Deployment~default~api",
    });

    expect(getK8sOwnerRefs(container)).toEqual([
      {
        key: "ReplicaSet~default~api-6f88b977f4",
        label: ownerMembershipLabel("ReplicaSet~default~api-6f88b977f4"),
        kind: "ReplicaSet",
        name: "api-6f88b977f4",
        namespace: "default",
      },
      {
        key: "Deployment~default~api",
        label: ownerMembershipLabel("Deployment~default~api"),
        kind: "Deployment",
        name: "api",
        namespace: "default",
      },
    ]);
  });

  test("falls back to legacy immediate owner labels", () => {
    const container = makeContainer("api", {
      namespace: "default",
      "owner.kind": "ReplicaSet",
      "owner.name": "api-6f88b977f4",
    });

    expect(getK8sOwnerRefs(container)).toEqual([
      {
        key: "ReplicaSet~default~api-6f88b977f4",
        label: ownerMembershipLabel("ReplicaSet~default~api-6f88b977f4"),
        kind: "ReplicaSet",
        name: "api-6f88b977f4",
        namespace: "default",
      },
    ]);
  });
});

describe("groupK8sOwners", () => {
  test("groups a container under every owner in its chain", () => {
    const container = makeContainer("api", {
      namespace: "default",
      "@k8s.owner.count": "2",
      "@k8s.owner.0.type": "ReplicaSet",
      "@k8s.owner.0.namespace": "default",
      "@k8s.owner.0.name": "api-6f88b977f4",
      "@k8s.owner.0.key": "ReplicaSet~default~api-6f88b977f4",
      "@k8s.owner.1.type": "Deployment",
      "@k8s.owner.1.namespace": "default",
      "@k8s.owner.1.name": "api",
      "@k8s.owner.1.key": "Deployment~default~api",
    });

    const owners = groupK8sOwners([container]);

    expect(owners.map((owner) => owner.key).sort()).toEqual([
      "Deployment~default~api",
      "ReplicaSet~default~api-6f88b977f4",
    ]);
    expect(owners.every((owner) => owner.containers.length === 1)).toBe(true);
  });

  test("keeps same-name owners in different namespaces separate", () => {
    const owners = groupK8sOwners([
      makeContainer("default-api", {
        namespace: "default",
        "@k8s.owner.count": "1",
        "@k8s.owner.0.type": "Deployment",
        "@k8s.owner.0.namespace": "default",
        "@k8s.owner.0.name": "api",
        "@k8s.owner.0.key": "Deployment~default~api",
      }),
      makeContainer("prod-api", {
        namespace: "prod",
        "@k8s.owner.count": "1",
        "@k8s.owner.0.type": "Deployment",
        "@k8s.owner.0.namespace": "prod",
        "@k8s.owner.0.name": "api",
        "@k8s.owner.0.key": "Deployment~prod~api",
      }),
    ]);

    expect(owners.map((owner) => owner.key).sort()).toEqual(["Deployment~default~api", "Deployment~prod~api"]);
  });
});
