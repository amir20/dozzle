/**
 * @vitest-environment jsdom
 */
import { mount } from "@vue/test-utils";
import { describe, expect, test, vi } from "vitest";
import { createI18n } from "vue-i18n";
import ContainerLink from "./ContainerLink.vue";
import { Container } from "@/models/Container";

vi.mock("@/stores/config", () => ({
  __esModule: true,
  default: { base: "", hosts: [{ name: "localhost", id: "localhost" }] },
  withBase: (path: string) => path,
}));

const i18n = createI18n({
  legacy: false,
  locale: "en",
  fallbackLocale: "en",
  missingWarn: false,
  fallbackWarn: false,
  messages: { en: { tooltip: { "open-container-url": "Open {url}" } } },
});

function makeContainer(labels: Record<string, string>) {
  return new Container(
    "id-1",
    new Date(),
    new Date(),
    new Date(),
    "image",
    "name",
    "command",
    "localhost",
    labels,
    "running",
    0,
    0,
    [],
  );
}

function mountLink(labels: Record<string, string>) {
  return mount(ContainerLink, {
    props: { container: makeContainer(labels) },
    global: { plugins: [i18n], stubs: { "mdi:open-in-new": true } },
  });
}

describe("ContainerLink", () => {
  test("renders nothing without the label", () => {
    expect(mountLink({}).find("a").exists()).toBe(false);
  });

  test("renders nothing for a non http scheme", () => {
    expect(mountLink({ "dev.dozzle.url": "javascript:alert(1)" }).find("a").exists()).toBe(false);
  });

  test("renders an anchor that opens in a new tab safely", () => {
    const link = mountLink({ "dev.dozzle.url": "https://grafana.example.com" }).get("a");
    expect(link.attributes("href")).toBe("https://grafana.example.com");
    expect(link.attributes("target")).toBe("_blank");
    expect(link.attributes("rel")).toBe("noopener noreferrer");
    expect(link.attributes("title")).toBe("Open https://grafana.example.com");
  });
});
