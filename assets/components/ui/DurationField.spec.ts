// @vitest-environment jsdom
import { describe, expect, it } from "vitest";
import { mount } from "@vue/test-utils";
import { nextTick, ref } from "vue";
import DurationField from "./DurationField.vue";

const stops = [0, 60, 3600, 21600];

function mountField(modelValue: number) {
  const model = ref(modelValue);
  const wrapper = mount(DurationField, {
    props: {
      label: "Cooldown",
      hint: (duration: string) => duration,
      stops,
      modelValue,
      "onUpdate:modelValue": (v: number) => {
        model.value = v;
        wrapper.setProps({ modelValue: v });
      },
    },
    global: { mocks: { $t: (key: string) => key } },
  });
  return { wrapper, model };
}

describe("DurationField with stops", () => {
  it("snaps the slider to the stop's value", async () => {
    const { wrapper, model } = mountField(60);
    await wrapper.find("input[type=range]").setValue(3);
    await nextTick();
    expect(model.value).toBe(21600);
  });

  it("shows a saved value between stops at the nearest one, without changing it", () => {
    const { wrapper, model } = mountField(3000);
    expect((wrapper.find("input[type=range]").element as HTMLInputElement).value).toBe("2");
    expect(model.value).toBe(3000);
  });

  it("has no seconds box", () => {
    expect(mountField(60).wrapper.find("input[type=number]").exists()).toBe(false);
  });

  it("states the value once, in the hint", () => {
    const { wrapper } = mountField(3600);
    expect(wrapper.text().replace("Cooldown", "")).toBe("1h");
  });
});
