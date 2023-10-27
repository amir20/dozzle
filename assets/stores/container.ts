import { acceptHMRUpdate, defineStore } from "pinia";
import { Ref, UnwrapNestedRefs } from "vue";
import type { ContainerHealth, ContainerJson, ContainerStat } from "@/types/Container";
import { Container } from "@/models/Container";
import i18n from "@/modules/i18n";

const { showToast } = useToast();
// @ts-ignore
const { t } = i18n.global;

export const useContainerStore = defineStore("container", () => {
  const containers: Ref<Container[]> = ref([]);
  const activeContainerIds: Ref<string[]> = ref([]);
  let es: EventSource | null = null;
  const ready = ref(false);

  const allContainersById = computed(() =>
    containers.value.reduce(
      (acc, container) => {
        acc[container.id] = container;
        return acc;
      },
      {} as Record<string, Container>,
    ),
  );

  const visibleContainers = computed(() => {
    const filter = showAllContainers.value ? () => true : (c: Container) => c.state === "running";
    return containers.value.filter(filter);
  });

  const activeContainers = computed(() => activeContainerIds.value.map((id) => allContainersById.value[id]));

  function connect() {
    es?.close();
    ready.value = false;
    es = new EventSource(withBase("/api/events/stream"));
    es.addEventListener("error", (e) => {
      if (es?.readyState === EventSource.CLOSED) {
        showToast(
          {
            id: "events-stream",
            message: t("error.events-stream.message"),
            title: t("error.events-stream.title"),
            type: "error",
          },
          { once: true },
        );
      }
    });

    es.addEventListener("containers-changed", (e: Event) =>
      updateContainers(JSON.parse((e as MessageEvent).data) as ContainerJson[]),
    );
    es.addEventListener("container-stat", (e) => {
      const stat = JSON.parse((e as MessageEvent).data) as ContainerStat;
      const container = allContainersById.value[stat.id] as unknown as UnwrapNestedRefs<Container>;
      if (container) {
        const { id, ...rest } = stat;
        container.updateStat(rest);
      }
    });
    es.addEventListener("container-die", (e) => {
      const event = JSON.parse((e as MessageEvent).data) as { actorId: string };
      const container = allContainersById.value[event.actorId];
      if (container) {
        container.state = "dead";
      }
    });

    es.addEventListener("container-health", (e) => {
      const event = JSON.parse((e as MessageEvent).data) as { actorId: string; health: ContainerHealth };
      const container = allContainersById.value[event.actorId];
      if (container) {
        container.health = event.health;
      }
    });

    es.onopen = () => {
      if (containers.value.length > 0) {
        containers.value = [];
      }
    };

    watchOnce(containers, () => (ready.value = true));
  }

  connect();

  (async function () {
    try {
      await until(ready).toBe(true, { timeout: 8000, throwOnTimeout: true });
    } catch (e) {
      showToast(
        {
          id: "events-timeout",
          message: t("error.events-timeout.message"),
          title: t("error.events-timeout.title"),
          type: "error",
        },
        { once: true },
      );
    }
  })();

  const updateContainers = (containersPayload: ContainerJson[]) => {
    const existingContainers = containersPayload.filter((c) => allContainersById.value[c.id]);
    const newContainers = containersPayload.filter((c) => !allContainersById.value[c.id]);

    existingContainers.forEach((c) => {
      const existing = allContainersById.value[c.id];
      existing.status = c.status;
      existing.state = c.state;
      existing.health = c.health;
    });

    containers.value = [
      ...containers.value,
      ...newContainers.map((c) => {
        return new Container(
          c.id,
          new Date(c.created * 1000),
          c.image,
          c.name,
          c.command,
          c.host,
          c.status,
          c.state,
          c.health,
        );
      }),
    ];
  };

  const currentContainer = (id: Ref<string>) => computed(() => allContainersById.value[id.value]);
  const appendActiveContainer = ({ id }: { id: string }) => activeContainerIds.value.push(id);
  const removeActiveContainer = ({ id }: { id: string }) =>
    activeContainerIds.value.splice(activeContainerIds.value.indexOf(id), 1);

  return {
    containers,
    activeContainerIds,
    allContainersById,
    visibleContainers,
    activeContainers,
    currentContainer,
    appendActiveContainer,
    removeActiveContainer,
    ready,
  };
});

if (import.meta.hot) {
  import.meta.hot.accept(acceptHMRUpdate(useContainerStore, import.meta.hot));
}
