/**
 * @vitest-environment jsdom
 */
import { mount } from "@vue/test-utils";
import { describe, expect, test, vi } from "vitest";
import { createI18n } from "vue-i18n";
import AlertRowActions from "./AlertRowActions.vue";

const jump = vi.hoisted(() => ({ jumpTo: vi.fn() }));
vi.mock("@/composable/logs/logJump", () => ({ useLogJump: () => jump }));
const rail = vi.hoisted(() => ({ closeRail: vi.fn() }));
vi.mock("@/composable/cloud/cloudRail", () => ({ useCloudRail: () => ({ closeRail: rail.closeRail }) }));

describe("<AlertRowActions />", () => {
  // The alerts tab lives in the modal cloud drawer, so a jump has to close it
  // or the stream scrolls behind it and nothing appears to happen.
  test("showing the lines closes the cloud drawer first", async () => {
    const moment = { containerId: "c", date: new Date(1e9), logId: 7 };
    const wrapper = mount(AlertRowActions, {
      props: { moment },
      global: { plugins: [createI18n({ legacy: false, locale: "en", missingWarn: false, fallbackWarn: false })] },
    });

    await wrapper.find("button").trigger("click");

    expect(rail.closeRail).toHaveBeenCalled();
    expect(jump.jumpTo).toHaveBeenCalledWith(moment);
  });
});
