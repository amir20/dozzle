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

  it("gives a saved value between stops its own stop, without changing it", () => {
    const { wrapper, model } = mountField(3000);
    const range = wrapper.find("input[type=range]").element as HTMLInputElement;
    expect(range.max).toBe("4");
    expect(range.value).toBe("2");
    expect(wrapper.text()).toContain("50m");
    expect(model.value).toBe(3000);
  });

  it("keeps the saved value's stop after moving off it", async () => {
    const { wrapper, model } = mountField(3000);
    const range = wrapper.find("input[type=range]");
    await range.setValue(3);
    await nextTick();
    expect(model.value).toBe(3600);
    expect((range.element as HTMLInputElement).max).toBe("4");
    await range.setValue(2);
    await nextTick();
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
