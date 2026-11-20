import { describe, expect, it } from "vitest";
import { nextTick } from "vue";
import { useListFilter } from "./listFilter";

type Group = { name?: string; items: string[] };

const groups: Group[] = [
  { name: "billing", items: ["api", "worker"] },
  { name: "search", items: ["indexer", "api-gateway"] },
  { name: undefined, items: ["redis"] },
];

const run = (f: ReturnType<typeof useListFilter>) =>
  f.filterGroups(
    groups,
    (g) => g.name,
    (g) => g.items,
    (i) => i,
    (g, items) => ({ ...g, items }),
  );

describe("useListFilter", () => {
  it("returns every group untouched without a query", () => {
    const f = useListFilter();
    expect(run(f)).toBe(groups);
  });

  it("keeps a whole group when its name matches, ignoring case", () => {
    const f = useListFilter();
    f.query.value = "  BILL ";
    expect(run(f)).toEqual([{ name: "billing", items: ["api", "worker"] }]);
  });

  it("keeps only matching children and drops empty groups", () => {
    const f = useListFilter();
    f.query.value = "api";
    expect(run(f)).toEqual([
      { name: "billing", items: ["api"] },
      { name: "search", items: ["api-gateway"] },
    ]);
  });

  it("never matches a group that has no name of its own", () => {
    const f = useListFilter();
    f.query.value = "undefined";
    expect(run(f)).toEqual([]);
  });

  it("opens every group while filtering and leaves persisted state alone", async () => {
    const f = useListFilter();
    const persisted = new Set(["billing"]);
    const persist = (key: string) => (open: boolean) => (open ? persisted.delete(key) : persisted.add(key));

    expect(f.isOpen("billing", !persisted.has("billing"))).toBe(false);

    f.query.value = "api";
    expect(f.isOpen("billing", !persisted.has("billing"))).toBe(true);

    f.setOpen("search", false, persist("search"));
    expect(f.isOpen("search", true)).toBe(false);
    expect(persisted.has("search")).toBe(false);

    f.query.value = "";
    await nextTick();
    expect(f.isOpen("search", true)).toBe(true);
    expect(f.isOpen("billing", !persisted.has("billing"))).toBe(false);

    f.setOpen("billing", true, persist("billing"));
    expect(persisted.has("billing")).toBe(false);
  });
});
