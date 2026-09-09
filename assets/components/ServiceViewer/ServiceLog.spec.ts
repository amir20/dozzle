/** @vitest-environment jsdom */
import { flushPromises, mount } from "@vue/test-utils";
import { createPinia, setActivePinia } from "pinia";
import { describe, expect, test, vi } from "vitest";
import { defineComponent, h, nextTick, provide, shallowRef, Suspense, type Ref } from "vue";
import { createI18n } from "vue-i18n";
import { Container } from "@/models/Container";
import { ComplexLogEntry } from "@/models/LogEntry";
import { drawerContext } from "@/composable/drawer";
import { useContainerStore } from "@/stores/container";
import ComplexLogItem from "@/components/LogViewer/ComplexLogItem.vue";
import LogDetails from "@/components/LogViewer/LogDetails.vue";
import ServiceLog from "./ServiceLog.vue";

vi.mock("@/stores/config", () => ({
  default: { base: "", mode: "swarm", hosts: [{ id: "host", name: "test host" }] },
  withBase: (path: string) => path,
}));

vi.mock("@/stores/container", async () => {
  const { defineStore } = await import("pinia");
  const { shallowRef, computed } = await import("vue");
  return {
    useContainerStore: defineStore("service-log-test-containers", () => {
      const containers = shallowRef<Container[]>([]);
      const findContainerById = (id: string) => containers.value.find((container) => container.id === id);
      const currentContainer = (id: Ref<string>) => computed(() => findContainerById(id.value));
      return { containers, findContainerById, currentContainer };
    }),
  };
});

function container(id: string, image: string, command: string) {
  return new Container(
    id,
    new Date(),
    new Date(),
    new Date(),
    image,
    id,
    command,
    "host",
    { "com.docker.swarm.service.name": "service" },
    "running",
    0,
    0,
    [],
  );
}

function entry(id: string, sequence: number) {
  const payload = { message: id, secret: "hidden candidate", nested: { debug: "details" } };
  return new ComplexLogEntry(payload, id, sequence, new Date(), "info", "stdout", JSON.stringify(payload));
}

describe("service log field settings", () => {
  test("applies drawer toggles per source and preserves shared replica settings", async () => {
    const pinia = createPinia();
    setActivePinia(pinia);
    const store = useContainerStore();
    store.containers = [
      container("a", "fixture/shared:1", "/app"),
      container("b", "fixture/shared:2", "/app"),
      container("c", "fixture/other:1", "/other"),
    ];
    const messages = shallowRef([entry("a", 1), entry("b", 2), entry("c", 3)]);
    const selected = shallowRef<ComplexLogEntry | null>(null);
    const Root = defineComponent({
      setup() {
        provide(drawerContext, (_component, props) => {
          selected.value = props.entry;
        });
        return () =>
          h("div", [
            h(ServiceLog, { name: "service" }),
            h(Suspense, null, {
              default: () =>
                selected.value ? h(LogDetails, { entry: selected.value, key: selected.value.id }) : h("div"),
            }),
          ]);
      },
    });
    const wrapper = mount(Root, {
      global: {
        plugins: [
          pinia,
          createI18n({ legacy: false, locale: "en", messages: { en: {} }, missingWarn: false, fallbackWarn: false }),
        ],
        stubs: {
          ScrollableView: { template: "<section><slot /></section>" },
          EventSource: defineComponent({
            props: ["streamSource", "entity"],
            setup(_props, { slots }) {
              return () => slots.default?.({ messages: messages.value });
            },
          }),
          LogList: defineComponent({
            props: ["messages"],
            setup(props) {
              return () =>
                h(
                  "section",
                  { "data-testid": "service-lines" },
                  props.messages.map((log: ComplexLogEntry) =>
                    h("article", { "data-container": log.containerID, key: log.id }, [
                      h(ComplexLogItem, { logEntry: log }),
                    ]),
                  ),
                );
            },
          }),
          LogItem: { template: "<div><slot /></div>" },
          LogLevel: true,
          DateTime: true,
          RelativeTime: true,
          JsonFormatted: true,
        },
      },
    });
    const line = (id: string) => wrapper.find(`[data-container="${id}"]`);
    const toggle = async (id: string, key: string) => {
      await line(id).find(".cursor-pointer").trigger("click");
      await flushPromises();
      await vi.waitFor(() => expect(wrapper.find("tbody .field-row").exists()).toBe(true));
      const row = wrapper.findAll("tbody .field-row").find((row) => row.find("td span").text() === key);
      expect(row, `field ${key}`).toBeDefined();
      await row!.find("input").setValue(false);
      await nextTick();
    };
    try {
      expect(line("c").text()).toContain("secret=");
      await toggle("c", "secret");
      expect(line("c").text()).not.toContain("secret=");
      expect(line("a").text()).toContain("secret=");
      expect(line("b").text()).toContain("secret=");

      await toggle("a", "nested.debug");
      expect(line("a").text()).not.toContain("nested.debug=");
      expect(line("b").text()).not.toContain("nested.debug=");
      expect(line("c").text()).toContain("nested.debug=");
      expect(line("c").text()).not.toContain("secret=");

      messages.value = [...messages.value, entry("b", 4)];
      await nextTick();
      expect(wrapper.findAll('[data-container="b"]')).toHaveLength(2);
      for (const row of wrapper.findAll('[data-container="b"]')) {
        expect(row.text()).not.toContain("nested.debug=");
      }

      messages.value = [...messages.value, entry("missing", 5)];
      await nextTick();
      expect(line("missing").text()).toContain("secret=");
      store.containers = [...store.containers, container("missing", "fixture/other:2", "/other")];
      await nextTick();
      expect(line("missing").text()).not.toContain("secret=");
    } finally {
      wrapper.unmount();
    }
  });
});
