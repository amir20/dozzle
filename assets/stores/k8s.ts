import { acceptHMRUpdate, defineStore } from "pinia";

import { Container, GroupedContainers } from "@/models/Container";

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
    public readonly containers: Container[],
  ) {}

  namespace?: K8sNamespace;

  get updatedAt() {
    return this.containers.map((c) => c.created).reduce((acc, date) => (date > acc ? date : acc), new Date(0));
  }
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
      const ownerGroups: Record<string, Container[]> = {};

      for (const container of containers) {
        const ownerKind = container.labels["owner.kind"];
        const ownerName = container.labels["owner.name"];

        if (ownerKind === undefined || ownerName === undefined) continue;
        const key = `${ownerKind}:${ownerName}`;
        ownerGroups[key] ||= [];
        ownerGroups[key].push(container);
      }

      const newOwners: K8sOwner[] = [];

      for (const [key, containers] of Object.entries(ownerGroups)) {
        const [kind, name] = key.split(":");
        newOwners.push(new K8sOwner(name, kind, containers));
      }

      if (newOwners.length === 0) continue;

      newNamespaces.push(
        new K8sNamespace(
          name,
          containers,
          newOwners.sort((a, b) => a.name.localeCompare(b.name)),
        ),
      );
    }
    return newNamespaces.sort((a, b) => a.name.localeCompare(b.name));
  });

  const owners = computed(() => {
    const ownerGroups: Record<string, Container[]> = {};

    for (const container of runningContainers.value) {
      const ownerKind = container.labels["owner.kind"];
      const ownerName = container.labels["owner.name"];
      const namespace = container.labels["namespace"];

      if (ownerKind === undefined || ownerName === undefined) continue;
      if (namespace) {
        // Skip containers that are already part of a namespace
        const hasNamespace = namespaces.value.some((ns) => ns.name === namespace);
        if (hasNamespace) continue;
      }

      const key = `${ownerKind}:${ownerName}`;
      ownerGroups[key] ||= [];
      ownerGroups[key].push(container);
    }

    const ownersWithNamespace = namespaces.value.flatMap((ns) => ns.owners);

    const ownersWithoutNamespace = Object.entries(ownerGroups).map(([key, containers]) => {
      const [kind, name] = key.split(":");
      return new K8sOwner(name, kind, containers);
    });

    return [...ownersWithNamespace, ...ownersWithoutNamespace].sort((a, b) => a.name.localeCompare(b.name));
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
