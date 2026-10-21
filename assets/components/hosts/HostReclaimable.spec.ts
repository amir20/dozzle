/**
 * @vitest-environment jsdom
 */
import { flushPromises, mount } from "@vue/test-utils";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { createI18n } from "vue-i18n";
import type { Host } from "@/stores/hosts";
import HostReclaimable from "./HostReclaimable.vue";

const config = vi.hoisted(() => ({ hosts: [], base: "", version: "test", enableActions: true }));
vi.mock("@/stores/config", () => ({ __esModule: true, default: config, withBase: (path: string) => path }));

const i18n = createI18n({
  legacy: false,
  locale: "en",
  missingWarn: false,
  fallbackWarn: false,
  messages: {
    en: {
      label: {
        reclaimable: "Reclaimable",
        "unused-images": "Unused images",
        "unused-volumes": "Unused volumes",
        "stopped-containers": "Stopped containers",
        "build-cache": "Build cache",
      },
      tooltip: { "reclaimable-source": "Same total as {command} reports." },
      button: { cancel: "Cancel" },
      reclaim: {
        "clean-up": "Clean up",
        "free-up": "Free up {size}",
        "confirm-title": "Free up {size} on {host}?",
        "cannot-undo": "This can't be undone.",
        "images-repull": "A stopped stack pulls its images again on its next start.",
        "volumes-warning": "1 volume holds data. | {count} volumes hold data.",
        "volumes-confirm": "I understand, delete the volumes",
        delete: "Delete",
        deleting: "Freeing up {size}…",
        freed: "Freed {size}",
        failed: "Clean up failed",
        "agent-too-old": "This agent can't clean up yet.",
        "enable-actions": "Enable container actions to clean up from here.",
        "preview-note": "Clean up checks again when it runs.",
        "preview-failed": "Couldn't list them.",
        "preview-agent-too-old": "Update this agent to see the list.",
      },
      hint: { "learn-more": "Learn more" },
    },
  },
});

const GB = 1024 ** 3;
const MB = 1024 ** 2;

const host: Host = {
  id: "host1",
  name: "prod-1",
  nCPU: 4,
  memTotal: 0,
  type: "local",
  endpoint: "local",
  available: true,
  dockerVersion: "",
  agentVersion: "",
  reclaimable: {
    images: 12,
    imagesSize: 3 * GB,
    volumes: 4,
    volumesSize: 2 * GB,
    containers: 3,
    containersSize: 512 * MB,
    buildCacheSize: 512 * MB,
  },
};

const fetchMock = vi.fn();
beforeEach(() => {
  config.enableActions = true;
  fetchMock.mockReset();
  vi.stubGlobal("fetch", fetchMock);
});
afterEach(() => vi.unstubAllGlobals());

function mountWith(extra: Partial<Host> = {}, diskPercent?: number) {
  return mount(HostReclaimable, { props: { host: { ...host, ...extra }, diskPercent }, global: { plugins: [i18n] } });
}

const button = (wrapper: ReturnType<typeof mountWith>, text: string) =>
  wrapper.findAll("button").find((b) => b.text().startsWith(text))!;

