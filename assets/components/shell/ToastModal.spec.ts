/**
 * @vitest-environment jsdom
 */
import { afterEach, describe, expect, test, vi } from "vitest";
import { mount } from "@vue/test-utils";
import ToastModal from "./ToastModal.vue";

describe("ToastModal secondary action", () => {
  test("renders primary, secondary and close together", async () => {
    const { useToast } = await import("@/composable/app/toast");
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
    const { useToast } = await import("@/composable/app/toast");
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
    const { useToast } = await import("@/composable/app/toast");
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
    const { useToast } = await import("@/composable/app/toast");
    const { showToast, toasts } = useToast();
    toasts.value = [];

    showToast({ message: "hello", type: "info" });
    const wrapper = mount(ToastModal);
    expect(wrapper.findAll("button")).toHaveLength(1);
    expect(wrapper.find("progress").exists()).toBe(false);
  });
});

describe("ToastModal countdown", () => {
  afterEach(() => vi.useRealTimers());

  test("draws a countdown for an expiring toast and closes it when time is up", async () => {
    vi.useFakeTimers();
    const { useToast } = await import("@/composable/app/toast");
    const { showToast, toasts } = useToast();
    toasts.value = [];

    showToast({ message: "copied", type: "info" }, { expire: 3000 });
    const wrapper = mount(ToastModal);
    const bar = wrapper.find("[data-testid=toast-countdown]");
    expect(bar.exists()).toBe(true);
    expect(bar.attributes("style")).toContain("3000ms");

    vi.advanceTimersByTime(3000);
    expect(toasts.value).toHaveLength(0);
  });

  test("runs a timed action when time is up", async () => {
    vi.useFakeTimers();
    const { useToast } = await import("@/composable/app/toast");
    const { showToast, toasts } = useToast();
    toasts.value = [];

    let ran = false;
    showToast(
      { message: "redirecting", type: "info", action: { label: "Cancel", handler: () => (ran = true) } },
      { timed: 4000 },
    );
    const wrapper = mount(ToastModal);
    expect(wrapper.find("[data-testid=toast-countdown]").exists()).toBe(true);

    vi.advanceTimersByTime(4000);
    expect(ran).toBe(true);
    expect(toasts.value).toHaveLength(0);
  });

  test("pressing a timed button closes the toast without acting", async () => {
    vi.useFakeTimers();
    const { useToast } = await import("@/composable/app/toast");
    const { showToast, toasts } = useToast();
    toasts.value = [];

    let ran = false;
    showToast(
      { message: "redirecting", type: "info", action: { label: "Cancel", handler: () => (ran = true) } },
      { timed: 4000 },
    );
    const wrapper = mount(ToastModal);
    await wrapper
      .findAll("button")
      .find((b) => b.text() === "Cancel")!
      .trigger("click");
    expect(toasts.value).toHaveLength(0);

    vi.advanceTimersByTime(4000);
    expect(ran).toBe(false);
  });

  test("a toast with no timer has no countdown", async () => {
    const { useToast } = await import("@/composable/app/toast");
    const { showToast, toasts } = useToast();
    toasts.value = [];

    showToast({ message: "hello", type: "info" });
    const wrapper = mount(ToastModal);
    expect(wrapper.find("[data-testid=toast-countdown]").exists()).toBe(false);
  });
});
