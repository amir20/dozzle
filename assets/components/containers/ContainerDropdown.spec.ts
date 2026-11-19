/**
 * @vitest-environment jsdom
 */
import { mount } from "@vue/test-utils";
import { describe, expect, test, vi } from "vitest";
import ContainerDropdown from "./ContainerDropdown.vue";
import { containerFixture } from "@/testing/containerFixture";

vi.mock("@/stores/config", () => ({
  __esModule: true,
  default: { base: "", hosts: [] },
  withBase: (path: string) => path,
}));

function entries(containers: ReturnType<typeof containerFixture>[]) {
  const wrapper = mount(ContainerDropdown, {
    props: { containers },
    global: {
      stubs: {
        Popover: { template: "<div><slot /></div>" },
        RouterLink: { template: "<a><slot /></a>" },
        RelativeTime: true,
        "carbon:caret-down": true,
      },
    },
  });
  // Each entry as "name" or "name @ host".
  return wrapper.findAll("li").map((li) => {
    const host = li.find("span");
    const hostText = host.exists() ? host.text() : "";
    const name = li.text().replace(hostText, "").replace("running", "").trim();
    return hostText ? `${name} @ ${hostText}` : name;
  });
}

describe("ContainerDropdown", () => {
  // A global swarm service runs one same-named task per node (#5681).
  test("shows the host for a name running on several hosts, sorted by host", () => {
    expect(
      entries([
        containerFixture({ id: "b", name: "app", host: "node-02" }),
        containerFixture({ id: "c", name: "db", host: "node-01" }),
        containerFixture({ id: "a", name: "app", host: "node-01" }),
      ]),
    ).toEqual(["app @ node-01", "app @ node-02", "db"]);
  });

  test("leaves the host out when names do not collide across hosts", () => {
    expect(
      entries([
        containerFixture({ id: "a", name: "web", host: "node-01" }),
        containerFixture({ id: "b", name: "worker", host: "node-02" }),
      ]),
    ).toEqual(["web", "worker"]);
  });
});