describe("<HostReclaimable />", () => {
  test("is read-only without actions", () => {
    config.enableActions = false;
    const wrapper = mountWith();
    expect(wrapper.find("input[type=checkbox]").exists()).toBe(false);
    expect(button(wrapper, "Clean up")).toBeUndefined();
    expect(wrapper.text()).toContain("Enable container actions");
    expect(wrapper.find("a[href='https://dozzle.dev/guide/actions']").exists()).toBe(true);
  });

  test("says nothing about actions on k8s, which can't clean up at all", () => {
    config.enableActions = false;
    expect(mountWith({ type: "k8s" }).text()).not.toContain("Enable container actions");
  });

  test("turns into a free up button once disk is tight", () => {
    expect(button(mountWith({}, 97), "Free up 6 GB")).toBeDefined();
    expect(button(mountWith({}, 40), "Reclaimable")).toBeDefined();
  });

  test("stays plain text on a tight disk without actions", () => {
    config.enableActions = false;
    expect(button(mountWith({}, 97), "Reclaimable")).toBeDefined();
  });

  test("hides the actions hint once actions are on", () => {
    expect(mountWith().text()).not.toContain("Enable container actions");
  });

  test("starts with everything but volumes picked", () => {
    const wrapper = mountWith();
    const checked = wrapper.findAll<HTMLInputElement>("input[type=checkbox]").map((i) => i.element.checked);
    expect(checked).toEqual([true, false, true, true]);
    // 3 GB of images, 512 MB of containers and 512 MB of build cache
    expect(button(wrapper, "Clean up").text()).toBe("Clean up 4 GB");
  });

  test("leaves out a kind with nothing in it", () => {
    const wrapper = mountWith({ reclaimable: { ...host.reclaimable!, buildCacheSize: 0 } });
    const boxes = wrapper.findAll<HTMLInputElement>("input[type=checkbox]");
    expect(boxes[3].element.disabled).toBe(true);
    expect(boxes[3].element.checked).toBe(false);
    expect(button(wrapper, "Clean up").text()).toBe("Clean up 3.5 GB");
  });

  test("confirms, deletes, and shows what it freed", async () => {
    fetchMock.mockResolvedValue({
      ok: true,
      json: async () => ({ spaceReclaimed: 3.9 * GB, reclaimable: { ...host.reclaimable!, images: 0, imagesSize: 0 } }),
    });
    const wrapper = mountWith();

    await button(wrapper, "Clean up").trigger("click");
    expect(wrapper.text()).toContain("Free up 4 GB on prod-1?");
    expect(wrapper.text()).toContain("This can't be undone.");
    expect(wrapper.text()).toContain("pulls its images again");
    expect(wrapper.findAll("li").map((li) => li.text())).toEqual([
      "Unused images123 GB",
      "Stopped containers3512 MB",
      "Build cache512 MB",
    ]);
    expect(fetchMock).not.toHaveBeenCalled();

    await button(wrapper, "Delete").trigger("click");
    await flushPromises();

    expect(fetchMock).toHaveBeenCalledWith("/api/hosts/host1/prune", expect.objectContaining({ method: "POST" }));
    expect(JSON.parse(fetchMock.mock.calls[0][1].body)).toEqual({ images: true, containers: true, buildCache: true });
    expect(wrapper.text()).toContain("Freed 3.9 GB");
  });

  test("volumes need their own confirmation", async () => {
    fetchMock.mockResolvedValue({ ok: true, json: async () => ({ spaceReclaimed: 0, reclaimable: null }) });
    const wrapper = mountWith();
    await wrapper.findAll("input[type=checkbox]")[1].setValue(true);
    await button(wrapper, "Clean up").trigger("click");

    expect(wrapper.text()).toContain("4 volumes hold data.");
    expect(button(wrapper, "Delete").attributes("disabled")).toBeDefined();

    await wrapper.findAll("input[type=checkbox]").at(-1)!.setValue(true);
    await button(wrapper, "Delete").trigger("click");
    await flushPromises();
    expect(JSON.parse(fetchMock.mock.calls[0][1].body)).toEqual({
      images: true,
      volumes: true,
      containers: true,
      buildCache: true,
    });
  });

  test("cancel goes back without deleting", async () => {
    const wrapper = mountWith();
    await button(wrapper, "Clean up").trigger("click");
    await button(wrapper, "Cancel").trigger("click");
    expect(button(wrapper, "Clean up")).toBeDefined();
    expect(fetchMock).not.toHaveBeenCalled();
  });

  test("tells an old agent apart from a failure", async () => {
    fetchMock.mockResolvedValue({ ok: false, status: 501, text: async () => "agent cannot prune" });
    const wrapper = mountWith({ type: "agent" });
    await button(wrapper, "Clean up").trigger("click");
    await button(wrapper, "Delete").trigger("click");
    await flushPromises();
    expect(wrapper.text()).toContain("Clean up failed");
    expect(wrapper.text()).toContain("This agent can't clean up yet.");
  });

  test("shows the engine's error", async () => {
    fetchMock.mockResolvedValue({ ok: false, status: 500, text: async () => "pruning volumes: device busy\n" });
    const wrapper = mountWith();
    await button(wrapper, "Clean up").trigger("click");
    await button(wrapper, "Delete").trigger("click");
    await flushPromises();
    expect(wrapper.text()).toContain("pruning volumes: device busy");
  });

  test("draws one bar for the total, solid where a clean up would free it", async () => {
    const wrapper = mountWith({ reclaimable: { ...host.reclaimable!, buildCacheSize: 0 } });
    const segments = () => wrapper.findAll("[style*=width]");
    // build cache is empty, so it has no segment
    // 3 GB, 2 GB and 512 MB of 5.5 GB
    expect(segments().map((s) => Math.round(parseFloat(s.attributes("style")!.slice(7))))).toEqual([55, 36, 9]);
    expect(segments().map((s) => s.classes("bg-base-content/50"))).toEqual([true, false, true]);

    await wrapper.findAll("input[type=checkbox]")[1].setValue(true);
    expect(segments().map((s) => s.classes("bg-base-content/50"))).toEqual([true, true, true]);
  });

  test("opens a row into what it would delete, fetching once", async () => {
    fetchMock.mockResolvedValue({
      ok: true,
      json: async () => ({
        images: [
          { id: "sha256:" + "a".repeat(64), tags: ["postgres:15", "postgres:latest"], size: 300 * MB },
          { id: "sha256:" + "b".repeat(64), tags: [], size: 50 * MB },
        ],
        volumes: [
          { name: "pg_data", size: 2 * GB, created: "2026-01-02T03:04:05Z" },
          { name: "c".repeat(64), size: -1, created: "0001-01-01T00:00:00Z" },
        ],
      }),
    });
    const wrapper = mountWith();
    await wrapper.findAll("input[type=checkbox]")[1].setValue(true);
    await button(wrapper, "Clean up").trigger("click");
    expect(fetchMock).not.toHaveBeenCalled();

    await button(wrapper, "Unused images").trigger("click");
    await flushPromises();
    expect(fetchMock).toHaveBeenCalledWith("/api/hosts/host1/reclaimable");
    expect(wrapper.text()).toContain("postgres:15+1300 MB");
    expect(wrapper.text()).toContain("<none> bbbbbbbbbbbb50 MB");
    expect(wrapper.text()).toContain("Clean up checks again when it runs.");

    await button(wrapper, "Unused volumes").trigger("click");
    await flushPromises();
    expect(fetchMock).toHaveBeenCalledTimes(1);
    expect(wrapper.text()).not.toContain("postgres:15");
    expect(wrapper.text()).toContain("pg_data");
    // an anonymous volume shows a short hash, and a driver that cannot size it a dash
    expect(wrapper.text()).toContain("cccccccccccc—");
  });

  test("tells an old agent it can't preview", async () => {
    fetchMock.mockResolvedValue({ ok: false, status: 501, text: async () => "" });
    const wrapper = mountWith({ type: "agent" });
    await button(wrapper, "Clean up").trigger("click");
    await button(wrapper, "Unused images").trigger("click");
    await flushPromises();
    expect(wrapper.text()).toContain("Update this agent to see the list.");
  });
});
