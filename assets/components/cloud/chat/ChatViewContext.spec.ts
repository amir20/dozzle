/**
 * @vitest-environment jsdom
 */
import { mount } from "@vue/test-utils";
import { describe, expect, test } from "vitest";
import { createI18n } from "vue-i18n";
import type { ViewContext } from "@/composable/logs/viewContext";
import ChatViewContext from "./ChatViewContext.vue";

const i18n = createI18n({
  legacy: false,
  locale: "en",
  fallbackLocale: "en",
  missingWarn: false,
  fallbackWarn: false,
  messages: {
    en: {
      "cloud-chat": {
        evidence: "Sent with your question",
        "lines-in-view": "{n} lines in view",
        "n-containers": "{n} containers",
        historical: "historical",
      },
    },
  },
});

const view: ViewContext = {
  kind: "container",
  target: "abc",
  containers: [{ id: "abc", name: "doligence_postgres.1", host: "localhost" }],
  hosts: ["localhost"],
  visibleAt: "2024-01-01T00:00:00Z",
  historical: false,
  lines: [{ timestamp: "2024-01-01T00:00:00Z", message: "hello" }],
};

function mountWith(slot?: string) {
  return mount(ChatViewContext, {
    props: { view },
    slots: slot === undefined ? {} : { default: slot },
    global: { plugins: [i18n] },
  });
}

describe("ChatViewContext", () => {
  test("names the containers and the window in view", () => {
    expect(mountWith().text()).toContain("doligence_postgres.1");
    expect(mountWith().text()).toContain("1 lines in view");
  });

  test("draws no divider when no slot is passed", () => {
    expect(mountWith().find(".h-px").exists()).toBe(false);
  });

  // The composer always writes slot content, and it is a `v-if` on the focused
  // line. That renders a comment node when nothing is focused, which used to
  // still draw a hairline with nothing under it.
  test("draws no divider when the slot renders nothing", () => {
    expect(mountWith("<!-- v-if -->").find(".h-px").exists()).toBe(false);
  });

  test("draws a divider when the slot has content", () => {
    expect(mountWith("<div>this log line</div>").find(".h-px").exists()).toBe(true);
  });
});
