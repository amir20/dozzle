// Node v25+ ships a built-in localStorage that lacks the Web Storage API
// (getItem/setItem/removeItem). Replace it with a spec-compliant shim so
// libraries like @vue/devtools-kit work correctly in tests. jsdom already
// provides a compliant one behind a getter-only property, so leave it alone.
function currentLocalStorage() {
  try {
    return globalThis.localStorage;
  } catch {
    return undefined;
  }
}

if (typeof currentLocalStorage()?.getItem !== "function") {
  const store = new Map<string, string>();

  Object.defineProperty(globalThis, "localStorage", {
    configurable: true,
    writable: true,
    value: {
      getItem: (key: string) => store.get(key) ?? null,
      setItem: (key: string, value: string) => store.set(key, String(value)),
      removeItem: (key: string) => store.delete(key),
      clear: () => store.clear(),
      get length() {
        return store.size;
      },
      key: (index: number) => [...store.keys()][index] ?? null,
    } as Storage,
  });
}
