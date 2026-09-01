/**
 * @vitest-environment jsdom
 */
import { describe, expect, test } from "vitest";
import { mount } from "@vue/test-utils";
import ToastModal from "@/components/common/ToastModal.vue";

describe("ToastModal secondary action", () => {
  test("renders primary, secondary and close together", async () => {
    const { useToast } = await import("@/composable/toast");
    const { showToast, toasts } = useToast();
    toasts.value = [];

    let dismissed = false;
    showToast({
      title: "Update available",
      message: "A newer image was published for nginx:latest.",
      type: "info",
      action: { label: "Update", handler: () => {} },
      secondaryAction: { label: "Dismiss this update", handler: () => (dismissed = true) },
    });

    const wrapper = mount(ToastModal);
    const buttons = wrapper.findAll("button");
    expect(buttons.map((b) => b.text()).filter(Boolean)).toEqual(["Dismiss this update", "Update"]);

    await buttons.find((b) => b.text() === "Dismiss this update")!.trigger("click");
    expect(dismissed).toBe(true);
    // Dismissing also closes the toast.
    expect(toasts.value).toHaveLength(0);
  });

  test("still renders a plain toast with only a close button", async () => {
    const { useToast } = await import("@/composable/toast");
    const { showToast, toasts } = useToast();
    toasts.value = [];

    showToast({ message: "hello", type: "info" });
    const wrapper = mount(ToastModal);
    expect(wrapper.findAll("button")).toHaveLength(1);
  });
});
