/**
 * @vitest-environment jsdom
 */
import { mount } from "@vue/test-utils";
import { describe, expect, test, vi } from "vitest";
import NetworkTopology from "./NetworkTopology.vue";
import DependencyGraph from "./DependencyGraph.vue";
import { Container } from "@/models/Container";

vi.mock("@/stores/config", () => ({
  __esModule: true,
  default: { base: "", hosts: [{ name: "localhost", id: "localhost" }] },
  withBase: (path: string) => path,
}));

vi.mock("vue-router", async (importOriginal) => ({
  ...(await importOriginal<object>()),
  useRouter: () => ({ push: vi.fn() }),
}));

function makeContainer(
  id: string,
  name: string,
  networks: string[],
  labels: Record<string, string> = {},
  state = "running",
): Container {
  return new Container(
    id,
    new Date(),
    new Date(),
    new Date(),
    `${name}:latest`,
    name,
    "cmd",
    "localhost",
    labels,
    state as never,
    0,
    0,
    [],
    undefined,
    undefined,
    false,
    [],
    {},
    networks,
  );
}

const containers = [
  makeContainer("a1", "web", ["app_default"], {
    "com.docker.compose.project": "app",
    "com.docker.compose.service": "web",
    "com.docker.compose.depends_on": "db:service_started:true",
  }),
  makeContainer("a2", "db", ["app_default"], {
    "com.docker.compose.project": "app",
    "com.docker.compose.service": "db",
  }),
  makeContainer("b1", "proxy", ["app_default", "edge"], {}, "exited"),
  makeContainer("c1", "standalone", []),
];

const globalStubs = { global: { stubs: { "router-link": true } } };

describe("<NetworkTopology />", () => {
  test("renders a node per container and links shared networks", () => {
    const wrapper = mount(NetworkTopology, { props: { containers }, ...globalStubs });
    expect(wrapper.findAll("circle").length).toBeGreaterThanOrEqual(containers.length);
    expect(wrapper.text()).toContain("4");
    expect(wrapper.findAll("line").length).toBeGreaterThan(0);
    expect(wrapper.text()).toContain("app_default");
  });
});

describe("<DependencyGraph />", () => {
  test("groups compose projects into cards and draws depends_on links", () => {
    const wrapper = mount(DependencyGraph, { props: { containers }, ...globalStubs });
    expect(wrapper.text()).toContain("app");
    expect(wrapper.findAll("rect").length).toBe(1);
    expect(wrapper.findAll("path").filter((p) => p.attributes("d")?.startsWith("M")).length).toBeGreaterThan(0);
  });
});
