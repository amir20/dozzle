/**
 * @vitest-environment jsdom
 */
import { flushPromises, mount } from "@vue/test-utils";
import { afterEach, describe, expect, test, vi } from "vitest";
import { nextTick } from "vue";
import LoadMoreLogItem from "./LoadMoreLogItem.vue";
import { LoadMoreLogEntry } from "@/models/LogEntry";

// jsdom has no IntersectionObserver and no layout: the observer's callback is
// fired by hand, and positions come from the stubs below.
let observed: ((entries: { intersectionRatio: number }[]) => unknown) | undefined;
vi.mock("@vueuse/core", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@vueuse/core")>()),
  useIntersectionObserver: (_target: unknown, callback: typeof observed) => {
    observed = callback;
    return { stop: () => {} };
  },
}));

const rect = (top: number, height = 20) => ({ top, bottom: top + height, height }) as DOMRect;

// A scroller 500px tall holding the loader's row and the row under it, the one
// the reader sees at the top.
function setup(loader: (entry: LoadMoreLogEntry) => Promise<boolean | "start" | void>) {
  const scroller = document.createElement("main");
  scroller.setAttribute("data-scrolling", "");
  const list = document.createElement("ul");
  const loaderRow = document.createElement("li");
  const anchor = document.createElement("li");
  list.append(loaderRow, anchor);
  scroller.append(list);
  document.body.append(scroller);

  const layout = { scrollTop: 0, scrollHeight: 1000, anchorTop: 10, loaderTop: 0 };
  Object.defineProperty(scroller, "scrollTop", {
    get: () => layout.scrollTop,
    set: (v: number) => (layout.scrollTop = v),
  });
  Object.defineProperty(scroller, "scrollHeight", { get: () => layout.scrollHeight });
  scroller.getBoundingClientRect = () => rect(0, 500);
  anchor.getBoundingClientRect = () => rect(layout.anchorTop);

  const entry = new LoadMoreLogEntry(new Date(), loader);
  const wrapper = mount(LoadMoreLogItem, { props: { logEntry: entry }, attachTo: loaderRow });
  wrapper.element.getBoundingClientRect = () => rect(layout.loaderTop, 1);
  return { layout, wrapper, scroller, anchor };
}

async function fire() {
  await observed!([{ intersectionRatio: 1 }]);
  await flushPromises();
  await nextTick();
}

afterEach(() => {
  document.body.innerHTML = "";
  observed = undefined;
});

describe("LoadMoreLogItem scroll position", () => {
  test("moves by the lines added above the reader, not the ones streamed in below", async () => {
    const { layout } = setup(async () => {
      layout.scrollHeight += 300; // older lines above the reader
      layout.anchorTop += 300;
      layout.scrollHeight += 200; // live lines appended at the bottom meanwhile
      layout.loaderTop = -310; // the restore below takes the row out of view
    });
    await fire();
    expect(layout.scrollTop).toBe(300);
  });

  test("follows the reader's row when folding redraws it as a new element", async () => {
    const { layout, anchor } = setup(async () => {
      // Older repeats joined the top run: same line id, new element, 300px lower.
      const redrawn = document.createElement("li");
      redrawn.id = anchor.id;
      redrawn.getBoundingClientRect = () => rect(310);
      anchor.replaceWith(redrawn);
      layout.scrollHeight += 300;
      layout.loaderTop = -310;
    });
    anchor.id = "42";
    await fire();
    expect(layout.scrollTop).toBe(300);
  });

  test("stays put when the older lines merged in below the view", async () => {
    const { layout } = setup(async () => {
      layout.scrollHeight += 500;
    });
    await fire();
    expect(layout.scrollTop).toBe(0);
  });

  test("stays put when the load merged nothing or failed while lines streamed in", async () => {
    const quiet = setup(async () => {
      quiet.layout.scrollHeight += 500;
    });
    await fire();
    expect(quiet.layout.scrollTop).toBe(0);

    document.body.innerHTML = "";
    const failed = setup(async () => {
      failed.layout.scrollHeight += 500;
      throw new Error("offline");
    });
    vi.spyOn(console, "error").mockImplementation(() => {});
    await fire();
    expect(failed.layout.scrollTop).toBe(0);
  });

  test("leaves a window that replaced this one alone", async () => {
    const { layout, wrapper, anchor } = setup(async () => {
      layout.scrollHeight += 800;
      wrapper.unmount();
      anchor.remove();
    });
    await fire();
    expect(layout.scrollTop).toBe(0);
  });

  test("holds the place on the last page, where the start edge replaces this row", async () => {
    const { layout, wrapper } = setup(async () => {
      layout.scrollHeight += 300;
      layout.anchorTop += 300;
      wrapper.unmount();
      return true;
    });
    await fire();
    expect(layout.scrollTop).toBe(300);
  });
});

describe("LoadMoreLogItem while it stays in view", () => {
  test("loads again after a load that got somewhere left the row on screen", async () => {
    const loader = vi.fn<(entry: LoadMoreLogEntry) => Promise<boolean>>();
    loader
      .mockImplementationOnce(async () => {
        anchor.before(document.createElement("li")); // an older line, merged in below the view
        return true;
      })
      .mockResolvedValueOnce(false);
    const { anchor } = setup(loader);
    await fire();
    expect(loader).toHaveBeenCalledTimes(2);
  });

  test("loads again after a load that only found a container's start", async () => {
    const loader = vi.fn<(entry: LoadMoreLogEntry) => Promise<boolean | "start">>();
    loader.mockResolvedValueOnce("start").mockResolvedValueOnce(false);
    setup(loader);
    await fire();
    expect(loader).toHaveBeenCalledTimes(2);
  });

  test("stops when the older lines only folded into the top run", async () => {
    // Every page is more of the same line, so it adds to the ×N and no row.
    const loader = vi.fn(async () => true);
    setup(loader);
    await fire();
    expect(loader).toHaveBeenCalledTimes(1);
  });

  test("stops once a load got nowhere", async () => {
    const loader = vi.fn(async () => false);
    setup(loader);
    await fire();
    expect(loader).toHaveBeenCalledTimes(1);
  });

  test("does not load again once the row left the view", async () => {
    const loader = vi.fn(async () => {
      layout.loaderTop = -400;
      return true;
    });
    const { layout } = setup(loader);
    await fire();
    expect(loader).toHaveBeenCalledTimes(1);
  });
});
