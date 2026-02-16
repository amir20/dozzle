// Node v25+ ships a built-in localStorage that lacks the Web Storage API
// (getItem/setItem/removeItem). Replace it with a spec-compliant shim so
// libraries like @vue/devtools-kit work correctly in tests.
const store = new Map<string, string>();

globalThis.localStorage = {
  getItem: (key: string) => store.get(key) ?? null,
  setItem: (key: string, value: string) => store.set(key, String(value)),
  removeItem: (key: string) => store.delete(key),
  clear: () => store.clear(),
  get length() {
    return store.size;
  },
  key: (index: number) => [...store.keys()][index] ?? null,
} as Storage;
