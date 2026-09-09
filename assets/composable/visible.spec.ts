/** @vitest-environment jsdom */
import { afterEach, describe, expect, test, vi } from "vitest";
import { ref, shallowRef } from "vue";
import { ComplexLogEntry, SimpleLogEntry, type JSONObject } from "@/models/LogEntry";
import { useSearchFilter } from "./search";
import { useVisibleFilter, type VisibleKeysSource } from "./visible";

function entry(container: string, payload: JSONObject) {
  return new ComplexLogEntry(payload, container, 1, new Date(), "info", "stdout", JSON.stringify(payload));
}

afterEach(() => useSearchFilter().resetSearch());

describe("visible log fields", () => {
  test("tracks source-specific map mutations, replacements and new entries", () => {
    const key = ["secret"];
    const first = ref(new Map<string[], boolean>());
    const second = ref(new Map<string[], boolean>());
    const source = ref<VisibleKeysSource>((id) => (id === "a" ? first.value : second.value));
    const a = entry("a", { message: "a", secret: "one", nested: { detail: "shown" } });
    const b = entry("b", { message: "b", secret: "two" });
    const messages = shallowRef([a, b]);
    const filtered = useVisibleFilter(source).filteredPayload(messages);

    first.value = new Map([[key, false]]);
    expect(filtered.value[0].message).toEqual({ message: "a", "nested.detail": "shown" });
    expect(filtered.value[1].message).toEqual({ message: "b", secret: "two" });
    expect(a.unfilteredMessage).toEqual({ message: "a", secret: "one", nested: { detail: "shown" } });

    first.value.set(key, true);
    expect(filtered.value[0].message).toHaveProperty("secret", "one");
    first.value = new Map([[["secret"], false]]);
    messages.value = [...messages.value, entry("a", { secret: "three", newField: "visible" })];
    expect(filtered.value[2].message).toEqual({ newField: "visible" });

    source.value = new Map([[["message"], false]]);
    expect(filtered.value[0].message).not.toHaveProperty("message");
    expect(filtered.value[1].message).toEqual({ secret: "two" });
  });

  test("preserves the existing shared-map mode and non-JSON entries", () => {
    const fields = ref<VisibleKeysSource>(new Map<string[], boolean>([[["secret"], false]]));
    const plain = new SimpleLogEntry("plain", "a", 2, new Date(), "info", "stdout", "plain");
    const filtered = useVisibleFilter(fields).filteredPayload(
      shallowRef([entry("a", { secret: "hidden", message: "ok" }), plain]),
    );
    expect(filtered.value[0].message).toEqual({ message: "ok" });
    expect(filtered.value[1]).toBe(plain);

    fields.value = new Map([
      [["secret"], true],
      [["message"], true],
    ]);
    expect(Object.keys(filtered.value[0].message)).toEqual(["secret", "message"]);
  });

  test("applies search and inverse search after each source's field filtering", async () => {
    const search = useSearchFilter();
    search.searchQueryFilter.value = "hit";
    search.showSearch.value = true;
    await vi.waitFor(() => expect(search.isSearching.value).toBe(true));
    const fields = ref(new Map<string[], boolean>([[["match"], false]]));
    const source = ref<VisibleKeysSource>((id) => (id === "a" ? fields.value : new Map()));
    const messages = shallowRef([
      entry("a", { match: "<mark>hit</mark>", message: "a" }),
      entry("b", { match: "<mark>hit</mark>", message: "b" }),
    ]);
    const filtered = useVisibleFilter(source).filteredPayload(messages);
    expect(filtered.value.map((entry) => entry.containerID)).toEqual(["b"]);
    search.inverseFilter.value = true;
    expect(filtered.value.map((entry) => entry.containerID)).toEqual(["a"]);
    search.inverseFilter.value = false;
    fields.value = new Map();
    expect(filtered.value.map((entry) => entry.containerID)).toEqual(["a", "b"]);
  });
});
