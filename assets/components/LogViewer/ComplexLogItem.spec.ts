/**
 * @vitest-environment jsdom
 */
import { mount } from "@vue/test-utils";
import { describe, expect, test, vi } from "vitest";
import { ref } from "vue";
import ComplexLogItem from "./ComplexLogItem.vue";
import { ComplexLogEntry, type JSONObject } from "@/models/LogEntry";

vi.mock("@/stores/config", () => ({
  __esModule: true,
  default: { base: "", hosts: [{ name: "localhost", id: "localhost" }] },
  withBase: (path: string) => path,
}));

function mountItem(message: JSONObject, visibleKeys?: ReturnType<typeof ref<Map<string[], boolean>>>) {
  const entry = new ComplexLogEntry(message, "c1", 1, new Date(), "info", "stdout", "raw", visibleKeys as any);
  return mount(ComplexLogItem, {
    props: { logEntry: entry },
    global: {
      // LogItem pulls in stores/hosts; we only care about the payload rendering.
      stubs: { LogItem: { template: "<div><slot /></div>" }, LogLevel: true },
    },
  });
}

describe("<ComplexLogItem />", () => {
  test("renders key=value pairs", () => {
    const wrapper = mountItem({ foo: "bar", n: 1 });
    expect(wrapper.findAll("li")).toHaveLength(2);
    expect(wrapper.text()).toContain("foo=");
    expect(wrapper.text()).toContain("bar");
    expect(wrapper.text()).toContain("n=");
    expect(wrapper.text()).toContain("1");
  });

  test("renders null values as <null>", () => {
    const wrapper = mountItem({ x: null } as any);
    expect(wrapper.text()).toContain("<null>");
  });

  test("filters out undefined values", () => {
    const wrapper = mountItem({ a: undefined as any, b: 2 });
    expect(wrapper.findAll("li")).toHaveLength(1);
    expect(wrapper.text()).toContain("b=");
    expect(wrapper.text()).not.toContain("a=");
  });

  test("renders array values as a list", () => {
    const wrapper = mountItem({ tags: ["a", "b"] });
    expect(wrapper.find(".array").exists()).toBe(true);
    expect(wrapper.find(".array").text()).toContain("a");
    expect(wrapper.find(".array").text()).toContain("b");
  });

  test("shows a placeholder when every value is hidden", () => {
    const visibleKeys = ref(new Map<string[], boolean>([[["a"], false]]));
    const wrapper = mountItem({ a: 1 }, visibleKeys);
    expect(wrapper.text()).toContain("all values are hidden");
  });
});
