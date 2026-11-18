/**
 * @vitest-environment jsdom
 */
import { mount } from "@vue/test-utils";
import { afterEach, describe, expect, test, vi } from "vitest";
import { computed, nextTick, ref } from "vue";
import { createI18n } from "vue-i18n";
import ChatPane from "./ChatPane.vue";

const streaming = ref(false);

vi.mock("@/composable/cloud/cloudChat", () => ({
  useCloudChat: () => ({
    messages: ref([]),
    status: ref(""),
    activity: ref(""),
    streaming,
    focused: ref(undefined),
    ask: vi.fn(),
    clearFocus: vi.fn(),
  }),
}));

vi.mock("@/composable/cloud/cloudScope", () => ({
  useCloudScope: () => ({
    scopedView: computed(() => ({
      kind: "container",
      target: "abc",
      containers: [],
      hosts: [],
      visibleAt: "2024-01-01T00:00:00Z",
      historical: false,
      lines: [],
    })),
  }),
}));

const i18n = createI18n({ legacy: false, locale: "en", missingWarn: false, fallbackWarn: false, messages: { en: {} } });

let mounted: ReturnType<typeof mount> | undefined;

function mountPane() {
  mounted = mount(ChatPane, {
    attachTo: document.body,
    global: { plugins: [i18n], stubs: { ChatViewContext: true, ChatFocusedLine: true } },
  });
  return mounted;
}

afterEach(() => {
  mounted?.unmount();
  mounted = undefined;
  streaming.value = false;
  document.body.innerHTML = "";
});

describe("ChatPane", () => {
  test("puts focus back in the composer when an answer finishes", async () => {
    const wrapper = mountPane();
    const input = wrapper.find("input").element as HTMLInputElement;
    expect(document.activeElement).toBe(input);

    streaming.value = true;
    await nextTick();
    // A browser drops focus to the body when the focused input is disabled;
    // jsdom does not, so drop it the way a browser would end up.
    const tmp = document.createElement("button");
    document.body.appendChild(tmp);
    tmp.focus();
    tmp.remove();
    expect(document.activeElement).toBe(document.body);

    streaming.value = false;
    await nextTick();
    await nextTick();
    expect(document.activeElement).toBe(input);
  });

  test("leaves focus where the reader put it while the answer streamed", async () => {
    mountPane();
    const elsewhere = document.createElement("button");
    document.body.appendChild(elsewhere);

    streaming.value = true;
    await nextTick();
    elsewhere.focus();

    streaming.value = false;
    await nextTick();
    await nextTick();
    expect(document.activeElement).toBe(elsewhere);
  });
});
