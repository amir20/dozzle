/**
 * @vitest-environment jsdom
 */
import { mount } from "@vue/test-utils";
import { afterEach, describe, expect, test, vi } from "vitest";
import { ref } from "vue";
import { createI18n } from "vue-i18n";
import RailAccount from "./RailAccount.vue";
import type { CloudStatus } from "@/types/notifications";

vi.mock("@/stores/config", () => ({ __esModule: true, default: { cloudUrl: "https://cloud.example", hosts: [] } }));
const status = ref<CloudStatus | null>(null);
const error = ref<"auth" | "unavailable" | false>(false);
const fetchCloudStatus = vi.fn();
vi.mock("@/composable/cloud/cloudConfig", () => ({
  useCloudConfig: () => ({ cloudStatus: status, cloudStatusError: error, fetchCloudStatus }),
}));
vi.mock("@/composable/cloud/cloudSurface", () => ({ useCloudSurface: () => ({ canLink: ref(true) }) }));

const i18n = createI18n({
  legacy: false,
  locale: "en",
  missingWarn: false,
  fallbackWarn: false,
  messages: {
    en: {
      cloud: { account: "Account", plan: "Plan", dashboard: "Dashboard", error: "Connection error." },
      button: { retry: "Retry" },
    },
  },
});

const mountAccount = () =>
  mount(RailAccount, {
    global: { plugins: [i18n], stubs: { CloudUsage: { props: ["usage"], template: "<div data-usage />" } } },
  });

afterEach(() => {
  status.value = null;
  error.value = false;
  fetchCloudStatus.mockClear();
});

describe("<RailAccount />", () => {
  test("reads the status fresh and lays the account out as label and value rows", () => {
    status.value = {
      user: { email: "someone@example.com", name: "" },
      plan: { name: "pro", events_per_month: 1, retention_days: 30 },
      usage: { events_used: 1, events_limit: 10, period: "Nov" },
    };
    const wrapper = mountAccount();
    expect(fetchCloudStatus).toHaveBeenCalledOnce();
    expect(wrapper.find("[data-email]").text()).toBe("someone@example.com");
    expect(wrapper.text()).toContain("Plan");
    expect(wrapper.find(".status-pill").text()).toBe("pro");
    expect(wrapper.find("[data-usage]").exists()).toBe(true);
    expect(wrapper.find("a").attributes("href")).toBe("https://cloud.example");
  });

  test("a broken link says so in the same panel and offers to re-link", () => {
    error.value = "auth";
    const wrapper = mountAccount();
    expect(wrapper.find("[data-error]").text()).toContain("Connection error.");
    expect(wrapper.find("[data-error] .btn-primary").exists()).toBe(true);
    expect(wrapper.find("[data-email]").exists()).toBe(false);
  });

  test("while the status loads the panel keeps its shape", () => {
    const wrapper = mountAccount();
    expect(wrapper.find(".loading").exists()).toBe(true);
    expect(wrapper.find(".rounded-lg.border").exists()).toBe(true);
  });
});
