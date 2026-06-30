import { acceptHMRUpdate, defineStore } from "pinia";

import { Container, GroupedContainers } from "@/models/Container";

export type K8sOwnerRef = {
  key: string;
  label: string;
  kind: string;
  name: string;
  namespace?: string;
};

export class K8sNamespace {
  constructor(
    public readonly name: string,
    public readonly containers: Container[],
    public readonly owners: K8sOwner[],
  ) {
    for (const owner of owners) {
      owner.namespace = this;
    }
  }

  get updatedAt() {
    return this.containers.map((c) => c.created).reduce((acc, date) => (date > acc ? date : acc), new Date(0));
  }
}

export class K8sOwner {
  constructor(
    public readonly name: string,
    public readonly kind: string,
    public readonly namespaceName: string | undefined,
    public readonly key: string,
    public readonly label: string,
    public readonly containers: Container[],
  ) {}

  namespace?: K8sNamespace;

  get updatedAt() {
    return this.containers.map((c) => c.created).reduce((acc, date) => (date > acc ? date : acc), new Date(0));
  }
}

export function ownerMembershipLabel(key: string) {
  const bytes = new TextEncoder().encode(key);
  let binary = "";
  for (const byte of bytes) {
    binary += String.fromCharCode(byte);
  }
  return `@k8s.owner.key.${btoa(binary).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "")}`;
}

export function getK8sOwnerRefs(container: Container): K8sOwnerRef[] {
  const count = Number(container.labels["@k8s.owner.count"] ?? container.labels["k8s.owner.count"] ?? 0);
  if (count > 0) {
    const owners: K8sOwnerRef[] = [];
    for (let i = 0; i < count; i++) {
      const syntheticPrefix = `@k8s.owner.${i}.`;
      const legacyPrefix = `k8s.owner.${i}.`;
      const labelValue = (key: string) =>
        container.labels[`${syntheticPrefix}${key}`] ?? container.labels[`${legacyPrefix}${key}`];
      const kind = labelValue("type") ?? labelValue("kind");
      const name = labelValue("name");
      const namespace = labelValue("namespace");
      if (!kind || !name) continue;
      const key = labelValue("key") ?? `${kind}~${namespace ?? ""}~${name}`;
      owners.push({ key, label: ownerMembershipLabel(key), kind, name, namespace });
    }
    return owners;
  }

  // "~" matches the backend owner-key delimiter: URL-safe and invalid in Kubernetes names.
  const kind = container.labels["owner.kind"];
  const name = container.labels["owner.name"];
  if (!kind || !name) return [];

  const namespace = container.labels["namespace"];
  const key = container.labels["owner.key"] ?? `${kind}~${namespace ?? ""}~${name}`;
  return [{ key, label: ownerMembershipLabel(key), kind, name, namespace }];
}

export function groupK8sOwners(containers: Container[]) {
  const ownerGroups: Record<string, { owner: K8sOwnerRef; containers: Container[] }> = {};
  for (const container of containers) {
    for (const owner of getK8sOwnerRefs(container)) {
      ownerGroups[owner.key] ||= { owner, containers: [] };
      ownerGroups[owner.key].containers.push(container);
    }
  }
  return Object.values(ownerGroups).map(({ owner, containers }) => {
    return new K8sOwner(owner.name, owner.kind, owner.namespace, owner.key, owner.label, containers);
  });
}

export const useK8sStore = defineStore("k8s", () => {
  const containerStore = useContainerStore();
  const { containers } = storeToRefs(containerStore) as unknown as { containers: Ref<Container[]> };

  const runningContainers = computed(() => containers.value.filter((c) => c.state === "running"));

  const namespaces = computed(() => {
    const namespacedContainers: Record<string, Container[]> = {};
    for (const container of runningContainers.value) {
      const namespace = container.labels["namespace"];
      if (namespace === undefined) continue;
      namespacedContainers[namespace] ||= [];
      namespacedContainers[namespace].push(container);
    }

    const newNamespaces: K8sNamespace[] = [];

    for (const [name, containers] of Object.entries(namespacedContainers)) {
      const newOwners = groupK8sOwners(containers);

      if (newOwners.length === 0) continue;

      newNamespaces.push(
        new K8sNamespace(
          name,
          containers,
          newOwners.sort((a, b) => a.key.localeCompare(b.key)),
        ),
      );
    }
    return newNamespaces.sort((a, b) => a.name.localeCompare(b.name));
  });

  const owners = computed(() => {
    const containersWithoutNamespace: Container[] = [];

    for (const container of runningContainers.value) {
      const namespace = container.labels["namespace"];
      if (namespace) {
        // Skip containers that are already part of a namespace
        const hasNamespace = namespaces.value.some((ns) => ns.name === namespace);
        if (hasNamespace) continue;
      }
      containersWithoutNamespace.push(container);
    }

    const ownersWithNamespace = namespaces.value.flatMap((ns) => ns.owners);

    const ownersWithoutNamespace = groupK8sOwners(containersWithoutNamespace);

    return [...ownersWithNamespace, ...ownersWithoutNamespace].sort((a, b) => a.key.localeCompare(b.key));
  });

  const customGroups = computed(() => {
    const grouped: Record<string, Container[]> = {};

    for (const container of runningContainers.value) {
      const group = container.customGroup;
      if (group === undefined) continue;
      grouped[group] ||= [];
      grouped[group].push(container);
    }

    return Object.entries(grouped).map(([name, containers]) => new GroupedContainers(name, containers));
  });

  return {
    namespaces,
    owners,
    customGroups,
  };
});

if (import.meta.hot) {
  import.meta.hot.accept(acceptHMRUpdate(useK8sStore, import.meta.hot));
}
