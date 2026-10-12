/**
 * @vitest-environment jsdom
 */
import { flushPromises, mount } from "@vue/test-utils";
import { describe, expect, test, vi } from "vitest";
import { createMemoryHistory, createRouter, RouterView } from "vue-router";
import { setupLayouts } from "virtual:generated-layouts";

vi.mock("@/stores/config", () => ({
  __esModule: true,
  default: { base: "", hosts: [] },
  withBase: (path: string) => path,
}));

const Page = { template: "<p data-testid='page'>page body</p>" };

async function render(path: string) {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: setupLayouts([
      { path: "/", name: "/", component: Page },
      { path: "/login", name: "/login", component: Page, meta: { layout: "splash" } },
    ]),
  });
  router.push(path);
  await router.isReady();
  const wrapper = mount(RouterView, {
    global: {
      plugins: [router],
      // The real default layout is the whole app shell; only where it puts the page matters here.
      stubs: { default: { template: "<div data-testid='default-layout'><router-view /></div>" } },
    },
  });
  await flushPromises();
  return wrapper;
}

// Every page in assets/pages renders inside a layout from assets/layouts: the
// default one unless the page names another in its route meta.
describe("setupLayouts", () => {
  test("renders a page inside the default layout", async () => {
    const wrapper = await render("/");
    expect(wrapper.find("[data-testid='default-layout'] [data-testid='page']").exists()).toBe(true);
  });

  test("renders a page inside the layout its route meta names", async () => {
    const wrapper = await render("/login");
    expect(wrapper.find("[data-testid='default-layout']").exists()).toBe(false);
    expect(wrapper.find(".hero .hero-content [data-testid='page']").exists()).toBe(true);
  });
});
