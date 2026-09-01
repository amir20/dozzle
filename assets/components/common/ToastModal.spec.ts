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
    let updated = false;
    showToast({
      title: "Update available",
      message: "A newer image was published for nginx:latest.",
      type: "info",
      action: { label: "Update", handler: () => (updated = true) },
      secondaryAction: { label: "Dismiss this update", handler: () => (dismissed = true) },
    });

    const wrapper = mount(ToastModal);
    const buttons = wrapper.findAll("button");
    expect(buttons.map((b) => b.text()).filter(Boolean)).toEqual(["Dismiss this update", "Update"]);

    await buttons.find((b) => b.text() === "Dismiss this update")!.trigger("click");
    expect(dismissed).toBe(true);
    // Dismissing also closes the toast.
    expect(toasts.value).toHaveLength(0);
    expect(updated).toBe(false);
  });

  // Acting on a notice answers it, so it should not linger afterwards.
  test("closes the toast when the primary action is taken", async () => {
    const { useToast } = await import("@/composable/toast");
    const { showToast, toasts } = useToast();
    toasts.value = [];

    let updated = false;
    showToast({
      title: "Update available",
      message: "A newer image is available.",
      type: "info",
      action: { label: "Update", handler: () => (updated = true) },
      secondaryAction: { label: "Dismiss this update", handler: () => {} },
    });

    const wrapper = mount(ToastModal);
    await wrapper
      .findAll("button")
      .find((b) => b.text() === "Update")!
      .trigger("click");

    expect(updated).toBe(true);
    expect(toasts.value).toHaveLength(0);
  });

  test("renders a progress bar and updates it in place", async () => {
    const { useToast } = await import("@/composable/toast");
    const { showToast, updateToast, toasts } = useToast();
    toasts.value = [];

    showToast({ id: "job", title: "Update", message: "Pulling...", type: "info", progress: 25 });

    const wrapper = mount(ToastModal);
    expect(wrapper.find("progress").attributes("value")).toBe("25");
    expect(wrapper.text()).toContain("25%");

    updateToast("job", { progress: 80 });
    await wrapper.vm.$nextTick();
    expect(wrapper.find("progress").attributes("value")).toBe("80");

    // Work that cannot report progress hides the bar again.
    updateToast("job", { progress: undefined });
    await wrapper.vm.$nextTick();
    expect(wrapper.find("progress").exists()).toBe(false);
  });

  test("still renders a plain toast with only a close button", async () => {
    const { useToast } = await import("@/composable/toast");
    const { showToast, toasts } = useToast();
    toasts.value = [];

    showToast({ message: "hello", type: "info" });
    const wrapper = mount(ToastModal);
    expect(wrapper.findAll("button")).toHaveLength(1);
    expect(wrapper.find("progress").exists()).toBe(false);
  });
});
