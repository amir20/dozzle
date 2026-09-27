/**
 * @vitest-environment jsdom
 */
import { mount } from "@vue/test-utils";
import { describe, expect, test } from "vitest";
import { createI18n } from "vue-i18n";
import { defineComponent, h } from "vue";
import SetupHostsStep from "./SetupHostsStep.vue";
import type { SetupStatus } from "@/composable/setup/setup";

const AddHostPanelStub = defineComponent({ name: "AddHostPanel", props: ["status"], setup: () => () => h("div") });

const i18n = createI18n({ legacy: false, locale: "en", missingWarn: false, fallbackWarn: false, messages: { en: {} } });

function mountStep(agents: SetupStatus["agents"]) {
  const status = { agents } as SetupStatus;
  return mount(SetupHostsStep, {
    props: { status },
    global: { plugins: [i18n], stubs: { AddHostPanel: AddHostPanelStub } },
  });
}

describe("SetupHostsStep", () => {
  // Next with nothing added must not mark the step done, or the wizard ends on
  // "Dozzle is set up" for a step that was passed over.
  test("Next skips while no host is added", async () => {
    const vm = mountStep([]).vm as unknown as { next: () => Promise<string>; skipLabel?: string };
    expect(await vm.next()).toBe("skip");
    expect(vm.skipLabel).toBeTruthy();
  });

  test("Next advances once a host is added", async () => {
    const vm = mountStep([{ endpoint: "a:7007", address: "a:7007", locked: false }]).vm as unknown as {
      next: () => Promise<string>;
      skipLabel?: string;
    };
    expect(await vm.next()).toBe("advance");
    expect(vm.skipLabel).toBeUndefined();
  });
});
